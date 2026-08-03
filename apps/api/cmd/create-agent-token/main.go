package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	databaseURL := requiredEnv("DATABASE_URL")
	outputPath := requiredEnv("AGENT_TOKEN_FILE")
	apiBaseURL := strings.TrimSuffix(requiredEnv("CONTROL_PROPIEDADES_API_URL"), "/")
	userEmail := strings.ToLower(strings.TrimSpace(os.Getenv("AGENT_USER_EMAIL")))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	var userID, organizationID string
	query := `
		SELECT user_account.id, membership.organization_id
		FROM users user_account
		JOIN memberships membership ON membership.user_id = user_account.id
		WHERE user_account.status = 'active' AND membership.status = 'active'
		  AND membership.role IN ('organization_admin', 'property_manager')
	`
	args := []any{}
	if userEmail != "" {
		query += " AND lower(user_account.email) = $1"
		args = append(args, userEmail)
	}
	query += " ORDER BY CASE membership.role WHEN 'organization_admin' THEN 0 ELSE 1 END, membership.created_at LIMIT 1"
	if err := pool.QueryRow(ctx, query, args...).Scan(&userID, &organizationID); err != nil {
		log.Fatalf("no se encontró un usuario autorizado: %v", err)
	}

	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		log.Fatal(err)
	}
	token := "cp_" + base64.RawURLEncoding.EncodeToString(buffer)
	digest := sha256.Sum256([]byte(token))
	prefix := token[:12]
	_, err = pool.Exec(ctx, `
		INSERT INTO api_tokens (
			organization_id, user_id, name, token_prefix, token_hash, scopes, expires_at, created_by
		) VALUES ($1, $2, 'Hermes Agent local', $3, $4,
			ARRAY['property:read','incident:read','incident:update','rent:read','document:write','rent:write']::text[],
			now() + interval '180 days', $2)
	`, organizationID, userID, prefix, digest[:])
	if err != nil {
		log.Fatal(err)
	}

	content := fmt.Sprintf("CONTROL_PROPIEDADES_API_URL=%s\nCONTROL_PROPIEDADES_API_TOKEN=%s\n", apiBaseURL, token)
	if err := os.MkdirAll(filepath.Dir(outputPath), 0700); err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(outputPath, []byte(content), 0600); err != nil {
		log.Fatal(err)
	}
	log.Printf("token creado (%s…) y guardado con permisos 0600 en %s", prefix, outputPath)
}

func requiredEnv(name string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		log.Fatalf("falta %s", name)
	}
	return value
}
