package httpserver

import (
	"errors"
	"io"
	"mime"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	appauth "github.com/devjaime/control-propiedades/apps/api/internal/auth"
)

var nonRUTDigitPattern = regexp.MustCompile(`[^0-9]`)

func rutAccessCode(identifier string) (string, bool) {
	compact := strings.ToUpper(strings.NewReplacer(".", "", "-", "", " ", "").Replace(identifier))
	if len(compact) < 6 {
		return "", false
	}
	number := compact[:len(compact)-1]
	digits := nonRUTDigitPattern.ReplaceAllString(number, "")
	if len(digits) < 4 {
		return "", false
	}
	return digits[len(digits)-4:], true
}

type tenantPortalLinkResponse struct {
	ID             string                    `json:"id"`
	PropertyID     string                    `json:"property_id"`
	PublicToken    string                    `json:"public_token"`
	Status         string                    `json:"status"`
	URL            string                    `json:"url"`
	ExpiresAt      *time.Time                `json:"expires_at"`
	LastAccessedAt *time.Time                `json:"last_accessed_at"`
	CreatedAt      time.Time                 `json:"created_at"`
	AccessKey      string                    `json:"access_key,omitempty"`
	AccessEvents   []tenantPortalAccessEvent `json:"access_events"`
}

type tenantPortalAccessEvent struct {
	ID             string    `json:"id"`
	CredentialKind string    `json:"credential_kind"`
	AccessedAt     time.Time `json:"accessed_at"`
}

type tenantPortalTask struct {
	ID       string  `json:"id"`
	Title    string  `json:"title"`
	Priority string  `json:"priority"`
	Status   string  `json:"status"`
	DueOn    *string `json:"due_on"`
}

type tenantPortalUpdate struct {
	ID         string    `json:"id"`
	UpdateType string    `json:"update_type"`
	Content    string    `json:"content"`
	OccurredAt time.Time `json:"occurred_at"`
}

type tenantPortalIncidentDocument struct {
	ID           string `json:"id"`
	DisplayName  string `json:"display_name"`
	OriginalName string `json:"original_name"`
	MimeType     string `json:"mime_type"`
	DocumentDate string `json:"document_date"`
	URL          string `json:"url"`
}

type tenantPortalIncident struct {
	ID           string                         `json:"id"`
	Reference    string                         `json:"reference"`
	Title        string                         `json:"title"`
	Summary      string                         `json:"summary"`
	Status       string                         `json:"status"`
	Priority     string                         `json:"priority"`
	Habitability string                         `json:"habitability"`
	EventDate    string                         `json:"event_date"`
	Tasks        []tenantPortalTask             `json:"tasks"`
	Updates      []tenantPortalUpdate           `json:"updates"`
	Documents    []tenantPortalIncidentDocument `json:"documents"`
}

type tenantPortalRentSummary struct {
	LastPaidPeriod string `json:"last_paid_period"`
	CurrentPeriod  string `json:"current_period"`
	AmountMinor    int64  `json:"amount_minor"`
	ReceivedMinor  int64  `json:"received_minor"`
	BalanceMinor   int64  `json:"balance_minor"`
	CurrencyCode   string `json:"currency_code"`
	Status         string `json:"status"`
}

type tenantPortalPaymentDocument struct {
	ID              string `json:"id"`
	Kind            string `json:"kind"`
	DisplayName     string `json:"display_name"`
	OriginalName    string `json:"original_name"`
	Period          string `json:"period"`
	DocumentDate    string `json:"document_date"`
	AmountMinor     int64  `json:"amount_minor"`
	CurrencyCode    string `json:"currency_code"`
	MimeType        string `json:"mime_type"`
	Folio           string `json:"folio,omitempty"`
	VerificationURL string `json:"verification_url,omitempty"`
}

type tenantPortalAccess struct {
	LinkID, OrganizationID, PropertyID, CredentialKind string
}

func (app *application) getTenantPortalLink(w http.ResponseWriter, r *http.Request) {
	person := currentPrincipal(r)
	propertyID := chi.URLParam(r, "propertyID")
	var link tenantPortalLinkResponse
	err := app.db.QueryRow(r.Context(), `
		SELECT id, property_id, public_token, status, expires_at, last_accessed_at, created_at
		FROM tenant_portal_links
		WHERE organization_id = $1 AND property_id = $2 AND status = 'active'
	`, person.OrganizationID, propertyID).Scan(
		&link.ID, &link.PropertyID, &link.PublicToken, &link.Status,
		&link.ExpiresAt, &link.LastAccessedAt, &link.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, http.StatusOK, map[string]any{"link": nil})
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "tenant_portal_read_failed", "No fue posible consultar el acceso compartido.")
		return
	}
	link.URL = strings.TrimSuffix(app.options.PublicWebURL, "/") + "/seguimiento/" + link.PublicToken
	link.AccessEvents = make([]tenantPortalAccessEvent, 0)
	rows, rowsErr := app.db.Query(r.Context(), `
		SELECT event.id, event.credential_kind, event.accessed_at
		FROM tenant_portal_access_events event
		JOIN tenant_portal_links portal ON portal.id = event.portal_link_id
		WHERE portal.organization_id = $1 AND portal.property_id = $2
		ORDER BY event.accessed_at DESC
		LIMIT 50
	`, person.OrganizationID, propertyID)
	if rowsErr != nil {
		writeError(w, http.StatusInternalServerError, "tenant_portal_access_log_failed", "No fue posible consultar el historial de accesos.")
		return
	}
	defer rows.Close()
	for rows.Next() {
		var event tenantPortalAccessEvent
		if err := rows.Scan(&event.ID, &event.CredentialKind, &event.AccessedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "tenant_portal_access_log_failed", "No fue posible consultar el historial de accesos.")
			return
		}
		link.AccessEvents = append(link.AccessEvents, event)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "tenant_portal_access_log_failed", "No fue posible consultar el historial de accesos.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"link": link})
}

func (app *application) rotateTenantPortalLink(w http.ResponseWriter, r *http.Request) {
	person := currentPrincipal(r)
	if person.Role != "organization_admin" && person.Role != "property_manager" {
		writeError(w, http.StatusForbidden, "tenant_portal_forbidden", "No tienes permisos para administrar este acceso.")
		return
	}
	propertyID := chi.URLParam(r, "propertyID")
	publicToken := randomHex(24)
	accessKey := strings.ToUpper(randomHex(6))
	accessKeyHash, err := appauth.HashPassword(accessKey)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "tenant_portal_key_failed", "No fue posible generar la clave.")
		return
	}
	var link tenantPortalLinkResponse
	err = pgx.BeginFunc(r.Context(), app.db, func(tx pgx.Tx) error {
		var leaseID *string
		var tenantIdentifier *string
		if err := tx.QueryRow(r.Context(), `
			SELECT lease.id, tenant.identifier
			FROM properties property
			LEFT JOIN leases lease ON lease.property_id = property.id AND lease.status IN ('active', 'ending')
			LEFT JOIN tenants tenant ON tenant.id = lease.tenant_id
			WHERE property.id = $1 AND property.organization_id = $2 AND property.status <> 'archived'
		`, propertyID, person.OrganizationID).Scan(&leaseID, &tenantIdentifier); err != nil {
			return err
		}
		var tenantAccessCodeHash *string
		if tenantIdentifier != nil {
			if code, ok := rutAccessCode(*tenantIdentifier); ok {
				hash, hashErr := appauth.HashAccessCode(code)
				if hashErr != nil {
					return hashErr
				}
				tenantAccessCodeHash = &hash
			}
		}
		if _, err := tx.Exec(r.Context(), `
			UPDATE tenant_portal_links SET status = 'revoked', revoked_at = now()
			WHERE organization_id = $1 AND property_id = $2 AND status = 'active'
		`, person.OrganizationID, propertyID); err != nil {
			return err
		}
		if err := tx.QueryRow(r.Context(), `
			INSERT INTO tenant_portal_links (
				organization_id, property_id, lease_id, public_token, access_key_hash, tenant_access_code_hash, created_by
			) VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING id, property_id, public_token, status, expires_at, last_accessed_at, created_at
		`, person.OrganizationID, propertyID, leaseID, publicToken, accessKeyHash, tenantAccessCodeHash, person.UserID).Scan(
			&link.ID, &link.PropertyID, &link.PublicToken, &link.Status,
			&link.ExpiresAt, &link.LastAccessedAt, &link.CreatedAt,
		); err != nil {
			return err
		}
		return audit(r.Context(), tx, person, "tenant_portal.rotated", "tenant_portal_link", link.ID, nil,
			map[string]any{"property_id": propertyID, "public_token_prefix": publicToken[:8]})
	})
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "property_not_found", "La propiedad no existe.")
		return
	}
	if err != nil {
		app.logger.Error("rotate tenant portal", "error", err)
		writeError(w, http.StatusInternalServerError, "tenant_portal_create_failed", "No fue posible crear el acceso compartido.")
		return
	}
	link.URL = strings.TrimSuffix(app.options.PublicWebURL, "/") + "/seguimiento/" + link.PublicToken
	link.AccessKey = accessKey
	writeJSON(w, http.StatusCreated, map[string]any{"link": link})
}

func (app *application) viewTenantPortal(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	var input struct {
		AccessKey string `json:"access_key"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.AccessKey = strings.TrimSpace(input.AccessKey)
	access, ok := app.authenticateTenantPortal(w, r, input.AccessKey)
	if !ok {
		return
	}
	organizationID, propertyID := access.OrganizationID, access.PropertyID

	var propertyName, commune string
	if err := app.db.QueryRow(r.Context(), `
		SELECT name, commune FROM properties WHERE id = $1 AND organization_id = $2
	`, propertyID, organizationID).Scan(&propertyName, &commune); err != nil {
		writeError(w, http.StatusNotFound, "tenant_portal_not_found", "El seguimiento ya no está disponible.")
		return
	}
	incidents, err := app.loadTenantPortalIncidents(r, organizationID, propertyID)
	if err != nil {
		app.logger.Error("read tenant portal", "error", err)
		writeError(w, http.StatusInternalServerError, "tenant_portal_read_failed", "No fue posible cargar el seguimiento.")
		return
	}
	paymentDocuments, err := app.loadTenantPortalPaymentDocuments(r, organizationID, propertyID)
	if err != nil {
		app.logger.Error("read tenant portal payment documents", "error", err)
		writeError(w, http.StatusInternalServerError, "tenant_portal_read_failed", "No fue posible cargar los comprobantes de pago.")
		return
	}
	var rentSummary tenantPortalRentSummary
	if err := app.db.QueryRow(r.Context(), `
		WITH rent AS (
			SELECT period, original_amount_minor, balance_minor, currency_code, status
			FROM charges WHERE organization_id = $1 AND property_id = $2 AND charge_type = 'rent'
		), latest_paid AS (
			SELECT COALESCE(MAX(period) FILTER (WHERE status = 'paid'), '') AS period FROM rent
		), current_charge AS (
			SELECT * FROM rent WHERE status IN ('partially_paid','pending','overdue','disputed') ORDER BY period LIMIT 1
		)
		SELECT latest_paid.period, COALESCE(current_charge.period,''),
		       COALESCE(current_charge.original_amount_minor,0),
		       COALESCE(current_charge.original_amount_minor-current_charge.balance_minor,0),
		       COALESCE(current_charge.balance_minor,0), COALESCE(current_charge.currency_code,'CLP'),
		       COALESCE(current_charge.status,'paid')
		FROM latest_paid LEFT JOIN current_charge ON true
	`, organizationID, propertyID).Scan(&rentSummary.LastPaidPeriod, &rentSummary.CurrentPeriod,
		&rentSummary.AmountMinor, &rentSummary.ReceivedMinor, &rentSummary.BalanceMinor,
		&rentSummary.CurrencyCode, &rentSummary.Status); err != nil {
		app.logger.Error("read tenant portal rent summary", "error", err)
		writeError(w, http.StatusInternalServerError, "tenant_portal_read_failed", "No fue posible cargar el estado de arriendo.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"property":          map[string]string{"name": propertyName, "commune": commune},
		"rent_summary":      rentSummary,
		"payment_documents": paymentDocuments,
		"incidents":         incidents,
		"generated_at":      time.Now(),
	})
}

func (app *application) authenticateTenantPortal(w http.ResponseWriter, r *http.Request, accessKey string) (tenantPortalAccess, bool) {
	publicToken := chi.URLParam(r, "token")
	var linkID, organizationID, propertyID, accessKeyHash string
	var tenantAccessCodeHash, ownerAccessCodeHash *string
	var lockedUntil *time.Time
	err := app.db.QueryRow(r.Context(), `
		SELECT link.id, link.organization_id, link.property_id, link.access_key_hash,
		       link.tenant_access_code_hash, organization.portal_owner_access_code_hash, link.locked_until
		FROM tenant_portal_links link
		JOIN organizations organization ON organization.id = link.organization_id
		WHERE link.public_token = $1 AND link.status = 'active'
		  AND (link.expires_at IS NULL OR link.expires_at > now())
	`, publicToken).Scan(&linkID, &organizationID, &propertyID, &accessKeyHash, &tenantAccessCodeHash, &ownerAccessCodeHash, &lockedUntil)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid_tenant_portal_access", "El enlace o la clave no son válidos.")
		return tenantPortalAccess{}, false
	}
	if lockedUntil != nil && lockedUntil.After(time.Now()) {
		writeError(w, http.StatusTooManyRequests, "tenant_portal_locked", "Acceso bloqueado temporalmente por intentos fallidos.")
		return tenantPortalAccess{}, false
	}
	validAccess := appauth.ComparePassword(accessKeyHash, accessKey)
	credentialKind := "legacy_key"
	if !validAccess && tenantAccessCodeHash != nil {
		validAccess = appauth.ComparePassword(*tenantAccessCodeHash, accessKey)
		if validAccess {
			credentialKind = "tenant"
		}
	}
	if !validAccess && ownerAccessCodeHash != nil {
		validAccess = appauth.ComparePassword(*ownerAccessCodeHash, accessKey)
		if validAccess {
			credentialKind = "owner"
		}
	}
	if !validAccess {
		_, _ = app.db.Exec(r.Context(), `
			UPDATE tenant_portal_links SET
				failed_attempts = LEAST(failed_attempts + 1, 100),
				locked_until = CASE WHEN failed_attempts + 1 >= 5 THEN now() + interval '15 minutes' ELSE NULL END
			WHERE id = $1
		`, linkID)
		writeError(w, http.StatusUnauthorized, "invalid_tenant_portal_access", "El enlace o la clave no son válidos.")
		return tenantPortalAccess{}, false
	}
	if _, err := app.db.Exec(r.Context(), `
		WITH accessed AS (
			UPDATE tenant_portal_links
			SET failed_attempts = 0, locked_until = NULL, last_accessed_at = now()
			WHERE id = $1
			RETURNING id
		)
		INSERT INTO tenant_portal_access_events (portal_link_id, credential_kind)
		SELECT id, $2 FROM accessed
	`, linkID, credentialKind); err != nil {
		app.logger.Error("record tenant portal access", "error", err, "link_id", linkID)
		writeError(w, http.StatusInternalServerError, "tenant_portal_access_log_failed", "No fue posible registrar el acceso al seguimiento.")
		return tenantPortalAccess{}, false
	}
	return tenantPortalAccess{LinkID: linkID, OrganizationID: organizationID, PropertyID: propertyID, CredentialKind: credentialKind}, true
}

func (app *application) loadTenantPortalPaymentDocuments(r *http.Request, organizationID, propertyID string) ([]tenantPortalPaymentDocument, error) {
	rows, err := app.db.Query(r.Context(), `
		SELECT document_id, kind, display_name, original_name, period, document_date,
		       amount_minor, currency_code, mime_type, folio, public_token
		FROM (
			SELECT DISTINCT d.id AS document_id, 'bank_receipt'::text AS kind, d.display_name,
			       d.original_name, COALESCE(d.period, charge.period) AS period,
			       to_char(payment.payment_date, 'YYYY-MM-DD') AS document_date,
			       payment.amount_minor, payment.currency_code, d.mime_type,
			       ''::text AS folio, ''::text AS public_token, payment.payment_date::timestamptz AS occurred_at, 1 AS sort_order
			FROM documents d
			JOIN payments payment ON payment.supporting_document_id = d.id
			JOIN payment_allocations allocation ON allocation.payment_id = payment.id AND allocation.status = 'confirmed'
			JOIN charges charge ON charge.id = allocation.charge_id AND charge.charge_type = 'rent'
			WHERE d.organization_id = $1 AND d.property_id = $2 AND d.document_type = 'bank_receipt'
			  AND d.status = 'verified' AND payment.status = 'reconciled'
			UNION ALL
			SELECT d.id, 'rent_receipt'::text, d.display_name, d.original_name,
			       COALESCE(d.period, charge.period), to_char(receipt.issued_at, 'YYYY-MM-DD'),
			       d.amount_minor, COALESCE(d.currency_code, 'CLP'), d.mime_type,
			       receipt.folio, receipt.public_token, receipt.issued_at, 2
			FROM documents d
			JOIN rent_receipts receipt ON receipt.document_id = d.id AND receipt.status = 'valid'
			JOIN charges charge ON charge.id = receipt.charge_id AND charge.charge_type = 'rent'
			WHERE d.organization_id = $1 AND d.property_id = $2 AND d.document_type = 'rent_receipt'
			  AND d.status = 'verified'
			UNION ALL
			SELECT d.id, 'utility'::text, d.display_name, d.original_name,
			       d.period, to_char(d.document_date, 'YYYY-MM-DD'), d.amount_minor,
			       COALESCE(d.currency_code, 'CLP'), d.mime_type,
			       ''::text, ''::text, d.document_date::timestamptz, 3
			FROM documents d
			WHERE d.organization_id = $1 AND d.property_id = $2 AND d.document_type = 'utility'
			  AND d.status = 'verified' AND d.period IS NOT NULL AND d.document_date IS NOT NULL
			  AND d.amount_minor IS NOT NULL AND d.tags @> ARRAY['pago']::text[]
		) archive
		ORDER BY period DESC, sort_order DESC, occurred_at DESC
	`, organizationID, propertyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	documents := make([]tenantPortalPaymentDocument, 0)
	for rows.Next() {
		var document tenantPortalPaymentDocument
		var publicToken string
		if err := rows.Scan(&document.ID, &document.Kind, &document.DisplayName, &document.OriginalName,
			&document.Period, &document.DocumentDate, &document.AmountMinor, &document.CurrencyCode,
			&document.MimeType, &document.Folio, &publicToken); err != nil {
			return nil, err
		}
		if publicToken != "" {
			document.VerificationURL = strings.TrimSuffix(app.options.PublicWebURL, "/") + "/verificar/comprobante/" + publicToken
		}
		documents = append(documents, document)
	}
	return documents, rows.Err()
}

func (app *application) downloadTenantPortalDocument(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "private, no-store")
	var input struct {
		AccessKey string `json:"access_key"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	access, ok := app.authenticateTenantPortal(w, r, strings.TrimSpace(input.AccessKey))
	if !ok {
		return
	}
	var storageKey, originalName, mimeType string
	err := app.db.QueryRow(r.Context(), `
		SELECT d.storage_key, d.original_name, d.mime_type
		FROM documents d
		WHERE d.id = $1 AND d.organization_id = $2 AND d.property_id = $3 AND d.status = 'verified'
		  AND (
			EXISTS (
				SELECT 1 FROM payments payment
				JOIN payment_allocations allocation ON allocation.payment_id = payment.id AND allocation.status = 'confirmed'
				JOIN charges charge ON charge.id = allocation.charge_id AND charge.charge_type = 'rent'
				WHERE payment.supporting_document_id = d.id AND payment.status = 'reconciled'
			) OR EXISTS (
				SELECT 1 FROM rent_receipts receipt
				JOIN charges charge ON charge.id = receipt.charge_id AND charge.charge_type = 'rent'
				WHERE receipt.document_id = d.id AND receipt.status = 'valid'
			) OR (d.document_type = 'utility' AND d.period IS NOT NULL AND d.document_date IS NOT NULL
			      AND d.amount_minor IS NOT NULL AND d.tags @> ARRAY['pago']::text[])
			OR EXISTS (
				SELECT 1
				FROM incident_documents incident_document
				JOIN incidents incident ON incident.id = incident_document.incident_id
				WHERE incident_document.document_id = d.id
				  AND incident.organization_id = d.organization_id
				  AND incident.property_id = d.property_id
				  AND incident.status IN ('open', 'in_progress', 'resolved')
				  AND d.tags @> ARRAY['tenant-visible']::text[]
			)
		  )
	`, chi.URLParam(r, "documentID"), access.OrganizationID, access.PropertyID).Scan(&storageKey, &originalName, &mimeType)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "document_not_found", "El archivo no existe o ya no está disponible.")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "document_read_failed", "No fue posible consultar el archivo.")
		return
	}
	object, err := app.storage.Get(r.Context(), storageKey)
	if err != nil {
		app.logger.Error("read tenant portal document", "error", err, "document_id", chi.URLParam(r, "documentID"))
		writeError(w, http.StatusBadGateway, "storage_read_failed", "No fue posible descargar el archivo.")
		return
	}
	defer object.Close()
	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": originalName}))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if _, err := io.Copy(w, object); err != nil {
		app.logger.Warn("stream tenant portal document", "error", err)
	}
}

func (app *application) loadTenantPortalIncidents(r *http.Request, organizationID, propertyID string) ([]tenantPortalIncident, error) {
	rows, err := app.db.Query(r.Context(), `
		SELECT id, reference, title, COALESCE(tenant_public_summary, ''), status, priority,
		       habitability, to_char(event_date, 'YYYY-MM-DD')
		FROM incidents
		WHERE organization_id = $1 AND property_id = $2 AND status IN ('open', 'in_progress', 'resolved')
		ORDER BY CASE status WHEN 'open' THEN 0 WHEN 'in_progress' THEN 1 ELSE 2 END, event_date DESC
	`, organizationID, propertyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	incidents := make([]tenantPortalIncident, 0)
	for rows.Next() {
		var incident tenantPortalIncident
		if err := rows.Scan(&incident.ID, &incident.Reference, &incident.Title, &incident.Summary,
			&incident.Status, &incident.Priority, &incident.Habitability, &incident.EventDate); err != nil {
			return nil, err
		}
		incident.Tasks = make([]tenantPortalTask, 0)
		taskRows, err := app.db.Query(r.Context(), `
			SELECT id, title, priority, status, to_char(due_on, 'YYYY-MM-DD')
			FROM incident_tasks WHERE incident_id = $1 AND visible_to_tenant = true
			ORDER BY CASE status WHEN 'pending' THEN 0 ELSE 1 END, due_on NULLS LAST, priority DESC
		`, incident.ID)
		if err != nil {
			return nil, err
		}
		for taskRows.Next() {
			var task tenantPortalTask
			if err := taskRows.Scan(&task.ID, &task.Title, &task.Priority, &task.Status, &task.DueOn); err != nil {
				taskRows.Close()
				return nil, err
			}
			incident.Tasks = append(incident.Tasks, task)
		}
		taskRows.Close()
		incident.Updates = make([]tenantPortalUpdate, 0)
		updateRows, err := app.db.Query(r.Context(), `
			SELECT id, update_type, content, occurred_at
			FROM incident_updates WHERE incident_id = $1 AND visible_to_tenant = true
			ORDER BY occurred_at DESC, created_at DESC LIMIT 50
		`, incident.ID)
		if err != nil {
			return nil, err
		}
		for updateRows.Next() {
			var update tenantPortalUpdate
			if err := updateRows.Scan(&update.ID, &update.UpdateType, &update.Content, &update.OccurredAt); err != nil {
				updateRows.Close()
				return nil, err
			}
			incident.Updates = append(incident.Updates, update)
		}
		updateRows.Close()
		incident.Documents = make([]tenantPortalIncidentDocument, 0)
		documentRows, err := app.db.Query(r.Context(), `
			SELECT document.id, document.display_name, document.original_name, document.mime_type, document.storage_key,
			       COALESCE(to_char(document.document_date, 'YYYY-MM-DD'), '')
			FROM incident_documents incident_document
			JOIN documents document ON document.id = incident_document.document_id
			WHERE incident_document.incident_id = $1
			  AND document.organization_id = $2
			  AND document.property_id = $3
			  AND document.status = 'verified'
			  AND document.tags @> ARRAY['tenant-visible']::text[]
			ORDER BY document.document_date DESC NULLS LAST, incident_document.created_at DESC, document.id
		`, incident.ID, organizationID, propertyID)
		if err != nil {
			return nil, err
		}
		for documentRows.Next() {
			var document tenantPortalIncidentDocument
			var storageKey string
			if err := documentRows.Scan(&document.ID, &document.DisplayName, &document.OriginalName,
				&document.MimeType, &storageKey, &document.DocumentDate); err != nil {
				documentRows.Close()
				return nil, err
			}
			directStorage, ok := app.storage.(DirectObjectStorage)
			if !ok {
				documentRows.Close()
				return nil, errors.New("tenant portal evidence requires direct object storage")
			}
			document.URL, err = directStorage.PresignGet(r.Context(), storageKey, document.OriginalName, "inline", document.MimeType, 5*time.Minute)
			if err != nil {
				documentRows.Close()
				return nil, err
			}
			incident.Documents = append(incident.Documents, document)
		}
		if err := documentRows.Err(); err != nil {
			documentRows.Close()
			return nil, err
		}
		documentRows.Close()
		incidents = append(incidents, incident)
	}
	return incidents, rows.Err()
}
