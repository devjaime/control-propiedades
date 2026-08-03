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

func (app *application) agentListProperties(w http.ResponseWriter, r *http.Request) {
	person := currentPrincipal(r)
	if !requireScope(w, person, "property:read") {
		return
	}
	app.listProperties(w, r)
}

func (app *application) agentPropertyOverview(w http.ResponseWriter, r *http.Request) {
	person := currentPrincipal(r)
	if !requireScope(w, person, "property:read") || !requireScope(w, person, "incident:read") || !requireScope(w, person, "rent:read") {
		return
	}
	propertyID := chi.URLParam(r, "propertyID")
	var property propertyResponse
	err := app.db.QueryRow(r.Context(), `
		SELECT id, name, property_type, usage, status, address_line, commune, region, created_at
		FROM properties WHERE id = $1 AND organization_id = $2 AND status <> 'archived'
	`, propertyID, person.OrganizationID).Scan(
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
	chargeRows, err := app.db.Query(r.Context(), `
		SELECT id, period, charge_type, concept, to_char(due_date, 'YYYY-MM-DD'),
		       original_amount_minor, balance_minor, currency_code, status
		FROM charges WHERE organization_id = $1 AND property_id = $2
		ORDER BY period DESC, charge_type LIMIT 24
	`, person.OrganizationID, propertyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "rent_read_failed", "No fue posible consultar los cobros.")
		return
	}
	charges := make([]map[string]any, 0)
	for chargeRows.Next() {
		var id, period, chargeType, concept, dueDate, currencyCode, status string
		var originalAmount, balance int64
		if err := chargeRows.Scan(&id, &period, &chargeType, &concept, &dueDate, &originalAmount, &balance, &currencyCode, &status); err != nil {
			chargeRows.Close()
			writeError(w, http.StatusInternalServerError, "rent_read_failed", "No fue posible consultar los cobros.")
			return
		}
		charges = append(charges, map[string]any{"id": id, "period": period, "charge_type": chargeType, "concept": concept,
			"due_date": dueDate, "original_amount_minor": originalAmount, "balance_minor": balance,
			"currency_code": currencyCode, "status": status})
	}
	chargeRows.Close()
	incidents, err := app.loadAgentIncidents(r, person, propertyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "incident_read_failed", "No fue posible consultar los tickets.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"property": property, "charges": charges, "incidents": incidents})
}

func (app *application) agentOpenActions(w http.ResponseWriter, r *http.Request) {
	person := currentPrincipal(r)
	if !requireScope(w, person, "incident:read") {
		return
	}
	propertyID := chi.URLParam(r, "propertyID")
	rows, err := app.db.Query(r.Context(), `
		SELECT task.id, incident.id, incident.reference, incident.title, task.title, task.priority,
		       task.status, to_char(task.due_on, 'YYYY-MM-DD'), task.visible_to_tenant, task.version
		FROM incident_tasks task
		JOIN incidents incident ON incident.id = task.incident_id
		WHERE incident.organization_id = $1 AND incident.property_id = $2
		  AND incident.status IN ('open', 'in_progress') AND task.status = 'pending'
		ORDER BY task.due_on NULLS LAST, task.priority DESC, task.created_at
	`, person.OrganizationID, propertyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "actions_read_failed", "No fue posible consultar los accionables.")
		return
	}
	defer rows.Close()
	actions := make([]map[string]any, 0)
	for rows.Next() {
		var taskID, incidentID, reference, incidentTitle, title, priority, status string
		var dueOn *string
		var visible bool
		var version int64
		if err := rows.Scan(&taskID, &incidentID, &reference, &incidentTitle, &title, &priority, &status, &dueOn, &visible, &version); err != nil {
			writeError(w, http.StatusInternalServerError, "actions_read_failed", "No fue posible consultar los accionables.")
			return
		}
		actions = append(actions, map[string]any{"task_id": taskID, "incident_id": incidentID, "reference": reference,
			"incident_title": incidentTitle, "title": title, "priority": priority, "status": status,
			"due_on": dueOn, "visible_to_tenant": visible, "version": version})
	}
	writeJSON(w, http.StatusOK, map[string]any{"actions": actions})
}

func (app *application) agentInitiateDocumentUpload(w http.ResponseWriter, r *http.Request) {
	person := currentPrincipal(r)
	if !requireScope(w, person, "document:write") {
		return
	}
	app.initiateDocumentUpload(w, r)
}

func (app *application) agentCompleteDocumentUpload(w http.ResponseWriter, r *http.Request) {
	person := currentPrincipal(r)
	if !requireScope(w, person, "document:write") {
		return
	}
	app.completeDocumentUpload(w, r)
}

func (app *application) agentCreatePaymentFromDocument(w http.ResponseWriter, r *http.Request) {
	person := currentPrincipal(r)
	if !requireScope(w, person, "document:write") || !requireScope(w, person, "rent:write") {
		return
	}
	app.createPaymentFromDocument(w, r)
}

func (app *application) agentConfirmPayment(w http.ResponseWriter, r *http.Request) {
	person := currentPrincipal(r)
	if !requireScope(w, person, "rent:write") {
		return
	}
	app.confirmPayment(w, r)
}

func (app *application) loadAgentIncidents(r *http.Request, person principal, propertyID string) ([]incidentResponse, error) {
	rows, err := app.db.Query(r.Context(), `
		SELECT i.id, i.property_id, i.reference, i.title, i.summary, i.status, i.priority,
		       to_char(i.event_date, 'YYYY-MM-DD'), i.categories, i.tags, i.habitability,
		       i.created_at, i.updated_at, i.version
		FROM incidents i WHERE i.organization_id = $1 AND i.property_id = $2
		ORDER BY i.event_date DESC, i.created_at DESC
	`, person.OrganizationID, propertyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	incidents := make([]incidentResponse, 0)
	for rows.Next() {
		var incident incidentResponse
		if err := rows.Scan(&incident.ID, &incident.PropertyID, &incident.Reference, &incident.Title,
			&incident.Summary, &incident.Status, &incident.Priority, &incident.EventDate,
			&incident.Categories, &incident.Tags, &incident.Habitability, &incident.CreatedAt,
			&incident.UpdatedAt, &incident.Version); err != nil {
			return nil, err
		}
		if err := app.loadIncidentRelations(r, &incident, person); err != nil {
			return nil, err
		}
		incidents = append(incidents, incident)
	}
	return incidents, rows.Err()
}

func (app *application) agentAddIncidentUpdate(w http.ResponseWriter, r *http.Request) {
	person := currentPrincipal(r)
	if !requireScope(w, person, "incident:update") {
		return
	}
	var input struct {
		UpdateType      string `json:"update_type"`
		Content         string `json:"content"`
		OccurredAt      string `json:"occurred_at"`
		VisibleToTenant bool   `json:"visible_to_tenant"`
		Confirm         bool   `json:"confirm"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if !input.Confirm {
		writeError(w, http.StatusUnprocessableEntity, "explicit_confirmation_required", "La escritura requiere confirm=true.")
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
	incidentID := chi.URLParam(r, "incidentID")
	var updateID string
	err := pgx.BeginFunc(r.Context(), app.db, func(tx pgx.Tx) error {
		if err := tx.QueryRow(r.Context(), `
			INSERT INTO incident_updates (incident_id, update_type, content, occurred_at, created_by, visible_to_tenant)
			SELECT incident.id, $3, $4, $5::timestamptz, $2, $6
			FROM incidents incident WHERE incident.id = $1 AND incident.organization_id = $7
			RETURNING id
		`, incidentID, person.UserID, input.UpdateType, input.Content, input.OccurredAt,
			input.VisibleToTenant, person.OrganizationID).Scan(&updateID); err != nil {
			return err
		}
		return audit(r.Context(), tx, person, "incident.update_added", "incident_update", updateID, nil,
			map[string]any{"incident_id": incidentID, "update_type": input.UpdateType, "visible_to_tenant": input.VisibleToTenant})
	})
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "incident_not_found", "El incidente no existe.")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "incident_update_failed", "No fue posible guardar la actualización.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"id": updateID, "status": "created"})
}

func (app *application) agentUpdateIncidentTask(w http.ResponseWriter, r *http.Request) {
	person := currentPrincipal(r)
	if !requireScope(w, person, "incident:update") {
		return
	}
	var input struct {
		Status          string `json:"status"`
		Version         int64  `json:"version"`
		VisibleToTenant *bool  `json:"visible_to_tenant"`
		Confirm         bool   `json:"confirm"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if !input.Confirm || !slices.Contains([]string{"pending", "completed", "cancelled"}, input.Status) {
		writeError(w, http.StatusUnprocessableEntity, "invalid_task_update", "La actualización requiere estado válido y confirm=true.")
		return
	}
	taskID := chi.URLParam(r, "taskID")
	err := pgx.BeginFunc(r.Context(), app.db, func(tx pgx.Tx) error {
		result, err := tx.Exec(r.Context(), `
			UPDATE incident_tasks task SET status = $3,
				completed_at = CASE WHEN $3 = 'completed' THEN now() ELSE NULL END,
				visible_to_tenant = COALESCE($4, task.visible_to_tenant),
				updated_by = $5, updated_at = now(), version = task.version + 1
			FROM incidents incident
			WHERE task.id = $1 AND task.version = $2 AND incident.id = task.incident_id
			  AND incident.organization_id = $6
		`, taskID, input.Version, input.Status, input.VisibleToTenant, person.UserID, person.OrganizationID)
		if err != nil {
			return err
		}
		if result.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
		return audit(r.Context(), tx, person, "incident.task_updated", "incident_task", taskID, nil,
			map[string]any{"status": input.Status, "visible_to_tenant": input.VisibleToTenant})
	})
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusConflict, "incident_task_conflict", "La tarea cambió o no existe.")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "incident_task_update_failed", "No fue posible actualizar la tarea.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": taskID, "status": input.Status})
}
