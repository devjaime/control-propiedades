package httpserver

import (
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

type incidentTaskResponse struct {
	ID              string     `json:"id"`
	Title           string     `json:"title"`
	Priority        string     `json:"priority"`
	Status          string     `json:"status"`
	DueOn           *string    `json:"due_on"`
	CompletedAt     *time.Time `json:"completed_at"`
	Version         int64      `json:"version"`
	VisibleToTenant bool       `json:"visible_to_tenant"`
}

type incidentUpdateResponse struct {
	ID              string    `json:"id"`
	UpdateType      string    `json:"update_type"`
	Content         string    `json:"content"`
	OccurredAt      time.Time `json:"occurred_at"`
	CreatedBy       string    `json:"created_by"`
	VisibleToTenant bool      `json:"visible_to_tenant"`
}

type incidentDocumentResponse struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	MimeType    string `json:"mime_type"`
}

type incidentResponse struct {
	ID           string                     `json:"id"`
	PropertyID   string                     `json:"property_id"`
	Reference    string                     `json:"reference"`
	Title        string                     `json:"title"`
	Summary      string                     `json:"summary"`
	Status       string                     `json:"status"`
	Priority     string                     `json:"priority"`
	EventDate    string                     `json:"event_date"`
	Categories   []string                   `json:"categories"`
	Tags         []string                   `json:"tags"`
	Habitability string                     `json:"habitability"`
	Tasks        []incidentTaskResponse     `json:"tasks"`
	Updates      []incidentUpdateResponse   `json:"updates"`
	Documents    []incidentDocumentResponse `json:"documents"`
	CreatedAt    time.Time                  `json:"created_at"`
	UpdatedAt    time.Time                  `json:"updated_at"`
	Version      int64                      `json:"version"`
}

func (app *application) listPropertyIncidents(w http.ResponseWriter, r *http.Request) {
	person := currentPrincipal(r)
	propertyID := chi.URLParam(r, "propertyID")
	rows, err := app.db.Query(r.Context(), `
		SELECT i.id, i.property_id, i.reference, i.title, i.summary, i.status, i.priority,
		       to_char(i.event_date, 'YYYY-MM-DD'), i.categories, i.tags, i.habitability,
		       i.created_at, i.updated_at, i.version
		FROM incidents i
		JOIN properties p ON p.id = i.property_id AND p.organization_id = i.organization_id
		WHERE i.organization_id = $1 AND i.property_id = $2
		ORDER BY i.event_date DESC, i.created_at DESC
	`, person.OrganizationID, propertyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "incidents_read_failed", "No fue posible consultar los incidentes.")
		return
	}
	defer rows.Close()
	incidents := make([]incidentResponse, 0)
	for rows.Next() {
		var incident incidentResponse
		if err := rows.Scan(&incident.ID, &incident.PropertyID, &incident.Reference, &incident.Title,
			&incident.Summary, &incident.Status, &incident.Priority, &incident.EventDate,
			&incident.Categories, &incident.Tags, &incident.Habitability, &incident.CreatedAt,
			&incident.UpdatedAt, &incident.Version); err != nil {
			writeError(w, http.StatusInternalServerError, "incidents_read_failed", "No fue posible consultar los incidentes.")
			return
		}
		incidents = append(incidents, incident)
	}
	for index := range incidents {
		if err := app.loadIncidentRelations(r, &incidents[index], person); err != nil {
			writeError(w, http.StatusInternalServerError, "incidents_read_failed", "No fue posible consultar el seguimiento del incidente.")
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"incidents": incidents})
}

func (app *application) loadIncidentRelations(r *http.Request, incident *incidentResponse, person principal) error {
	taskRows, err := app.db.Query(r.Context(), `
		SELECT id, title, priority, status, to_char(due_on, 'YYYY-MM-DD'), completed_at, version, visible_to_tenant
		FROM incident_tasks WHERE incident_id = $1 ORDER BY status, priority DESC, created_at
	`, incident.ID)
	if err != nil {
		return err
	}
	incident.Tasks = make([]incidentTaskResponse, 0)
	for taskRows.Next() {
		var task incidentTaskResponse
		if err := taskRows.Scan(&task.ID, &task.Title, &task.Priority, &task.Status, &task.DueOn, &task.CompletedAt, &task.Version, &task.VisibleToTenant); err != nil {
			taskRows.Close()
			return err
		}
		incident.Tasks = append(incident.Tasks, task)
	}
	taskRows.Close()

	updateRows, err := app.db.Query(r.Context(), `
		SELECT u.id, u.update_type, u.content, u.occurred_at, creator.name, u.visible_to_tenant
		FROM incident_updates u JOIN users creator ON creator.id = u.created_by
		WHERE u.incident_id = $1 ORDER BY u.occurred_at DESC, u.created_at DESC
	`, incident.ID)
	if err != nil {
		return err
	}
	incident.Updates = make([]incidentUpdateResponse, 0)
	for updateRows.Next() {
		var update incidentUpdateResponse
		if err := updateRows.Scan(&update.ID, &update.UpdateType, &update.Content, &update.OccurredAt, &update.CreatedBy, &update.VisibleToTenant); err != nil {
			updateRows.Close()
			return err
		}
		incident.Updates = append(incident.Updates, update)
	}
	updateRows.Close()

	documentRows, err := app.db.Query(r.Context(), `
		SELECT d.id, d.display_name, d.mime_type
		FROM incident_documents linked
		JOIN documents d ON d.id = linked.document_id
		WHERE linked.incident_id = $1 AND d.organization_id = $2
		ORDER BY linked.created_at DESC
	`, incident.ID, person.OrganizationID)
	if err != nil {
		return err
	}
	defer documentRows.Close()
	incident.Documents = make([]incidentDocumentResponse, 0)
	for documentRows.Next() {
		var document incidentDocumentResponse
		if err := documentRows.Scan(&document.ID, &document.DisplayName, &document.MimeType); err != nil {
			return err
		}
		incident.Documents = append(incident.Documents, document)
	}
	return documentRows.Err()
}

func (app *application) createIncident(w http.ResponseWriter, r *http.Request) {
	person := currentPrincipal(r)
	propertyID := chi.URLParam(r, "propertyID")
	var input struct {
		Reference    string   `json:"reference"`
		Title        string   `json:"title"`
		Summary      string   `json:"summary"`
		Priority     string   `json:"priority"`
		EventDate    string   `json:"event_date"`
		Categories   []string `json:"categories"`
		Tags         []string `json:"tags"`
		Habitability string   `json:"habitability"`
		Tasks        []struct {
			Title    string `json:"title"`
			Priority string `json:"priority"`
			DueOn    string `json:"due_on"`
		} `json:"tasks"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.Reference = strings.TrimSpace(input.Reference)
	input.Title = strings.TrimSpace(input.Title)
	input.Summary = strings.TrimSpace(input.Summary)
	input.Categories = normalizeTagSlice(input.Categories)
	input.Tags = normalizeTagSlice(input.Tags)
	if input.Reference == "" || len(input.Title) < 3 || len(input.Summary) < 3 ||
		!slices.Contains([]string{"low", "medium", "high", "critical"}, input.Priority) ||
		!slices.Contains([]string{"unaffected", "partially_affected", "uninhabitable"}, input.Habitability) {
		writeError(w, http.StatusUnprocessableEntity, "invalid_incident", "Revisa los datos obligatorios del incidente.")
		return
	}
	if _, err := time.Parse("2006-01-02", input.EventDate); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid_event_date", "La fecha del incidente no es válida.")
		return
	}
	var incident incidentResponse
	err := pgx.BeginFunc(r.Context(), app.db, func(tx pgx.Tx) error {
		if err := tx.QueryRow(r.Context(), `SELECT id FROM properties WHERE id = $1 AND organization_id = $2 AND status <> 'archived'`, propertyID, person.OrganizationID).Scan(&incident.PropertyID); err != nil {
			return err
		}
		if err := tx.QueryRow(r.Context(), `
			INSERT INTO incidents (organization_id, property_id, reference, title, summary, priority, event_date, categories, tags, habitability, created_by, updated_by)
			VALUES ($1, $2, $3, $4, $5, $6, $7::date, $8, $9, $10, $11, $11)
			RETURNING id, property_id, reference, title, summary, status, priority, to_char(event_date, 'YYYY-MM-DD'), categories, tags, habitability, created_at, updated_at, version
		`, person.OrganizationID, propertyID, input.Reference, input.Title, input.Summary, input.Priority,
			input.EventDate, input.Categories, input.Tags, input.Habitability, person.UserID).Scan(
			&incident.ID, &incident.PropertyID, &incident.Reference, &incident.Title, &incident.Summary,
			&incident.Status, &incident.Priority, &incident.EventDate, &incident.Categories, &incident.Tags,
			&incident.Habitability, &incident.CreatedAt, &incident.UpdatedAt, &incident.Version); err != nil {
			return err
		}
		for _, task := range input.Tasks {
			task.Title = strings.TrimSpace(task.Title)
			if len(task.Title) < 3 || !slices.Contains([]string{"low", "medium", "high", "critical"}, task.Priority) {
				return errors.New("invalid incident task")
			}
			if task.DueOn != "" {
				if _, err := time.Parse("2006-01-02", task.DueOn); err != nil {
					return err
				}
			}
			if _, err := tx.Exec(r.Context(), `
				INSERT INTO incident_tasks (incident_id, title, priority, due_on, created_by, updated_by)
				VALUES ($1, $2, $3, NULLIF($4, '')::date, $5, $5)
			`, incident.ID, task.Title, task.Priority, task.DueOn, person.UserID); err != nil {
				return err
			}
		}
		if _, err := tx.Exec(r.Context(), `
			INSERT INTO incident_updates (incident_id, update_type, content, occurred_at, created_by)
			VALUES ($1, 'status_change', 'Incidente registrado con estado abierto.', $2::date, $3)
		`, incident.ID, input.EventDate, person.UserID); err != nil {
			return err
		}
		return audit(r.Context(), tx, person, "incident.created", "incident", incident.ID, nil, incident)
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "property_not_found", "La propiedad no existe.")
			return
		}
		writeError(w, http.StatusUnprocessableEntity, "incident_create_failed", "No fue posible registrar el incidente. Revisa la referencia y las tareas.")
		return
	}
	_ = app.loadIncidentRelations(r, &incident, person)
	writeJSON(w, http.StatusCreated, map[string]any{"incident": incident})
}

func (app *application) addIncidentUpdate(w http.ResponseWriter, r *http.Request) {
	person := currentPrincipal(r)
	incidentID := chi.URLParam(r, "incidentID")
	var input struct {
		UpdateType      string `json:"update_type"`
		Content         string `json:"content"`
		OccurredAt      string `json:"occurred_at"`
		VisibleToTenant bool   `json:"visible_to_tenant"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.Content = strings.TrimSpace(input.Content)
	if len(input.Content) < 2 || !slices.Contains([]string{"note", "tenant_contact", "builder_claim", "insurance_claim", "inspection", "status_change"}, input.UpdateType) {
		writeError(w, http.StatusUnprocessableEntity, "invalid_incident_update", "Revisa el tipo y contenido de la actualización.")
		return
	}
	if input.OccurredAt == "" {
		input.OccurredAt = time.Now().Format(time.RFC3339)
	}
	if _, err := time.Parse(time.RFC3339, input.OccurredAt); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid_update_date", "La fecha de actualización no es válida.")
		return
	}
	var update incidentUpdateResponse
	err := app.db.QueryRow(r.Context(), `
		INSERT INTO incident_updates (incident_id, update_type, content, occurred_at, created_by, visible_to_tenant)
		SELECT i.id, $3, $4, $5::timestamptz, $2, $6 FROM incidents i WHERE i.id = $1 AND i.organization_id = $7
		RETURNING id, update_type, content, occurred_at, visible_to_tenant
	`, incidentID, person.UserID, input.UpdateType, input.Content, input.OccurredAt, input.VisibleToTenant, person.OrganizationID).Scan(
		&update.ID, &update.UpdateType, &update.Content, &update.OccurredAt, &update.VisibleToTenant)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "incident_not_found", "El incidente no existe.")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "incident_update_failed", "No fue posible guardar la actualización.")
		return
	}
	update.CreatedBy = person.Name
	writeJSON(w, http.StatusCreated, map[string]any{"update": update})
}

func (app *application) updateIncidentTask(w http.ResponseWriter, r *http.Request) {
	person := currentPrincipal(r)
	var input struct {
		Status          string `json:"status"`
		Version         int64  `json:"version"`
		VisibleToTenant *bool  `json:"visible_to_tenant"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if !slices.Contains([]string{"pending", "completed", "cancelled"}, input.Status) {
		writeError(w, http.StatusUnprocessableEntity, "invalid_task_status", "El estado de la tarea no es válido.")
		return
	}
	var task incidentTaskResponse
	err := app.db.QueryRow(r.Context(), `
		UPDATE incident_tasks task SET status = $3,
			completed_at = CASE WHEN $3 = 'completed' THEN now() ELSE NULL END,
			visible_to_tenant = COALESCE($4, task.visible_to_tenant),
			updated_by = $5, updated_at = now(), version = task.version + 1
		FROM incidents i
		WHERE task.id = $1 AND task.version = $2 AND i.id = task.incident_id AND i.organization_id = $6
		RETURNING task.id, task.title, task.priority, task.status, to_char(task.due_on, 'YYYY-MM-DD'), task.completed_at, task.version, task.visible_to_tenant
	`, chi.URLParam(r, "taskID"), input.Version, input.Status, input.VisibleToTenant, person.UserID, person.OrganizationID).Scan(
		&task.ID, &task.Title, &task.Priority, &task.Status, &task.DueOn, &task.CompletedAt, &task.Version, &task.VisibleToTenant)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusConflict, "incident_task_conflict", "La tarea cambió o ya no está disponible. Recarga la página.")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "incident_task_update_failed", "No fue posible actualizar la tarea.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"task": task})
}

func (app *application) linkIncidentDocument(w http.ResponseWriter, r *http.Request) {
	person := currentPrincipal(r)
	var input struct {
		DocumentID string `json:"document_id"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := app.db.Exec(r.Context(), `
		INSERT INTO incident_documents (incident_id, document_id)
		SELECT i.id, d.id FROM incidents i JOIN documents d ON d.organization_id = i.organization_id AND d.property_id = i.property_id
		WHERE i.id = $1 AND d.id = $2 AND i.organization_id = $3
		ON CONFLICT DO NOTHING
	`, chi.URLParam(r, "incidentID"), input.DocumentID, person.OrganizationID)
	if err != nil || result.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "incident_document_not_found", "No fue posible vincular la evidencia al incidente.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
