package httpserver

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func sampleLeaseContractSnapshot() leaseContractSnapshot {
	return leaseContractSnapshot{
		DraftVersion: 1, GeneratedAt: time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC), Organization: "Patrimonio Familiar",
		PropertyName: "Casa de prueba", PropertyAddress: "Avenida Ejemplo 1234", PropertyCommune: "Coquimbo", PropertyRegion: "Coquimbo",
		LandlordName: "Propietario de prueba", LandlordIdentifier: "11.111.111-1", LandlordNationality: "Chilena",
		LandlordMaritalStatus: "casado", LandlordProfession: "profesional", LandlordAddress: "Santiago", LandlordEmail: "propietario@example.com",
		TenantName: "Postulante de prueba", TenantIdentifier: "12.345.678-5", TenantNationality: "Chilena",
		TenantMaritalStatus: "soltero", TenantProfession: "técnico", TenantAddress: "Coquimbo", TenantEmail: "postulante@example.com", TenantPhone: "+56 9 0000 0000",
		AuthorizedOccupants: "Una persona adulta", StartsOn: "2026-09-01", EndsOn: "2027-08-31", RentAmountMinor: 500000,
		PaymentDay: 5, DepositMonths: 2, DepositAmountMinor: 1000000, AdjustmentFrequency: 12, NoticeDays: 60,
		AdditionalGuarantee: "insurance", AdditionalGuaranteeText: "Seguro sujeto a evaluación y emisión de póliza",
		CommercialReportStatus: "Adjunto al expediente confidencial del postulante", TemplateVersion: leaseContractTemplateVersion, LegalBasisVersion: leaseContractLegalVersion,
	}
}

func TestValidChileRUT(t *testing.T) {
	for _, value := range []string{"11.111.111-1", "12.345.678-5"} {
		if !validChileRUT(value) {
			t.Fatalf("expected valid RUT: %s", value)
		}
	}
	for _, value := range []string{"11.111.111-2", "123", "not-a-rut"} {
		if validChileRUT(value) {
			t.Fatalf("expected invalid RUT: %s", value)
		}
	}
}

func TestBuildLeaseContractDraftArtifacts(t *testing.T) {
	snapshot := sampleLeaseContractSnapshot()
	pdf, err := BuildLeaseContractDraftPDF(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(pdf, []byte("%PDF")) {
		t.Fatal("PDF signature is missing")
	}
	docx, err := BuildLeaseContractDraftDOCX(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := zip.NewReader(bytes.NewReader(docx), int64(len(docx)))
	if err != nil {
		t.Fatal(err)
	}
	required := map[string]bool{"[Content_Types].xml": false, "word/document.xml": false, "word/styles.xml": false, "word/header1.xml": false}
	for _, file := range reader.File {
		if _, ok := required[file.Name]; ok {
			required[file.Name] = true
		}
		if file.Name == "word/document.xml" {
			stream, err := file.Open()
			if err != nil {
				t.Fatal(err)
			}
			var content bytes.Buffer
			_, _ = content.ReadFrom(stream)
			_ = stream.Close()
			if !strings.Contains(content.String(), "Avenida Ejemplo 1234") || !strings.Contains(content.String(), "BORRADOR") {
				t.Fatal("DOCX is missing canonical contract content")
			}
		}
	}
	for name, found := range required {
		if !found {
			t.Fatalf("DOCX part missing: %s", name)
		}
	}
	if outputDir := os.Getenv("CONTRACT_SAMPLE_OUTPUT"); outputDir != "" {
		if err := os.MkdirAll(outputDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(outputDir, "contrato-borrador-muestra.pdf"), pdf, 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(outputDir, "contrato-borrador-muestra.docx"), docx, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
