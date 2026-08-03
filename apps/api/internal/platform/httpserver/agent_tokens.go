package httpserver

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	appauth "github.com/devjaime/control-propiedades/apps/api/internal/auth"
)

var hermesAgentScopes = []string{
	"property:read", "incident:read", "incident:update", "rent:read", "document:write", "rent:write",
}

type agentTokenResponse struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Prefix     string     `json:"prefix"`
	Scopes     []string   `json:"scopes"`
	Status     string     `json:"status"`
	ExpiresAt  *time.Time `json:"expires_at"`
	LastUsedAt *time.Time `json:"last_used_at"`
	CreatedAt  time.Time  `json:"created_at"`
	Token      string     `json:"token,omitempty"`
}

func canManageAgentTokens(person principal) bool {
	return person.Role == "organization_admin" || person.Role == "property_manager"
}

func (app *application) listAgentTokens(w http.ResponseWriter, r *http.Request) {
	person := currentPrincipal(r)
	if !canManageAgentTokens(person) {
		writeError(w, http.StatusForbidden, "agent_token_forbidden", "No tienes permisos para administrar accesos de agentes.")
		return
	}
	rows, err := app.db.Query(r.Context(), `
		SELECT id, name, token_prefix, scopes, status, expires_at, last_used_at, created_at
		FROM api_tokens WHERE organization_id = $1
		ORDER BY CASE status WHEN 'active' THEN 0 ELSE 1 END, created_at DESC
	`, person.OrganizationID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "agent_token_read_failed", "No fue posible consultar los accesos del agente.")
		return
	}
	defer rows.Close()
	tokens := make([]agentTokenResponse, 0)
	for rows.Next() {
		var token agentTokenResponse
		if err := rows.Scan(&token.ID, &token.Name, &token.Prefix, &token.Scopes, &token.Status, &token.ExpiresAt, &token.LastUsedAt, &token.CreatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "agent_token_read_failed", "No fue posible consultar los accesos del agente.")
			return
		}
		tokens = append(tokens, token)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "agent_token_read_failed", "No fue posible consultar los accesos del agente.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tokens": tokens})
}

func (app *application) createAgentToken(w http.ResponseWriter, r *http.Request) {
	person := currentPrincipal(r)
	if !canManageAgentTokens(person) {
		writeError(w, http.StatusForbidden, "agent_token_forbidden", "No tienes permisos para administrar accesos de agentes.")
		return
	}
	var input struct {
		Name string `json:"name"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	if len(input.Name) < 3 || len(input.Name) > 120 {
		writeError(w, http.StatusUnprocessableEntity, "invalid_agent_token_name", "El nombre debe tener entre 3 y 120 caracteres.")
		return
	}
	randomValue := make([]byte, 32)
	if _, err := rand.Read(randomValue); err != nil {
		writeError(w, http.StatusInternalServerError, "agent_token_create_failed", "No fue posible generar el acceso del agente.")
		return
	}
	plainToken := "cp_" + base64.RawURLEncoding.EncodeToString(randomValue)
	prefix := plainToken[:12]
	expiresAt := time.Now().AddDate(0, 6, 0)
	var token agentTokenResponse
	err := pgx.BeginFunc(r.Context(), app.db, func(tx pgx.Tx) error {
		if err := tx.QueryRow(r.Context(), `
			INSERT INTO api_tokens (organization_id, user_id, name, token_prefix, token_hash, scopes, expires_at, created_by)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $2)
			RETURNING id, name, token_prefix, scopes, status, expires_at, last_used_at, created_at
		`, person.OrganizationID, person.UserID, input.Name, prefix, appauth.HashSessionToken(plainToken), hermesAgentScopes, expiresAt).Scan(
			&token.ID, &token.Name, &token.Prefix, &token.Scopes, &token.Status,
			&token.ExpiresAt, &token.LastUsedAt, &token.CreatedAt,
		); err != nil {
			return err
		}
		return audit(r.Context(), tx, person, "agent_token.created", "api_token", token.ID, nil,
			map[string]any{"name": token.Name, "prefix": token.Prefix, "expires_at": expiresAt, "scopes": token.Scopes})
	})
	if err != nil {
		app.logger.Error("create agent token", "error", err)
		writeError(w, http.StatusInternalServerError, "agent_token_create_failed", "No fue posible generar el acceso del agente.")
		return
	}
	token.Token = plainToken
	writeJSON(w, http.StatusCreated, map[string]any{"token": token})
}

func (app *application) revokeAgentToken(w http.ResponseWriter, r *http.Request) {
	person := currentPrincipal(r)
	if !canManageAgentTokens(person) {
		writeError(w, http.StatusForbidden, "agent_token_forbidden", "No tienes permisos para administrar accesos de agentes.")
		return
	}
	tokenID := chi.URLParam(r, "tokenID")
	err := pgx.BeginFunc(r.Context(), app.db, func(tx pgx.Tx) error {
		result, err := tx.Exec(r.Context(), `
			UPDATE api_tokens SET status = 'revoked', revoked_at = now()
			WHERE id = $1 AND organization_id = $2 AND status = 'active'
		`, tokenID, person.OrganizationID)
		if err != nil {
			return err
		}
		if result.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
		return audit(r.Context(), tx, person, "agent_token.revoked", "api_token", tokenID,
			map[string]any{"status": "active"}, map[string]any{"status": "revoked"})
	})
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "agent_token_not_found", "El acceso no existe o ya fue revocado.")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "agent_token_revoke_failed", "No fue posible revocar el acceso.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
