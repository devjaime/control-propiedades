package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/devjaime/control-propiedades/apps/api/internal/platform/httpserver"
	"github.com/devjaime/control-propiedades/apps/api/internal/platform/objectstorage"
)

type receiptRepair struct {
	id             string
	organizationID string
	documentID     string
	folio          string
	publicToken    string
	storageKey     string
	snapshot       []byte
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	databaseURL := requiredEnv("DATABASE_URL")
	publicWebURL := strings.TrimSuffix(requiredEnv("PUBLIC_WEB_URL"), "/")
	storage, err := objectstorage.New(
		requiredEnv("STORAGE_ENDPOINT"),
		requiredEnv("STORAGE_REGION"),
		requiredEnv("STORAGE_ACCESS_KEY"),
		requiredEnv("STORAGE_SECRET_KEY"),
		requiredEnv("STORAGE_BUCKET"),
		true,
	)
	if err != nil {
		log.Fatal(err)
	}
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	rows, err := pool.Query(ctx, `
		SELECT receipt.id, receipt.organization_id, receipt.document_id, receipt.folio,
		       receipt.public_token, document.storage_key, receipt.snapshot
		FROM rent_receipts receipt
		JOIN documents document ON document.id = receipt.document_id
		WHERE receipt.status = 'valid'
		  AND receipt.snapshot->>'verification_url' LIKE 'http://localhost%'
		ORDER BY receipt.issued_at
	`)
	if err != nil {
		log.Fatal(err)
	}
	var repairs []receiptRepair
	for rows.Next() {
		var repair receiptRepair
		if err := rows.Scan(&repair.id, &repair.organizationID, &repair.documentID, &repair.folio,
			&repair.publicToken, &repair.storageKey, &repair.snapshot); err != nil {
			rows.Close()
			log.Fatal(err)
		}
		repairs = append(repairs, repair)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		log.Fatal(err)
	}
	rows.Close()

	for _, repair := range repairs {
		if err := repairReceipt(ctx, pool, storage, publicWebURL, repair); err != nil {
			log.Fatalf("reparar %s: %v", repair.folio, err)
		}
		fmt.Printf("reparado=%s\n", repair.folio)
	}
	fmt.Printf("comprobantes_reparados=%d\n", len(repairs))
	if folio := strings.TrimSpace(os.Getenv("RECEIPT_EXPORT_FOLIO")); folio != "" {
		if err := exportReceipt(ctx, pool, storage, folio, requiredEnv("RECEIPT_EXPORT_PATH")); err != nil {
			log.Fatalf("exportar %s: %v", folio, err)
		}
		fmt.Printf("comprobante_exportado=%s\n", folio)
	}
}

func exportReceipt(ctx context.Context, pool *pgxpool.Pool, storage *objectstorage.Storage, folio, outputPath string) error {
	var storageKey string
	if err := pool.QueryRow(ctx, `
		SELECT document.storage_key
		FROM rent_receipts receipt
		JOIN documents document ON document.id = receipt.document_id
		WHERE receipt.folio = $1 AND receipt.status = 'valid'
	`, folio).Scan(&storageKey); err != nil {
		return err
	}
	object, err := storage.Get(ctx, storageKey)
	if err != nil {
		return err
	}
	defer object.Close()
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(file, object)
	closeErr := file.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func repairReceipt(ctx context.Context, pool *pgxpool.Pool, storage *objectstorage.Storage, publicWebURL string, repair receiptRepair) error {
	verificationURL := publicWebURL + "/verificar/comprobante/" + repair.publicToken
	repairedAt := time.Now().UTC()
	pdfBytes, updatedSnapshot, err := httpserver.RebuildRentReceiptPDF(repair.snapshot, verificationURL, repairedAt)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(pdfBytes)
	hash := hex.EncodeToString(digest[:])
	newStorageKey := strings.TrimSuffix(repair.storageKey, ".pdf") + "-qr-fixed.pdf"
	if err := storage.Put(ctx, newStorageKey, bytes.NewReader(pdfBytes), int64(len(pdfBytes)), "application/pdf"); err != nil {
		return err
	}

	err = pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		command, err := tx.Exec(ctx, `
			UPDATE rent_receipts
			SET snapshot = $2::jsonb, pdf_sha256 = $3, template_version = 'rent-receipt-v1.1'
			WHERE id = $1 AND snapshot->>'verification_url' LIKE 'http://localhost%'
		`, repair.id, string(updatedSnapshot), hash)
		if err != nil {
			return err
		}
		if command.RowsAffected() != 1 {
			return fmt.Errorf("el comprobante cambió durante la reparación")
		}
		if _, err := tx.Exec(ctx, `
			UPDATE documents
			SET storage_key = $2, size_bytes = $3, sha256 = $4,
			    updated_at = now(), version = version + 1
			WHERE id = $1
		`, repair.documentID, newStorageKey, len(pdfBytes), hash); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO audit_events (
				organization_id, actor_type, action, entity_type, entity_id,
				previous_state, new_state, reason, technical_context
			) VALUES ($1, 'system', 'receipt.qr_repaired', 'rent_receipt', $2,
			          $3::jsonb, $4::jsonb, 'Corrección de URL pública del código QR',
			          jsonb_build_object('old_storage_key', $5::text, 'new_storage_key', $6::text))
		`, repair.organizationID, repair.id, string(repair.snapshot), string(updatedSnapshot), repair.storageKey, newStorageKey)
		return err
	})
	if err != nil {
		_ = storage.Delete(ctx, newStorageKey)
		return err
	}
	if repair.storageKey != newStorageKey {
		_ = storage.Delete(ctx, repair.storageKey)
	}
	return nil
}

func requiredEnv(name string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		log.Fatalf("falta %s", name)
	}
	return value
}
