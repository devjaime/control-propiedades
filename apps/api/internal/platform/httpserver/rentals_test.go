package httpserver

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestBuildRentReceiptPDF(t *testing.T) {
	snapshot := map[string]any{
		"folio":        "CP-2026-000001",
		"organization": "Control de Prueba",
		"landlord":     "Propietario Prueba",
		"tenant":       "Arrendatario Prueba",
		"property":     "Casa Prueba",
		"address":      "Avenida Ejemplo 1234",
		"commune":      "Coquimbo",
		"period":       "2026-07",
		"amount_minor": int64(500000),
		"payment_lines": []receiptPaymentLine{
			{InstallmentNumber: 1, PaymentDate: "2026-07-04", AmountMinor: 250000, PaymentMethod: "bank_transfer", BankReference: "UNO"},
			{InstallmentNumber: 2, PaymentDate: "2026-07-22", AmountMinor: 250000, PaymentMethod: "deposit", BankReference: "DOS"},
		},
		"verification_code": "ABCD1234",
		"signature_name":    "Propietario Prueba",
		"signed_at":         time.Date(2026, 7, 12, 10, 30, 0, 0, time.UTC),
		"reviewer_name":     "C. Leiva",
		"reviewer_role":     "Contador Auditor",
		"reviewed_at":       time.Date(2026, 7, 12, 11, 0, 0, 0, time.UTC),
	}
	pdf, err := buildRentReceiptPDF(snapshot, "http://localhost:3000/verificar/comprobante/token-seguro")
	if err != nil {
		t.Fatalf("buildRentReceiptPDF() error = %v", err)
	}
	if !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		t.Fatal("generated document does not start with PDF signature")
	}
	if len(pdf) < 2000 {
		t.Fatalf("generated PDF is unexpectedly small: %d bytes", len(pdf))
	}
	if output := os.Getenv("RECEIPT_SAMPLE_OUTPUT"); output != "" {
		if err := os.WriteFile(output, pdf, 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRebuildRentReceiptPDFReplacesVerificationURL(t *testing.T) {
	snapshotJSON, err := json.Marshal(map[string]any{
		"folio": "CP-2026-000005", "organization": "Control de Prueba", "landlord": "Propietario Prueba",
		"tenant": "Arrendatario Prueba", "property": "Casa Prueba", "address": "Avenida Ejemplo 1234", "commune": "Coquimbo", "period": "2026-07",
		"amount_minor": int64(500000), "currency_code": "CLP", "verification_code": "ABCD1234",
		"signature_name": "Propietario Prueba", "signed_at": time.Date(2026, 7, 12, 10, 30, 0, 0, time.UTC),
		"payment_lines":    []receiptPaymentLine{{InstallmentNumber: 1, PaymentDate: "2026-07-04", AmountMinor: 500000, PaymentMethod: "bank_transfer"}},
		"verification_url": "http://localhost:3000/verificar/comprobante/token-seguro",
	})
	if err != nil {
		t.Fatal(err)
	}
	productionURL := "https://control-propiedades-web.vercel.app/verificar/comprobante/token-seguro"
	pdf, updatedJSON, err := RebuildRentReceiptPDF(snapshotJSON, productionURL, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		t.Fatal("rebuilt document does not start with PDF signature")
	}
	var updated map[string]any
	if err := json.Unmarshal(updatedJSON, &updated); err != nil {
		t.Fatal(err)
	}
	if updated["verification_url"] != productionURL {
		t.Fatalf("verification_url = %v", updated["verification_url"])
	}
	if updated["qr_repair_reason"] != "public_url_correction" {
		t.Fatalf("qr_repair_reason = %v", updated["qr_repair_reason"])
	}
}

func TestMaskName(t *testing.T) {
	if got, want := maskName("Ana María Pérez"), "A** M**** P****"; got != want {
		t.Fatalf("maskName() = %q, want %q", got, want)
	}
}

func TestFormatCLP(t *testing.T) {
	if got, want := formatCLP(1250000), "$1.250.000 CLP"; got != want {
		t.Fatalf("formatCLP() = %q, want %q", got, want)
	}
}
