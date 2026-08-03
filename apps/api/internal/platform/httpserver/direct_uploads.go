package httpserver

import (
	"errors"
	"net/http"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

var (
	errDirectUploadUnavailable = errors.New("direct upload storage unavailable")
	errDirectUploadExpired     = errors.New("direct upload expired")
	errDirectUploadDuplicate   = errors.New("direct upload duplicate")
)

const multipartPartSize int64 = 5 * 1024 * 1024

type directUploadPart struct {
	PartNumber int32  `json:"part_number"`
	UploadURL  string `json:"upload_url"`
}

type directUploadResponse struct {
	ID        string             `json:"id"`
	Mode      string             `json:"mode"`
	UploadURL string             `json:"upload_url,omitempty"`
	Headers   map[string]string  `json:"headers,omitempty"`
	Parts     []directUploadPart `json:"parts,omitempty"`
	ExpiresAt time.Time          `json:"expires_at"`
}

func (app *application) initiateDocumentUpload(w http.ResponseWriter, r *http.Request) {
	person := currentPrincipal(r)
	propertyID := chi.URLParam(r, "propertyID")
	directStorage, ok := app.storage.(DirectObjectStorage)
	if !ok {
		writeError(w, http.StatusNotImplemented, "direct_upload_unavailable", "El almacenamiento no permite cargas directas.")
		return
	}

	var input struct {
		OriginalName string   `json:"original_name"`
		DisplayName  string   `json:"display_name"`
		DocumentType string   `json:"document_type"`
		MimeType     string   `json:"mime_type"`
		SizeBytes    int64    `json:"size_bytes"`
		SHA256       string   `json:"sha256"`
		Tags         []string `json:"tags"`
		IncidentID   string   `json:"incident_id"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	input.OriginalName = filepath.Base(strings.TrimSpace(input.OriginalName))
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	if input.DisplayName == "" {
		input.DisplayName = input.OriginalName
	}
	input.MimeType = strings.ToLower(strings.TrimSpace(input.MimeType))
	input.SHA256 = strings.ToLower(strings.TrimSpace(input.SHA256))
	input.IncidentID = strings.TrimSpace(input.IncidentID)
	input.Tags = normalizeTagSlice(input.Tags)
	extensions := map[string]string{
		"application/pdf": ".pdf",
		"image/jpeg":      ".jpg",
		"image/png":       ".png",
		"image/webp":      ".webp",
	}
	extension, mimeAllowed := extensions[input.MimeType]
	if input.OriginalName == "." || input.OriginalName == "" || len(input.OriginalName) > 255 ||
		input.DisplayName == "" || len(input.DisplayName) > 240 ||
		!slices.Contains(allowedDocumentTypes, input.DocumentType) || !mimeAllowed ||
		input.SizeBytes <= 0 || input.SizeBytes > app.options.MaxUploadBytes ||
		len(input.SHA256) != 64 || !isLowerHex(input.SHA256) {
		writeError(w, http.StatusUnprocessableEntity, "invalid_upload", "Revisa el archivo, su tamaño y los metadatos.")
		return
	}

	var propertyName string
	if err := app.db.QueryRow(r.Context(), `
		SELECT name FROM properties
		WHERE id = $1 AND organization_id = $2 AND status <> 'archived'
	`, propertyID, person.OrganizationID).Scan(&propertyName); err != nil {
		writeError(w, http.StatusNotFound, "property_not_found", "La propiedad no existe.")
		return
	}
	_ = propertyName
	if input.IncidentID != "" {
		var exists bool
		if err := app.db.QueryRow(r.Context(), `
			SELECT EXISTS (
				SELECT 1 FROM incidents
				WHERE id = $1 AND property_id = $2 AND organization_id = $3
			)
		`, input.IncidentID, propertyID, person.OrganizationID).Scan(&exists); err != nil || !exists {
			writeError(w, http.StatusNotFound, "incident_not_found", "El incidente no existe para esta propiedad.")
			return
		}
	}

	storageKey := "organizations/" + person.OrganizationID + "/properties/" + propertyID + "/documents/" + randomHex(16) + extension
	expiresAt := time.Now().Add(10 * time.Minute)
	var upload directUploadResponse
	err := app.db.QueryRow(r.Context(), `
		INSERT INTO document_uploads (
			organization_id, property_id, incident_id, uploaded_by, storage_key,
			display_name, original_name, document_type, mime_type, size_bytes,
			sha256, tags, expires_at
		) VALUES ($1, $2, NULLIF($3, '')::uuid, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id, expires_at
	`, person.OrganizationID, propertyID, input.IncidentID, person.UserID, storageKey,
		input.DisplayName, input.OriginalName, input.DocumentType, input.MimeType,
		input.SizeBytes, input.SHA256, input.Tags, expiresAt).Scan(&upload.ID, &upload.ExpiresAt)
	if err != nil {
		app.logger.Error("create direct document upload", "error", err)
		writeError(w, http.StatusInternalServerError, "upload_create_failed", "No fue posible preparar la carga.")
		return
	}

	if input.SizeBytes > multipartPartSize {
		upload.Mode = "multipart"
		partCount := int32((input.SizeBytes + multipartPartSize - 1) / multipartPartSize)
		var uploadID string
		var signedParts map[int32]string
		uploadID, signedParts, err = directStorage.InitiateMultipart(r.Context(), storageKey, input.MimeType, input.SHA256, partCount, 10*time.Minute)
		if err == nil {
			if _, updateErr := app.db.Exec(r.Context(), `
				UPDATE document_uploads SET multipart_upload_id = $2 WHERE id = $1
			`, upload.ID, uploadID); updateErr != nil {
				_ = directStorage.AbortMultipart(r.Context(), storageKey, uploadID)
				err = updateErr
			} else {
				upload.Parts = make([]directUploadPart, 0, partCount)
				for partNumber := int32(1); partNumber <= partCount; partNumber++ {
					upload.Parts = append(upload.Parts, directUploadPart{PartNumber: partNumber, UploadURL: signedParts[partNumber]})
				}
			}
		}
	} else {
		upload.Mode = "single"
		upload.UploadURL, upload.Headers, err = directStorage.PresignPut(r.Context(), storageKey, input.MimeType, input.SHA256, 10*time.Minute)
	}
	if err != nil {
		_, _ = app.db.Exec(r.Context(), "DELETE FROM document_uploads WHERE id = $1", upload.ID)
		app.logger.Error("sign direct document upload", "error", err)
		writeError(w, http.StatusBadGateway, "upload_sign_failed", "No fue posible autorizar la carga.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"upload": upload})
}

func (app *application) completeDocumentUpload(w http.ResponseWriter, r *http.Request) {
	person := currentPrincipal(r)
	uploadID := chi.URLParam(r, "uploadID")
	directStorage, ok := app.storage.(DirectObjectStorage)
	if !ok {
		writeError(w, http.StatusNotImplemented, "direct_upload_unavailable", "El almacenamiento no permite cargas directas.")
		return
	}

	var storageKey, expectedMime, expectedSHA256, multipartUploadID string
	var expectedSize int64
	err := app.db.QueryRow(r.Context(), `
		SELECT storage_key, size_bytes, mime_type, sha256, COALESCE(multipart_upload_id, '')
		FROM document_uploads
		WHERE id = $1 AND organization_id = $2 AND uploaded_by = $3
		  AND completed_at IS NULL AND expires_at > now()
	`, uploadID, person.OrganizationID, person.UserID).Scan(&storageKey, &expectedSize, &expectedMime, &expectedSHA256, &multipartUploadID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusGone, "upload_expired", "La autorización de carga venció. Intenta subir el archivo nuevamente.")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "upload_read_failed", "No fue posible verificar la carga.")
		return
	}
	if multipartUploadID != "" {
		var input struct {
			Parts []struct {
				PartNumber int32  `json:"part_number"`
				ETag       string `json:"etag"`
			} `json:"parts"`
		}
		if err := decodeJSON(w, r, &input); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}
		expectedParts := int32((expectedSize + multipartPartSize - 1) / multipartPartSize)
		etags := make(map[int32]string, expectedParts)
		for _, part := range input.Parts {
			part.ETag = strings.TrimSpace(part.ETag)
			if part.PartNumber < 1 || part.PartNumber > expectedParts || part.ETag == "" || len(part.ETag) > 256 {
				writeError(w, http.StatusUnprocessableEntity, "invalid_multipart_upload", "La carga multipartes está incompleta.")
				return
			}
			etags[part.PartNumber] = part.ETag
		}
		if int32(len(etags)) != expectedParts {
			writeError(w, http.StatusUnprocessableEntity, "invalid_multipart_upload", "La carga multipartes está incompleta.")
			return
		}
		if err := directStorage.CompleteMultipart(r.Context(), storageKey, multipartUploadID, etags); err != nil {
			app.logger.Error("complete multipart document upload", "error", err)
			writeError(w, http.StatusBadGateway, "multipart_complete_failed", "No fue posible finalizar la carga del archivo.")
			return
		}
	}
	storedSize, storedMime, storedMetadata, err := directStorage.Stat(r.Context(), storageKey)
	storedSHA256 := metadataValue(storedMetadata, "x-amz-meta-sha256", "sha256")
	if err != nil || storedSize != expectedSize || !strings.EqualFold(storedMime, expectedMime) ||
		(storedSHA256 != "" && !strings.EqualFold(storedSHA256, expectedSHA256)) {
		_ = app.storage.Delete(r.Context(), storageKey)
		writeError(w, http.StatusUnprocessableEntity, "upload_mismatch", "El archivo recibido no coincide con la carga autorizada.")
		return
	}

	var document documentResponse
	var incidentID string
	err = pgx.BeginFunc(r.Context(), app.db, func(tx pgx.Tx) error {
		var displayName, originalName, documentType, mimeType, sha256, propertyID, propertyName string
		var tags []string
		var sizeBytes int64
		var expiresAt time.Time
		var completedAt *time.Time
		if err := tx.QueryRow(r.Context(), `
			SELECT upload.property_id, property.name, COALESCE(upload.incident_id::text, ''),
			       upload.display_name, upload.original_name, upload.document_type,
			       upload.mime_type, upload.size_bytes, upload.sha256, upload.tags,
			       upload.expires_at, upload.completed_at
			FROM document_uploads upload
			JOIN properties property ON property.id = upload.property_id
			WHERE upload.id = $1 AND upload.organization_id = $2 AND upload.uploaded_by = $3
			FOR UPDATE OF upload
		`, uploadID, person.OrganizationID, person.UserID).Scan(
			&propertyID, &propertyName, &incidentID, &displayName, &originalName,
			&documentType, &mimeType, &sizeBytes, &sha256, &tags, &expiresAt, &completedAt,
		); err != nil {
			return err
		}
		if completedAt != nil {
			return errDirectUploadUnavailable
		}
		if time.Now().After(expiresAt) {
			return errDirectUploadExpired
		}
		var duplicateID string
		duplicateErr := tx.QueryRow(r.Context(), `
			SELECT id FROM documents WHERE organization_id = $1 AND sha256 = $2 LIMIT 1
		`, person.OrganizationID, sha256).Scan(&duplicateID)
		if duplicateErr == nil {
			return errDirectUploadDuplicate
		}
		if !errors.Is(duplicateErr, pgx.ErrNoRows) {
			return duplicateErr
		}
		if err := tx.QueryRow(r.Context(), `
			INSERT INTO documents (
				organization_id, property_id, document_type, display_name, original_name,
				storage_key, mime_type, size_bytes, sha256, tags, uploaded_by
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			RETURNING id, property_id, document_type, display_name, original_name, mime_type,
			          size_bytes, sha256, tags, status, confidentiality, created_at, updated_at, version
		`, person.OrganizationID, propertyID, documentType, displayName, originalName,
			storageKey, mimeType, sizeBytes, sha256, tags, person.UserID).Scan(
			&document.ID, &document.PropertyID, &document.DocumentType, &document.DisplayName,
			&document.OriginalName, &document.MimeType, &document.SizeBytes, &document.SHA256,
			&document.Tags, &document.Status, &document.Confidentiality, &document.CreatedAt,
			&document.UpdatedAt, &document.Version,
		); err != nil {
			return err
		}
		document.PropertyName = propertyName
		if incidentID != "" {
			if _, err := tx.Exec(r.Context(), `
				INSERT INTO incident_documents (incident_id, document_id) VALUES ($1, $2)
				ON CONFLICT DO NOTHING
			`, incidentID, document.ID); err != nil {
				return err
			}
		}
		if _, err := tx.Exec(r.Context(), `
			UPDATE document_uploads SET completed_at = now() WHERE id = $1
		`, uploadID); err != nil {
			return err
		}
		return audit(r.Context(), tx, person, "document.uploaded", "document", document.ID, nil,
			map[string]any{"name": document.DisplayName, "sha256": document.SHA256, "incident_id": incidentID})
	})
	if errors.Is(err, errDirectUploadDuplicate) {
		_ = app.storage.Delete(r.Context(), storageKey)
		writeError(w, http.StatusConflict, "duplicate_document", "Este archivo ya fue cargado.")
		return
	}
	if errors.Is(err, errDirectUploadExpired) || errors.Is(err, errDirectUploadUnavailable) || errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusGone, "upload_expired", "La autorización de carga ya no está disponible.")
		return
	}
	if err != nil {
		_ = app.storage.Delete(r.Context(), storageKey)
		app.logger.Error("complete direct document upload", "error", err)
		writeError(w, http.StatusInternalServerError, "upload_complete_failed", "No fue posible registrar el archivo.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"document": document})
}

func isLowerHex(value string) bool {
	for _, char := range value {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
			return false
		}
	}
	return true
}

func metadataValue(metadata map[string]string, names ...string) string {
	for key, value := range metadata {
		for _, name := range names {
			if strings.EqualFold(key, name) {
				return strings.TrimSpace(value)
			}
		}
	}
	return ""
}
