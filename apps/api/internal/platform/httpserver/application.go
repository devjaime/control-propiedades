package httpserver

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"net/mail"
	"path/filepath"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	appauth "github.com/devjaime/control-propiedades/apps/api/internal/auth"
)

type ObjectStorage interface {
	Put(context.Context, string, io.Reader, int64, string) error
	Get(context.Context, string) (io.ReadCloser, error)
	Delete(context.Context, string) error
}

type DirectObjectStorage interface {
	PresignPut(context.Context, string, string, string, time.Duration) (string, map[string]string, error)
	InitiateMultipart(context.Context, string, string, string, int32, time.Duration) (string, map[int32]string, error)
	CompleteMultipart(context.Context, string, string, map[int32]string) error
	AbortMultipart(context.Context, string, string) error
	PresignGet(context.Context, string, string, string, string, time.Duration) (string, error)
	Stat(context.Context, string) (int64, string, map[string]string, error)
}

type Options struct {
	Environment    string
	WebOrigin      string
	PublicWebURL   string
	SessionTTL     time.Duration
	CookieName     string
	CookieSecure   bool
	MaxUploadBytes int64
}

type application struct {
	logger  *slog.Logger
	db      *pgxpool.Pool
	storage ObjectStorage
	options Options
}

type contextKey string

const principalKey contextKey = "principal"

type principal struct {
	UserID         string   `json:"user_id"`
	OrganizationID string   `json:"organization_id"`
	Name           string   `json:"name"`
	Email          string   `json:"email"`
	Organization   string   `json:"organization"`
	Role           string   `json:"role"`
	AuthKind       string   `json:"-"`
	Scopes         []string `json:"-"`
}

var allowedPropertyTypes = []string{"house", "apartment", "land", "commercial", "other"}
var allowedPropertyUses = []string{"primary_residence", "rental", "second_home", "vacant", "renovation", "for_sale", "acquisition"}
var allowedDocumentTypes = []string{
	"deed", "domain_registration", "valuation", "lease", "annex", "inventory",
	"bank_receipt", "rent_receipt", "mortgage", "property_tax", "insurance",
	"common_expense", "utility", "quote", "invoice", "warranty", "photo",
	"inspection", "certificate", "other",
}

func NewApplication(logger *slog.Logger, db *pgxpool.Pool, storage ObjectStorage, options Options) http.Handler {
	if strings.TrimSpace(options.PublicWebURL) == "" {
		options.PublicWebURL = options.WebOrigin
	}
	app := &application{logger: logger, db: db, storage: storage, options: options}
	router := newRouter(logger, db)
	router.Route("/api/v1", func(router chi.Router) {
		router.Use(app.sameOrigin)
		router.Get("/public/receipts/{token}", app.verifyRentReceipt)
		router.Post("/public/tenant-portals/{token}/view", app.viewTenantPortal)
		router.Post("/public/tenant-portals/{token}/documents/{documentID}/download", app.downloadTenantPortalDocument)
		router.Route("/auth", func(router chi.Router) {
			router.Post("/register", app.register)
			router.Post("/login", app.login)
		})
		router.Group(func(router chi.Router) {
			router.Use(app.authenticate)
			router.Get("/me", app.me)
			router.Post("/auth/logout", app.logout)
			router.Get("/agent-tokens", app.listAgentTokens)
			router.Post("/agent-tokens", app.createAgentToken)
			router.Post("/agent-tokens/{tokenID}/revoke", app.revokeAgentToken)
			router.Get("/properties", app.listProperties)
			router.Post("/properties", app.createProperty)
			router.Get("/properties/{propertyID}", app.getProperty)
			router.Get("/properties/{propertyID}/incidents", app.listPropertyIncidents)
			router.Post("/properties/{propertyID}/incidents", app.createIncident)
			router.Post("/incidents/{incidentID}/updates", app.addIncidentUpdate)
			router.Post("/incidents/{incidentID}/documents", app.linkIncidentDocument)
			router.Patch("/incident-tasks/{taskID}", app.updateIncidentTask)
			router.Get("/properties/{propertyID}/tenant-portal", app.getTenantPortalLink)
			router.Post("/properties/{propertyID}/tenant-portal", app.rotateTenantPortalLink)
			router.Get("/properties/{propertyID}/rental-ledger", app.getRentalLedger)
			router.Post("/properties/{propertyID}/lease", app.createLease)
			router.Get("/properties/{propertyID}/lease-contract-workflow", app.getLeaseContractWorkflow)
			router.Post("/properties/{propertyID}/tenant-applications", app.createTenantApplication)
			router.Post("/tenant-applications/{applicationID}/documents", app.linkTenantApplicationDocument)
			router.Post("/tenant-applications/{applicationID}/contract-drafts", app.createLeaseContractDraft)
			router.Post("/lease-contract-drafts/{draftID}/review", app.reviewLeaseContractDraft)
			router.Post("/lease-contract-drafts/{draftID}/signed-document", app.attachSignedLeaseContract)
			router.Patch("/leases/{leaseID}/terms", app.amendLeaseTerms)
			router.Patch("/leases/{leaseID}/end", app.endLease)
			router.Post("/properties/{propertyID}/charges", app.createCharge)
			router.Post("/properties/{propertyID}/payments", app.createPayment)
			router.Post("/properties/{propertyID}/payments/from-document", app.createPaymentFromDocument)
			router.Post("/properties/{propertyID}/observations", app.createRentalObservation)
			router.Post("/properties/{propertyID}/maintenance-schedules", app.createMaintenanceSchedule)
			router.Post("/maintenance-schedules/{maintenanceID}/complete", app.completeMaintenanceSchedule)
			router.Post("/payments/{paymentID}/confirm", app.confirmPayment)
			router.Patch("/notifications/{notificationID}/read", app.readNotification)
			router.Post("/properties/{propertyID}/documents", app.uploadDocument)
			router.Post("/properties/{propertyID}/document-uploads", app.initiateDocumentUpload)
			router.Post("/document-uploads/{uploadID}/complete", app.completeDocumentUpload)
			router.Get("/documents", app.listDocuments)
			router.Get("/documents/{documentID}", app.getDocument)
			router.Patch("/documents/{documentID}", app.updateDocument)
			router.Get("/documents/{documentID}/download", app.downloadDocument)
		})
		router.Route("/agent", func(router chi.Router) {
			router.Use(app.authenticateAgent)
			router.Get("/properties", app.agentListProperties)
			router.Get("/properties/{propertyID}/overview", app.agentPropertyOverview)
			router.Get("/properties/{propertyID}/open-actions", app.agentOpenActions)
			router.Post("/properties/{propertyID}/document-uploads", app.agentInitiateDocumentUpload)
			router.Post("/document-uploads/{uploadID}/complete", app.agentCompleteDocumentUpload)
			router.Post("/properties/{propertyID}/payments/from-document", app.agentCreatePaymentFromDocument)
			router.Post("/payments/{paymentID}/confirm", app.agentConfirmPayment)
			router.Post("/incidents/{incidentID}/updates", app.agentAddIncidentUpdate)
			router.Patch("/incident-tasks/{taskID}", app.agentUpdateIncidentTask)
		})
	})
	return router
}

func (app *application) sameOrigin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
			origin := strings.TrimSuffix(r.Header.Get("Origin"), "/")
			if origin != "" && origin != strings.TrimSuffix(app.options.WebOrigin, "/") {
				writeError(w, http.StatusForbidden, "invalid_origin", "El origen de la solicitud no está permitido.")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (app *application) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(app.options.CookieName)
		if err != nil || cookie.Value == "" {
			writeError(w, http.StatusUnauthorized, "authentication_required", "Debes iniciar sesión.")
			return
		}

		var person principal
		err = app.db.QueryRow(r.Context(), `
			SELECT u.id, s.organization_id, u.name, u.email, o.name, m.role
			FROM sessions s
			JOIN users u ON u.id = s.user_id
			JOIN organizations o ON o.id = s.organization_id
			JOIN memberships m ON m.user_id = u.id AND m.organization_id = s.organization_id
			WHERE s.token_hash = $1 AND s.revoked_at IS NULL AND s.expires_at > now()
			  AND u.status = 'active' AND o.status = 'active' AND m.status = 'active'
		`, appauth.HashSessionToken(cookie.Value)).Scan(
			&person.UserID, &person.OrganizationID, &person.Name, &person.Email,
			&person.Organization, &person.Role,
		)
		if err != nil {
			app.clearSessionCookie(w)
			writeError(w, http.StatusUnauthorized, "invalid_session", "La sesión expiró o fue revocada.")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), principalKey, person)))
	})
}

func (app *application) authenticateAgent(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorization := strings.TrimSpace(r.Header.Get("Authorization"))
		if !strings.HasPrefix(authorization, "Bearer cp_") {
			writeError(w, http.StatusUnauthorized, "agent_authentication_required", "Se requiere un token de agente válido.")
			return
		}
		token := strings.TrimPrefix(authorization, "Bearer ")
		var person principal
		err := app.db.QueryRow(r.Context(), `
			SELECT u.id, token.organization_id, u.name, u.email, organization.name,
			       membership.role, token.scopes
			FROM api_tokens token
			JOIN users u ON u.id = token.user_id AND u.status = 'active'
			JOIN organizations organization ON organization.id = token.organization_id AND organization.status = 'active'
			JOIN memberships membership ON membership.user_id = token.user_id
			  AND membership.organization_id = token.organization_id AND membership.status = 'active'
			WHERE token.token_hash = $1 AND token.status = 'active'
			  AND (token.expires_at IS NULL OR token.expires_at > now())
		`, appauth.HashSessionToken(token)).Scan(
			&person.UserID, &person.OrganizationID, &person.Name, &person.Email,
			&person.Organization, &person.Role, &person.Scopes,
		)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid_agent_token", "El token de agente no es válido, expiró o fue revocado.")
			return
		}
		person.AuthKind = "agent"
		_, _ = app.db.Exec(r.Context(), `UPDATE api_tokens SET last_used_at = now() WHERE token_hash = $1`, appauth.HashSessionToken(token))
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), principalKey, person)))
	})
}

func requireScope(w http.ResponseWriter, person principal, scope string) bool {
	if person.AuthKind != "agent" || !slices.Contains(person.Scopes, scope) {
		writeError(w, http.StatusForbidden, "agent_scope_required", "El token no autoriza esta operación.")
		return false
	}
	return true
}

func (app *application) register(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name             string `json:"name"`
		Email            string `json:"email"`
		Password         string `json:"password"`
		OrganizationName string `json:"organization_name"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	input.OrganizationName = strings.TrimSpace(input.OrganizationName)
	if len(input.Name) < 2 || len(input.OrganizationName) < 2 || !validEmail(input.Email) {
		writeError(w, http.StatusUnprocessableEntity, "invalid_registration", "Revisa el nombre, correo y nombre de la organización.")
		return
	}
	passwordHash, err := appauth.HashPassword(input.Password)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid_password", err.Error())
		return
	}
	token, tokenHash, err := appauth.NewSessionToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "session_error", "No fue posible crear la sesión.")
		return
	}

	var person principal
	err = pgx.BeginFunc(r.Context(), app.db, func(tx pgx.Tx) error {
		slug := slugify(input.OrganizationName) + "-" + randomHex(4)
		if err := tx.QueryRow(r.Context(), `
			INSERT INTO organizations (name, slug)
			VALUES ($1, $2)
			RETURNING id, name
		`, input.OrganizationName, slug).Scan(&person.OrganizationID, &person.Organization); err != nil {
			return err
		}
		if err := tx.QueryRow(r.Context(), `
			INSERT INTO users (name, email, password_hash, status)
			VALUES ($1, $2, $3, 'active')
			RETURNING id, name, email
		`, input.Name, input.Email, passwordHash).Scan(&person.UserID, &person.Name, &person.Email); err != nil {
			return err
		}
		person.Role = "organization_admin"
		if _, err := tx.Exec(r.Context(), `
			INSERT INTO memberships (organization_id, user_id, role, status)
			VALUES ($1, $2, $3, 'active')
		`, person.OrganizationID, person.UserID, person.Role); err != nil {
			return err
		}
		if _, err := tx.Exec(r.Context(), `
			INSERT INTO sessions (organization_id, user_id, token_hash, expires_at)
			VALUES ($1, $2, $3, $4)
		`, person.OrganizationID, person.UserID, tokenHash, time.Now().Add(app.options.SessionTTL)); err != nil {
			return err
		}
		return audit(r.Context(), tx, person, "account.registered", "organization", person.OrganizationID, nil, map[string]any{"name": person.Organization})
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			writeError(w, http.StatusConflict, "email_already_registered", "Ya existe una cuenta con ese correo.")
			return
		}
		app.logger.Error("register account", "error", err)
		writeError(w, http.StatusInternalServerError, "registration_failed", "No fue posible crear la cuenta.")
		return
	}
	app.setSessionCookie(w, token)
	writeJSON(w, http.StatusCreated, map[string]any{"user": person})
}

func (app *application) login(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	var passwordHash string
	var person principal
	err := app.db.QueryRow(r.Context(), `
		SELECT u.id, u.name, u.email, u.password_hash, o.id, o.name, m.role
		FROM users u
		JOIN memberships m ON m.user_id = u.id AND m.status = 'active'
		JOIN organizations o ON o.id = m.organization_id AND o.status = 'active'
		WHERE lower(u.email) = $1 AND u.status = 'active'
		ORDER BY m.created_at
		LIMIT 1
	`, input.Email).Scan(
		&person.UserID, &person.Name, &person.Email, &passwordHash,
		&person.OrganizationID, &person.Organization, &person.Role,
	)
	if err != nil || !appauth.ComparePassword(passwordHash, input.Password) {
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "Correo o contraseña incorrectos.")
		return
	}
	token, tokenHash, err := appauth.NewSessionToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "session_error", "No fue posible crear la sesión.")
		return
	}
	_, err = app.db.Exec(r.Context(), `
		INSERT INTO sessions (organization_id, user_id, token_hash, expires_at)
		VALUES ($1, $2, $3, $4)
	`, person.OrganizationID, person.UserID, tokenHash, time.Now().Add(app.options.SessionTTL))
	if err != nil {
		app.logger.Error("create login session", "error", err)
		writeError(w, http.StatusInternalServerError, "session_error", "No fue posible crear la sesión.")
		return
	}
	_, _ = app.db.Exec(r.Context(), "UPDATE users SET last_login_at = now() WHERE id = $1", person.UserID)
	app.setSessionCookie(w, token)
	writeJSON(w, http.StatusOK, map[string]any{"user": person})
}

func (app *application) logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(app.options.CookieName); err == nil {
		_, _ = app.db.Exec(r.Context(), "UPDATE sessions SET revoked_at = now() WHERE token_hash = $1", appauth.HashSessionToken(cookie.Value))
	}
	app.clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (app *application) me(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"user": currentPrincipal(r)})
}

type propertyResponse struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	PropertyType string    `json:"property_type"`
	Usage        string    `json:"usage"`
	Status       string    `json:"status"`
	AddressLine  string    `json:"address_line"`
	Commune      string    `json:"commune"`
	Region       string    `json:"region"`
	CreatedAt    time.Time `json:"created_at"`
}

func (app *application) createProperty(w http.ResponseWriter, r *http.Request) {
	person := currentPrincipal(r)
	var input struct {
		Name         string `json:"name"`
		PropertyType string `json:"property_type"`
		Usage        string `json:"usage"`
		AddressLine  string `json:"address_line"`
		Commune      string `json:"commune"`
		Region       string `json:"region"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	input.AddressLine = strings.TrimSpace(input.AddressLine)
	input.Commune = strings.TrimSpace(input.Commune)
	input.Region = strings.TrimSpace(input.Region)
	if input.Name == "" || input.AddressLine == "" || input.Commune == "" || input.Region == "" ||
		!slices.Contains(allowedPropertyTypes, input.PropertyType) || !slices.Contains(allowedPropertyUses, input.Usage) {
		writeError(w, http.StatusUnprocessableEntity, "invalid_property", "Completa los datos obligatorios de la propiedad.")
		return
	}

	var property propertyResponse
	err := pgx.BeginFunc(r.Context(), app.db, func(tx pgx.Tx) error {
		err := tx.QueryRow(r.Context(), `
			INSERT INTO properties (
				organization_id, name, slug, property_type, usage, status,
				address_line, commune, region, created_by, updated_by
			) VALUES ($1, $2, $3, $4, $5, 'active', $6, $7, $8, $9, $9)
			RETURNING id, name, property_type, usage, status, address_line, commune, region, created_at
		`, person.OrganizationID, input.Name, slugify(input.Name)+"-"+randomHex(3), input.PropertyType,
			input.Usage, input.AddressLine, input.Commune, input.Region, person.UserID).Scan(
			&property.ID, &property.Name, &property.PropertyType, &property.Usage, &property.Status,
			&property.AddressLine, &property.Commune, &property.Region, &property.CreatedAt,
		)
		if err != nil {
			return err
		}
		return audit(r.Context(), tx, person, "property.created", "property", property.ID, nil, property)
	})
	if err != nil {
		app.logger.Error("create property", "error", err)
		writeError(w, http.StatusInternalServerError, "property_create_failed", "No fue posible registrar la propiedad.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"property": property})
}

func (app *application) listProperties(w http.ResponseWriter, r *http.Request) {
	person := currentPrincipal(r)
	rows, err := app.db.Query(r.Context(), `
		SELECT id, name, property_type, usage, status, address_line, commune, region, created_at
		FROM properties
		WHERE organization_id = $1 AND status <> 'archived'
		ORDER BY created_at DESC
	`, person.OrganizationID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "properties_read_failed", "No fue posible consultar las propiedades.")
		return
	}
	defer rows.Close()
	properties := make([]propertyResponse, 0)
	for rows.Next() {
		var property propertyResponse
		if err := rows.Scan(&property.ID, &property.Name, &property.PropertyType, &property.Usage, &property.Status,
			&property.AddressLine, &property.Commune, &property.Region, &property.CreatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "properties_read_failed", "No fue posible consultar las propiedades.")
			return
		}
		properties = append(properties, property)
	}
	writeJSON(w, http.StatusOK, map[string]any{"properties": properties})
}

func (app *application) getProperty(w http.ResponseWriter, r *http.Request) {
	person := currentPrincipal(r)
	var property propertyResponse
	err := app.db.QueryRow(r.Context(), `
		SELECT id, name, property_type, usage, status, address_line, commune, region, created_at
		FROM properties WHERE id = $1 AND organization_id = $2 AND status <> 'archived'
	`, chi.URLParam(r, "propertyID"), person.OrganizationID).Scan(
		&property.ID, &property.Name, &property.PropertyType, &property.Usage, &property.Status,
		&property.AddressLine, &property.Commune, &property.Region, &property.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "property_not_found", "La propiedad no existe.")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "property_read_failed", "No fue posible consultar la propiedad.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"property": property})
}

type documentResponse struct {
	ID              string     `json:"id"`
	PropertyID      string     `json:"property_id"`
	PropertyName    string     `json:"property_name"`
	DocumentType    string     `json:"document_type"`
	DisplayName     string     `json:"display_name"`
	OriginalName    string     `json:"original_name"`
	MimeType        string     `json:"mime_type"`
	SizeBytes       int64      `json:"size_bytes"`
	SHA256          string     `json:"sha256"`
	Tags            []string   `json:"tags"`
	Status          string     `json:"status"`
	DocumentDate    *string    `json:"document_date"`
	Period          *string    `json:"period"`
	Issuer          *string    `json:"issuer"`
	AmountMinor     *int64     `json:"amount_minor"`
	CurrencyCode    *string    `json:"currency_code"`
	Confidentiality string     `json:"confidentiality"`
	RejectionReason *string    `json:"rejection_reason"`
	VerifiedAt      *time.Time `json:"verified_at"`
	VerifiedBy      *string    `json:"verified_by"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	Version         int64      `json:"version"`
}

func (app *application) uploadDocument(w http.ResponseWriter, r *http.Request) {
	person := currentPrincipal(r)
	propertyID := chi.URLParam(r, "propertyID")
	var propertyName string
	if err := app.db.QueryRow(r.Context(), `
		SELECT name FROM properties WHERE id = $1 AND organization_id = $2 AND status <> 'archived'
	`, propertyID, person.OrganizationID).Scan(&propertyName); err != nil {
		writeError(w, http.StatusNotFound, "property_not_found", "La propiedad no existe.")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, app.options.MaxUploadBytes+1024*1024)
	if err := r.ParseMultipartForm(app.options.MaxUploadBytes + 1024*1024); err != nil {
		writeError(w, http.StatusRequestEntityTooLarge, "upload_too_large", uploadTooLargeMessage(app.options.MaxUploadBytes))
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "file_required", "Selecciona un archivo para subir.")
		return
	}
	defer file.Close()
	if header.Size <= 0 || header.Size > app.options.MaxUploadBytes {
		writeError(w, http.StatusRequestEntityTooLarge, "upload_too_large", uploadTooLargeMessage(app.options.MaxUploadBytes))
		return
	}
	documentType := strings.TrimSpace(r.FormValue("document_type"))
	if !slices.Contains(allowedDocumentTypes, documentType) {
		writeError(w, http.StatusUnprocessableEntity, "invalid_document_type", "Selecciona un tipo de documento válido.")
		return
	}
	originalName := filepath.Base(strings.TrimSpace(header.Filename))
	displayName := strings.TrimSpace(r.FormValue("display_name"))
	if displayName == "" {
		displayName = originalName
	}
	if len(displayName) > 240 || len(originalName) > 255 {
		writeError(w, http.StatusUnprocessableEntity, "invalid_document_name", "El nombre del documento es demasiado largo.")
		return
	}

	buffer := make([]byte, 512)
	readBytes, err := file.Read(buffer)
	if err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "file_read_failed", "No fue posible leer el archivo.")
		return
	}
	contentType := http.DetectContentType(buffer[:readBytes])
	allowedMIME := map[string]string{
		"application/pdf": ".pdf", "image/jpeg": ".jpg", "image/png": ".png", "image/webp": ".webp",
	}
	extension, allowed := allowedMIME[contentType]
	if !allowed {
		writeError(w, http.StatusUnsupportedMediaType, "unsupported_file", "Solo se permiten PDF, JPG, PNG y WebP.")
		return
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		writeError(w, http.StatusBadRequest, "file_read_failed", "No fue posible procesar el archivo.")
		return
	}
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		writeError(w, http.StatusBadRequest, "file_read_failed", "No fue posible procesar el archivo.")
		return
	}
	hash := hex.EncodeToString(hasher.Sum(nil))
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		writeError(w, http.StatusBadRequest, "file_read_failed", "No fue posible procesar el archivo.")
		return
	}

	var duplicateID string
	err = app.db.QueryRow(r.Context(), `
		SELECT id FROM documents WHERE organization_id = $1 AND sha256 = $2 LIMIT 1
	`, person.OrganizationID, hash).Scan(&duplicateID)
	if err == nil {
		writeJSON(w, http.StatusConflict, map[string]any{
			"error": map[string]any{"code": "duplicate_document", "message": "Este archivo ya fue cargado.", "existing_document_id": duplicateID},
		})
		return
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusInternalServerError, "duplicate_check_failed", "No fue posible verificar el documento.")
		return
	}

	storageKey := fmt.Sprintf("organizations/%s/properties/%s/documents/%s%s", person.OrganizationID, propertyID, randomHex(16), extension)
	if err := app.storage.Put(r.Context(), storageKey, file, header.Size, contentType); err != nil {
		app.logger.Error("store document", "error", err)
		writeError(w, http.StatusBadGateway, "storage_failed", "No fue posible guardar el archivo.")
		return
	}

	tags := normalizeTags(r.FormValue("tags"))
	var document documentResponse
	err = app.db.QueryRow(r.Context(), `
		INSERT INTO documents (
			organization_id, property_id, document_type, display_name, original_name,
			storage_key, mime_type, size_bytes, sha256, tags, uploaded_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, property_id, document_type, display_name, original_name, mime_type,
		          size_bytes, sha256, tags, status, confidentiality, created_at, updated_at, version
	`, person.OrganizationID, propertyID, documentType, displayName, originalName, storageKey,
		contentType, header.Size, hash, tags, person.UserID).Scan(
		&document.ID, &document.PropertyID, &document.DocumentType, &document.DisplayName,
		&document.OriginalName, &document.MimeType, &document.SizeBytes, &document.SHA256,
		&document.Tags, &document.Status, &document.Confidentiality, &document.CreatedAt,
		&document.UpdatedAt, &document.Version,
	)
	if err != nil {
		_ = app.storage.Delete(r.Context(), storageKey)
		app.logger.Error("save document metadata", "error", err)
		writeError(w, http.StatusInternalServerError, "document_save_failed", "No fue posible registrar el documento.")
		return
	}
	document.PropertyName = propertyName
	_, _ = app.db.Exec(r.Context(), `
		INSERT INTO audit_events (organization_id, actor_type, actor_id, action, entity_type, entity_id, new_state)
		VALUES ($1, 'user', $2, 'document.uploaded', 'document', $3, $4)
	`, person.OrganizationID, person.UserID, document.ID, map[string]any{"name": document.DisplayName, "sha256": document.SHA256})
	writeJSON(w, http.StatusCreated, map[string]any{"document": document})
}

func uploadTooLargeMessage(maxBytes int64) string {
	return fmt.Sprintf("El archivo supera el límite permitido de %d MB.", maxBytes/(1024*1024))
}

func (app *application) listDocuments(w http.ResponseWriter, r *http.Request) {
	person := currentPrincipal(r)
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	propertyID := strings.TrimSpace(r.URL.Query().Get("property_id"))
	documentType := strings.TrimSpace(r.URL.Query().Get("type"))
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if status != "" && !slices.Contains([]string{"uploaded", "pending_review", "verified", "rejected", "duplicate", "archived"}, status) {
		writeError(w, http.StatusBadRequest, "invalid_document_status", "El estado documental no es válido.")
		return
	}
	rows, err := app.db.Query(r.Context(), `
		SELECT d.id, d.property_id, p.name, d.document_type, d.display_name, d.original_name,
		       d.mime_type, d.size_bytes, d.sha256, d.tags, d.status,
		       to_char(d.document_date, 'YYYY-MM-DD'), d.period, d.issuer, d.amount_minor,
		       d.currency_code, d.confidentiality, d.rejection_reason, d.verified_at,
		       d.verified_by, d.created_at, d.updated_at, d.version
		FROM documents d
		JOIN properties p ON p.id = d.property_id AND p.organization_id = d.organization_id
		WHERE d.organization_id = $1
		  AND ($2 = '' OR d.property_id = NULLIF($2, '')::uuid)
		  AND ($3 = '' OR d.document_type = $3)
		  AND ($5 = '' OR d.status = $5)
		  AND ($4 = '' OR d.display_name ILIKE '%' || $4 || '%'
		       OR d.original_name ILIKE '%' || $4 || '%'
		       OR EXISTS (SELECT 1 FROM unnest(d.tags) tag WHERE tag ILIKE '%' || $4 || '%'))
		  AND ($5 = 'archived' OR d.status <> 'archived')
		ORDER BY d.created_at DESC
		LIMIT 100
	`, person.OrganizationID, propertyID, documentType, query, status)
	if err != nil {
		writeError(w, http.StatusBadRequest, "documents_read_failed", "No fue posible realizar la búsqueda.")
		return
	}
	defer rows.Close()
	documents := make([]documentResponse, 0)
	for rows.Next() {
		var document documentResponse
		if err := scanDocument(rows, &document); err != nil {
			writeError(w, http.StatusInternalServerError, "documents_read_failed", "No fue posible consultar los documentos.")
			return
		}
		documents = append(documents, document)
	}
	writeJSON(w, http.StatusOK, map[string]any{"documents": documents})
}

type rowScanner interface {
	Scan(...any) error
}

func scanDocument(row rowScanner, document *documentResponse) error {
	return row.Scan(
		&document.ID, &document.PropertyID, &document.PropertyName, &document.DocumentType,
		&document.DisplayName, &document.OriginalName, &document.MimeType, &document.SizeBytes,
		&document.SHA256, &document.Tags, &document.Status, &document.DocumentDate,
		&document.Period, &document.Issuer, &document.AmountMinor, &document.CurrencyCode,
		&document.Confidentiality, &document.RejectionReason, &document.VerifiedAt,
		&document.VerifiedBy, &document.CreatedAt, &document.UpdatedAt, &document.Version,
	)
}

func (app *application) getDocument(w http.ResponseWriter, r *http.Request) {
	person := currentPrincipal(r)
	var document documentResponse
	err := scanDocument(app.db.QueryRow(r.Context(), `
		SELECT d.id, d.property_id, p.name, d.document_type, d.display_name, d.original_name,
		       d.mime_type, d.size_bytes, d.sha256, d.tags, d.status,
		       to_char(d.document_date, 'YYYY-MM-DD'), d.period, d.issuer, d.amount_minor,
		       d.currency_code, d.confidentiality, d.rejection_reason, d.verified_at,
		       d.verified_by, d.created_at, d.updated_at, d.version
		FROM documents d
		JOIN properties p ON p.id = d.property_id AND p.organization_id = d.organization_id
		WHERE d.id = $1 AND d.organization_id = $2
	`, chi.URLParam(r, "documentID"), person.OrganizationID), &document)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "document_not_found", "El documento no existe.")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "document_read_failed", "No fue posible consultar el documento.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"document": document})
}

func (app *application) updateDocument(w http.ResponseWriter, r *http.Request) {
	person := currentPrincipal(r)
	var input struct {
		DisplayName     string   `json:"display_name"`
		DocumentType    string   `json:"document_type"`
		DocumentDate    string   `json:"document_date"`
		Period          string   `json:"period"`
		Issuer          string   `json:"issuer"`
		AmountMinor     *int64   `json:"amount_minor"`
		CurrencyCode    string   `json:"currency_code"`
		Confidentiality string   `json:"confidentiality"`
		Tags            []string `json:"tags"`
		Status          string   `json:"status"`
		RejectionReason string   `json:"rejection_reason"`
		Version         int64    `json:"version"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	input.Issuer = strings.TrimSpace(input.Issuer)
	input.Period = strings.TrimSpace(input.Period)
	input.DocumentDate = strings.TrimSpace(input.DocumentDate)
	input.CurrencyCode = strings.ToUpper(strings.TrimSpace(input.CurrencyCode))
	input.RejectionReason = strings.TrimSpace(input.RejectionReason)
	input.Tags = normalizeTagSlice(input.Tags)
	if input.DisplayName == "" || len(input.DisplayName) > 240 ||
		!slices.Contains(allowedDocumentTypes, input.DocumentType) ||
		!slices.Contains([]string{"internal", "confidential", "restricted"}, input.Confidentiality) {
		writeError(w, http.StatusUnprocessableEntity, "invalid_document_metadata", "Revisa los metadatos del documento.")
		return
	}
	if input.DocumentDate != "" {
		if _, err := time.Parse("2006-01-02", input.DocumentDate); err != nil {
			writeError(w, http.StatusUnprocessableEntity, "invalid_document_date", "La fecha documental no es válida.")
			return
		}
	}
	if input.Period != "" {
		if _, err := time.Parse("2006-01", input.Period); err != nil {
			writeError(w, http.StatusUnprocessableEntity, "invalid_period", "El período debe usar el formato AAAA-MM.")
			return
		}
	}
	if input.AmountMinor != nil && (*input.AmountMinor < 0 || len(input.CurrencyCode) != 3) {
		writeError(w, http.StatusUnprocessableEntity, "invalid_amount", "El monto y la moneda no son válidos.")
		return
	}
	if input.AmountMinor == nil {
		input.CurrencyCode = ""
	}
	if input.Status == "rejected" && input.RejectionReason == "" {
		writeError(w, http.StatusUnprocessableEntity, "rejection_reason_required", "Indica el motivo del rechazo.")
		return
	}
	if slices.Contains([]string{"verified", "rejected", "archived"}, input.Status) && person.Role != "organization_admin" {
		writeError(w, http.StatusForbidden, "document_verify_forbidden", "No tienes permiso para cambiar este estado.")
		return
	}

	var updated documentResponse
	err := pgx.BeginFunc(r.Context(), app.db, func(tx pgx.Tx) error {
		var current documentResponse
		err := scanDocument(tx.QueryRow(r.Context(), `
			SELECT d.id, d.property_id, p.name, d.document_type, d.display_name, d.original_name,
			       d.mime_type, d.size_bytes, d.sha256, d.tags, d.status,
			       to_char(d.document_date, 'YYYY-MM-DD'), d.period, d.issuer, d.amount_minor,
			       d.currency_code, d.confidentiality, d.rejection_reason, d.verified_at,
			       d.verified_by, d.created_at, d.updated_at, d.version
			FROM documents d
			JOIN properties p ON p.id = d.property_id AND p.organization_id = d.organization_id
			WHERE d.id = $1 AND d.organization_id = $2
			FOR UPDATE OF d
		`, chi.URLParam(r, "documentID"), person.OrganizationID), &current)
		if err != nil {
			return err
		}
		if current.Version != input.Version {
			return errDocumentConflict
		}
		if !validDocumentTransition(current.Status, input.Status) {
			return errDocumentTransition
		}
		if (current.Status == "verified" && input.Status != "archived") || current.Status == "archived" {
			return errVerifiedDocumentImmutable
		}

		err = scanDocument(tx.QueryRow(r.Context(), `
			UPDATE documents d SET
				display_name = $3, document_type = $4,
				document_date = NULLIF($5, '')::date,
				period = NULLIF($6, ''), issuer = NULLIF($7, ''),
				amount_minor = $8, currency_code = NULLIF($9, ''),
				confidentiality = $10, tags = $11, status = $12,
				rejection_reason = CASE WHEN $12 = 'rejected' THEN NULLIF($13, '') ELSE NULL END,
				verified_at = CASE WHEN $12 = 'verified' THEN COALESCE(verified_at, now()) ELSE verified_at END,
				verified_by = CASE WHEN $12 = 'verified' THEN COALESCE(verified_by, $14::uuid) ELSE verified_by END,
				updated_by = $14, updated_at = now(), version = d.version + 1
			FROM properties p
			WHERE d.id = $1 AND d.organization_id = $2
			  AND p.id = d.property_id AND p.organization_id = d.organization_id
			RETURNING d.id, d.property_id, p.name, d.document_type, d.display_name, d.original_name,
			          d.mime_type, d.size_bytes, d.sha256, d.tags, d.status,
			          to_char(d.document_date, 'YYYY-MM-DD'), d.period, d.issuer, d.amount_minor,
			          d.currency_code, d.confidentiality, d.rejection_reason, d.verified_at,
			          d.verified_by, d.created_at, d.updated_at, d.version
		`, current.ID, person.OrganizationID, input.DisplayName, input.DocumentType,
			input.DocumentDate, input.Period, input.Issuer, input.AmountMinor, input.CurrencyCode,
			input.Confidentiality, input.Tags, input.Status, input.RejectionReason, person.UserID), &updated)
		if err != nil {
			return err
		}
		return audit(r.Context(), tx, person, "document.reviewed", "document", updated.ID, current, updated)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "document_not_found", "El documento no existe.")
		return
	}
	if errors.Is(err, errDocumentConflict) {
		writeError(w, http.StatusConflict, "document_version_conflict", "El documento fue modificado en otra sesión. Recarga antes de continuar.")
		return
	}
	if errors.Is(err, errDocumentTransition) || errors.Is(err, errVerifiedDocumentImmutable) {
		writeError(w, http.StatusConflict, "invalid_document_transition", "El cambio de estado no está permitido.")
		return
	}
	if err != nil {
		app.logger.Error("update document", "error", err)
		writeError(w, http.StatusInternalServerError, "document_update_failed", "No fue posible actualizar el documento.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"document": updated})
}

var errDocumentConflict = errors.New("document version conflict")
var errDocumentTransition = errors.New("invalid document transition")
var errVerifiedDocumentImmutable = errors.New("verified document is immutable")

func validDocumentTransition(current, next string) bool {
	transitions := map[string][]string{
		"uploaded":       {"uploaded", "pending_review", "verified", "rejected", "archived"},
		"pending_review": {"pending_review", "verified", "rejected", "archived"},
		"rejected":       {"rejected", "pending_review", "archived"},
		"verified":       {"verified", "archived"},
		"duplicate":      {"duplicate", "archived"},
		"archived":       {"archived"},
	}
	return slices.Contains(transitions[current], next)
}

func (app *application) downloadDocument(w http.ResponseWriter, r *http.Request) {
	person := currentPrincipal(r)
	var storageKey, originalName, mimeType string
	err := app.db.QueryRow(r.Context(), `
		SELECT storage_key, original_name, mime_type
		FROM documents
		WHERE id = $1 AND organization_id = $2 AND status <> 'archived'
	`, chi.URLParam(r, "documentID"), person.OrganizationID).Scan(&storageKey, &originalName, &mimeType)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "document_not_found", "El documento no existe.")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "document_read_failed", "No fue posible consultar el documento.")
		return
	}
	if directStorage, ok := app.storage.(DirectObjectStorage); ok {
		disposition := "attachment"
		if r.URL.Query().Get("inline") == "1" && (mimeType == "application/pdf" || strings.HasPrefix(mimeType, "image/")) {
			disposition = "inline"
		}
		signedURL, signErr := directStorage.PresignGet(r.Context(), storageKey, originalName, disposition, mimeType, 5*time.Minute)
		if signErr == nil {
			w.Header().Set("Cache-Control", "private, no-store")
			http.Redirect(w, r, signedURL, http.StatusTemporaryRedirect)
			return
		}
		app.logger.Warn("sign stored document download", "error", signErr)
	}
	object, err := app.storage.Get(r.Context(), storageKey)
	if err != nil {
		app.logger.Error("read stored document", "error", err)
		writeError(w, http.StatusBadGateway, "storage_read_failed", "No fue posible descargar el archivo.")
		return
	}
	defer object.Close()
	w.Header().Set("Content-Type", mimeType)
	disposition := "attachment"
	if r.URL.Query().Get("inline") == "1" && (mimeType == "application/pdf" || strings.HasPrefix(mimeType, "image/")) {
		disposition = "inline"
	}
	w.Header().Set("Content-Disposition", mime.FormatMediaType(disposition, map[string]string{"filename": originalName}))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if _, err := io.Copy(w, object); err != nil {
		app.logger.Warn("stream document", "error", err)
	}
}

func (app *application) setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name: app.options.CookieName, Value: token, Path: "/", HttpOnly: true,
		Secure: app.options.CookieSecure, SameSite: http.SameSiteLaxMode,
		MaxAge: int(app.options.SessionTTL.Seconds()), Expires: time.Now().Add(app.options.SessionTTL),
	})
}

func (app *application) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name: app.options.CookieName, Value: "", Path: "/", HttpOnly: true,
		Secure: app.options.CookieSecure, SameSite: http.SameSiteLaxMode,
		MaxAge: -1, Expires: time.Unix(0, 0),
	})
}

func currentPrincipal(r *http.Request) principal {
	return r.Context().Value(principalKey).(principal)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, destination any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return errors.New("La solicitud contiene datos inválidos.")
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return errors.New("La solicitud debe contener un único objeto JSON.")
	}
	return nil
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}

func validEmail(value string) bool {
	parsed, err := mail.ParseAddress(value)
	return err == nil && strings.EqualFold(parsed.Address, value) && len(value) <= 320
}

func slugify(value string) string {
	var result strings.Builder
	lastDash := false
	for _, char := range strings.ToLower(strings.TrimSpace(value)) {
		if unicode.IsLetter(char) || unicode.IsDigit(char) {
			if char <= unicode.MaxASCII {
				result.WriteRune(char)
				lastDash = false
			}
		} else if !lastDash && result.Len() > 0 {
			result.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(result.String(), "-")
}

func randomHex(bytesCount int) string {
	value := make([]byte, bytesCount)
	if _, err := rand.Read(value); err != nil {
		panic("secure random source unavailable")
	}
	return hex.EncodeToString(value)
}

func normalizeTags(value string) []string {
	return normalizeTagSlice(strings.Split(value, ","))
}

func normalizeTagSlice(values []string) []string {
	seen := make(map[string]struct{})
	tags := make([]string, 0, 10)
	for _, raw := range values {
		tag := strings.ToLower(strings.TrimSpace(raw))
		if tag == "" || len(tag) > 40 {
			continue
		}
		if _, exists := seen[tag]; exists {
			continue
		}
		seen[tag] = struct{}{}
		tags = append(tags, tag)
		if len(tags) == 10 {
			break
		}
	}
	return tags
}

func audit(ctx context.Context, tx pgx.Tx, person principal, action, entityType, entityID string, previous, next any) error {
	actorType := "user"
	if person.AuthKind == "agent" {
		actorType = "agent"
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO audit_events (
			organization_id, actor_type, actor_id, action, entity_type, entity_id, previous_state, new_state
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, person.OrganizationID, actorType, person.UserID, action, entityType, entityID, previous, next)
	return err
}
