package httpserver

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/phpdave11/gofpdf"
)

const (
	leaseContractTemplateVersion = "cl-residential-v1.0"
	leaseContractLegalVersion    = "CL-18.101-21.461-2026-08"
)

type tenantApplicationDocumentResponse struct {
	DocumentID   string `json:"document_id"`
	DocumentKind string `json:"document_kind"`
	DisplayName  string `json:"display_name"`
	MimeType     string `json:"mime_type"`
	LinkedAt     string `json:"linked_at"`
}

type tenantApplicationResponse struct {
	ID                                  string                              `json:"id"`
	FullName                            string                              `json:"full_name"`
	Identifier                          string                              `json:"identifier"`
	Nationality                         string                              `json:"nationality"`
	MaritalStatus                       string                              `json:"marital_status"`
	Profession                          string                              `json:"profession"`
	CurrentAddress                      string                              `json:"current_address"`
	Email                               string                              `json:"email"`
	Phone                               string                              `json:"phone"`
	Employer                            string                              `json:"employer"`
	MonthlyIncomeMinor                  *int64                              `json:"monthly_income_minor"`
	AuthorizedOccupants                 string                              `json:"authorized_occupants"`
	Notes                               string                              `json:"notes"`
	Status                              string                              `json:"status"`
	CommercialReportProvenanceConfirmed bool                                `json:"commercial_report_provenance_confirmed"`
	CreatedAt                           string                              `json:"created_at"`
	Documents                           []tenantApplicationDocumentResponse `json:"documents"`
}

type leaseContractDraftResponse struct {
	ID                string `json:"id"`
	ApplicationID     string `json:"application_id"`
	DraftVersion      int    `json:"draft_version"`
	TemplateVersion   string `json:"template_version"`
	LegalBasisVersion string `json:"legal_basis_version"`
	Status            string `json:"status"`
	PDFDocumentID     string `json:"pdf_document_id"`
	DOCXDocumentID    string `json:"docx_document_id"`
	SignedDocumentID  string `json:"signed_document_id"`
	PDFSHA256         string `json:"pdf_sha256"`
	DOCXSHA256        string `json:"docx_sha256"`
	LegalReviewedBy   string `json:"legal_reviewed_by"`
	CreatedAt         string `json:"created_at"`
}

type leaseContractSnapshot struct {
	DraftVersion            int       `json:"draft_version"`
	GeneratedAt             time.Time `json:"generated_at"`
	Organization            string    `json:"organization"`
	PropertyName            string    `json:"property_name"`
	PropertyAddress         string    `json:"property_address"`
	PropertyCommune         string    `json:"property_commune"`
	PropertyRegion          string    `json:"property_region"`
	LandlordName            string    `json:"landlord_name"`
	LandlordIdentifier      string    `json:"landlord_identifier"`
	LandlordNationality     string    `json:"landlord_nationality"`
	LandlordMaritalStatus   string    `json:"landlord_marital_status"`
	LandlordProfession      string    `json:"landlord_profession"`
	LandlordAddress         string    `json:"landlord_address"`
	LandlordEmail           string    `json:"landlord_email"`
	TenantName              string    `json:"tenant_name"`
	TenantIdentifier        string    `json:"tenant_identifier"`
	TenantNationality       string    `json:"tenant_nationality"`
	TenantMaritalStatus     string    `json:"tenant_marital_status"`
	TenantProfession        string    `json:"tenant_profession"`
	TenantAddress           string    `json:"tenant_address"`
	TenantEmail             string    `json:"tenant_email"`
	TenantPhone             string    `json:"tenant_phone"`
	AuthorizedOccupants     string    `json:"authorized_occupants"`
	StartsOn                string    `json:"starts_on"`
	EndsOn                  string    `json:"ends_on"`
	RentAmountMinor         int64     `json:"rent_amount_minor"`
	PaymentDay              int       `json:"payment_day"`
	DepositMonths           int       `json:"deposit_months"`
	DepositAmountMinor      int64     `json:"deposit_amount_minor"`
	AdjustmentFrequency     int       `json:"adjustment_frequency_months"`
	NoticeDays              int       `json:"notice_days"`
	AdditionalGuarantee     string    `json:"additional_guarantee"`
	AdditionalGuaranteeText string    `json:"additional_guarantee_text"`
	UtilitiesResponsibility string    `json:"utilities_responsibility"`
	SpecialTerms            string    `json:"special_terms"`
	CommercialReportStatus  string    `json:"commercial_report_status"`
	TemplateVersion         string    `json:"template_version"`
	LegalBasisVersion       string    `json:"legal_basis_version"`
}

func (app *application) getLeaseContractWorkflow(w http.ResponseWriter, r *http.Request) {
	person := currentPrincipal(r)
	propertyID := chi.URLParam(r, "propertyID")
	var exists bool
	if err := app.db.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM properties WHERE id=$1 AND organization_id=$2 AND status <> 'archived')`, propertyID, person.OrganizationID).Scan(&exists); err != nil || !exists {
		writeError(w, http.StatusNotFound, "property_not_found", "La propiedad no existe.")
		return
	}
	rows, err := app.db.Query(r.Context(), `
		SELECT id,full_name,identifier,nationality,marital_status,profession,current_address,email,phone,
		       employer,monthly_income_minor,authorized_occupants,notes,status,
		       commercial_report_provenance_confirmed,to_char(created_at,'YYYY-MM-DD HH24:MI')
		FROM tenant_applications
		WHERE organization_id=$1 AND property_id=$2 AND status <> 'archived'
		ORDER BY created_at DESC
	`, person.OrganizationID, propertyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "applications_read_failed", "No fue posible consultar las postulaciones.")
		return
	}
	applications := make([]tenantApplicationResponse, 0)
	for rows.Next() {
		var item tenantApplicationResponse
		if err := rows.Scan(&item.ID, &item.FullName, &item.Identifier, &item.Nationality, &item.MaritalStatus,
			&item.Profession, &item.CurrentAddress, &item.Email, &item.Phone, &item.Employer,
			&item.MonthlyIncomeMinor, &item.AuthorizedOccupants, &item.Notes, &item.Status,
			&item.CommercialReportProvenanceConfirmed, &item.CreatedAt); err != nil {
			rows.Close()
			writeError(w, http.StatusInternalServerError, "applications_read_failed", "No fue posible consultar las postulaciones.")
			return
		}
		item.Documents = make([]tenantApplicationDocumentResponse, 0)
		applications = append(applications, item)
	}
	rows.Close()
	applicationIndex := make(map[string]int, len(applications))
	for index := range applications {
		applicationIndex[applications[index].ID] = index
	}
	documentRows, err := app.db.Query(r.Context(), `
		SELECT link.application_id,link.document_id,link.document_kind,document.display_name,document.mime_type,
		       to_char(link.linked_at,'YYYY-MM-DD HH24:MI')
		FROM tenant_application_documents link
		JOIN tenant_applications application ON application.id=link.application_id
		JOIN documents document ON document.id=link.document_id
		WHERE application.organization_id=$1 AND application.property_id=$2
		ORDER BY link.linked_at DESC
	`, person.OrganizationID, propertyID)
	if err == nil {
		for documentRows.Next() {
			var applicationID string
			var document tenantApplicationDocumentResponse
			if documentRows.Scan(&applicationID, &document.DocumentID, &document.DocumentKind, &document.DisplayName, &document.MimeType, &document.LinkedAt) == nil {
				if index, found := applicationIndex[applicationID]; found {
					applications[index].Documents = append(applications[index].Documents, document)
				}
			}
		}
		documentRows.Close()
	}

	draftRows, err := app.db.Query(r.Context(), `
		SELECT id,application_id,draft_version,template_version,legal_basis_version,status,pdf_document_id,docx_document_id,
		       COALESCE(signed_document_id::text,''),pdf_sha256,docx_sha256,COALESCE(legal_reviewed_by,''),to_char(created_at,'YYYY-MM-DD HH24:MI')
		FROM lease_contract_drafts
		WHERE organization_id=$1 AND property_id=$2 AND status <> 'void'
		ORDER BY created_at DESC
	`, person.OrganizationID, propertyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "drafts_read_failed", "No fue posible consultar los contratos.")
		return
	}
	drafts := make([]leaseContractDraftResponse, 0)
	for draftRows.Next() {
		var draft leaseContractDraftResponse
		if err := draftRows.Scan(&draft.ID, &draft.ApplicationID, &draft.DraftVersion, &draft.TemplateVersion,
			&draft.LegalBasisVersion, &draft.Status, &draft.PDFDocumentID, &draft.DOCXDocumentID, &draft.SignedDocumentID,
			&draft.PDFSHA256, &draft.DOCXSHA256, &draft.LegalReviewedBy, &draft.CreatedAt); err != nil {
			draftRows.Close()
			writeError(w, http.StatusInternalServerError, "drafts_read_failed", "No fue posible consultar los contratos.")
			return
		}
		drafts = append(drafts, draft)
	}
	draftRows.Close()
	writeJSON(w, http.StatusOK, map[string]any{"applications": applications, "drafts": drafts})
}

func (app *application) createTenantApplication(w http.ResponseWriter, r *http.Request) {
	person := currentPrincipal(r)
	propertyID := chi.URLParam(r, "propertyID")
	var input struct {
		FullName            string `json:"full_name"`
		Identifier          string `json:"identifier"`
		Nationality         string `json:"nationality"`
		MaritalStatus       string `json:"marital_status"`
		Profession          string `json:"profession"`
		CurrentAddress      string `json:"current_address"`
		Email               string `json:"email"`
		Phone               string `json:"phone"`
		Employer            string `json:"employer"`
		MonthlyIncomeMinor  *int64 `json:"monthly_income_minor"`
		AuthorizedOccupants string `json:"authorized_occupants"`
		Notes               string `json:"notes"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.FullName, input.Identifier = strings.TrimSpace(input.FullName), strings.ToUpper(strings.TrimSpace(input.Identifier))
	if input.Nationality = strings.TrimSpace(input.Nationality); input.Nationality == "" {
		input.Nationality = "Chilena"
	}
	if len(input.FullName) < 2 || !validChileRUT(input.Identifier) || (input.Email != "" && !validEmail(strings.TrimSpace(input.Email))) || (input.MonthlyIncomeMinor != nil && *input.MonthlyIncomeMinor < 0) {
		writeError(w, http.StatusUnprocessableEntity, "invalid_application", "Revisa el nombre, RUT, correo e ingresos del postulante.")
		return
	}
	var application tenantApplicationResponse
	err := app.db.QueryRow(r.Context(), `
		INSERT INTO tenant_applications (organization_id,property_id,full_name,identifier,nationality,marital_status,
		 profession,current_address,email,phone,employer,monthly_income_minor,authorized_occupants,notes,created_by)
		SELECT $1,id,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15
		FROM properties WHERE id=$2 AND organization_id=$1 AND status <> 'archived'
		RETURNING id,full_name,identifier,nationality,marital_status,profession,current_address,email,phone,employer,
		 monthly_income_minor,authorized_occupants,notes,status,commercial_report_provenance_confirmed,to_char(created_at,'YYYY-MM-DD HH24:MI')
	`, person.OrganizationID, propertyID, input.FullName, input.Identifier, input.Nationality,
		strings.TrimSpace(input.MaritalStatus), strings.TrimSpace(input.Profession), strings.TrimSpace(input.CurrentAddress),
		strings.TrimSpace(input.Email), strings.TrimSpace(input.Phone), strings.TrimSpace(input.Employer), input.MonthlyIncomeMinor,
		strings.TrimSpace(input.AuthorizedOccupants), strings.TrimSpace(input.Notes), person.UserID).Scan(
		&application.ID, &application.FullName, &application.Identifier, &application.Nationality, &application.MaritalStatus,
		&application.Profession, &application.CurrentAddress, &application.Email, &application.Phone, &application.Employer,
		&application.MonthlyIncomeMinor, &application.AuthorizedOccupants, &application.Notes, &application.Status,
		&application.CommercialReportProvenanceConfirmed, &application.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "property_not_found", "La propiedad no existe.")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "application_create_failed", "No fue posible crear la postulación.")
		return
	}
	application.Documents = make([]tenantApplicationDocumentResponse, 0)
	_, _ = app.db.Exec(r.Context(), `INSERT INTO audit_events (organization_id,actor_type,actor_id,action,entity_type,entity_id,new_state) VALUES ($1,'user',$2,'tenant_application.created','tenant_application',$3,$4)`, person.OrganizationID, person.UserID, application.ID, application)
	writeJSON(w, http.StatusCreated, map[string]any{"application": application})
}

func (app *application) linkTenantApplicationDocument(w http.ResponseWriter, r *http.Request) {
	person := currentPrincipal(r)
	applicationID := chi.URLParam(r, "applicationID")
	var input struct {
		DocumentID          string `json:"document_id"`
		DocumentKind        string `json:"document_kind"`
		ProvenanceConfirmed bool   `json:"provenance_confirmed"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	validKinds := map[string]bool{"commercial_report": true, "identity": true, "income_proof": true, "employment": true, "guarantor": true, "insurance": true, "other": true}
	if !validKinds[input.DocumentKind] || strings.TrimSpace(input.DocumentID) == "" {
		writeError(w, http.StatusUnprocessableEntity, "invalid_application_document", "Selecciona un antecedente válido.")
		return
	}
	if input.DocumentKind == "commercial_report" && !input.ProvenanceConfirmed {
		writeError(w, http.StatusUnprocessableEntity, "commercial_report_provenance_required", "Confirma que el informe fue entregado por el postulante o obtenido con autorización.")
		return
	}
	err := pgx.BeginFunc(r.Context(), app.db, func(tx pgx.Tx) error {
		var propertyID string
		if err := tx.QueryRow(r.Context(), `SELECT property_id FROM tenant_applications WHERE id=$1 AND organization_id=$2 FOR UPDATE`, applicationID, person.OrganizationID).Scan(&propertyID); err != nil {
			return err
		}
		var documentExists bool
		if err := tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM documents WHERE id=$1 AND organization_id=$2 AND property_id=$3 AND status <> 'archived')`, input.DocumentID, person.OrganizationID, propertyID).Scan(&documentExists); err != nil || !documentExists {
			if err != nil {
				return err
			}
			return pgx.ErrNoRows
		}
		if _, err := tx.Exec(r.Context(), `INSERT INTO tenant_application_documents (application_id,document_id,document_kind,linked_by) VALUES ($1,$2,$3,$4) ON CONFLICT DO NOTHING`, applicationID, input.DocumentID, input.DocumentKind, person.UserID); err != nil {
			return err
		}
		if input.DocumentKind == "commercial_report" {
			if _, err := tx.Exec(r.Context(), `UPDATE tenant_applications SET commercial_report_provenance_confirmed=true,status='under_review',updated_at=now(),version=version+1 WHERE id=$1`, applicationID); err != nil {
				return err
			}
			if _, err := tx.Exec(r.Context(), `UPDATE documents SET confidentiality='restricted',tags=ARRAY(SELECT DISTINCT tag FROM unnest(tags || ARRAY['antecedente-arrendatario','informe-comercial']) tag),updated_at=now(),version=version+1 WHERE id=$1`, input.DocumentID); err != nil {
				return err
			}
		}
		return audit(r.Context(), tx, person, "tenant_application.document_linked", "tenant_application", applicationID, nil, map[string]any{"document_id": input.DocumentID, "kind": input.DocumentKind})
	})
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "application_or_document_not_found", "La postulación o el documento no pertenece a esta propiedad.")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "application_document_link_failed", "No fue posible asociar el antecedente.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "linked"})
}

func (app *application) createLeaseContractDraft(w http.ResponseWriter, r *http.Request) {
	person := currentPrincipal(r)
	applicationID := chi.URLParam(r, "applicationID")
	var input struct {
		LandlordName            string `json:"landlord_name"`
		LandlordIdentifier      string `json:"landlord_identifier"`
		LandlordNationality     string `json:"landlord_nationality"`
		LandlordMaritalStatus   string `json:"landlord_marital_status"`
		LandlordProfession      string `json:"landlord_profession"`
		LandlordAddress         string `json:"landlord_address"`
		LandlordEmail           string `json:"landlord_email"`
		StartsOn                string `json:"starts_on"`
		EndsOn                  string `json:"ends_on"`
		RentAmountMinor         int64  `json:"rent_amount_minor"`
		PaymentDay              int    `json:"payment_day"`
		DepositMonths           int    `json:"deposit_months"`
		AdjustmentFrequency     int    `json:"adjustment_frequency_months"`
		NoticeDays              int    `json:"notice_days"`
		AdditionalGuarantee     string `json:"additional_guarantee"`
		AdditionalGuaranteeText string `json:"additional_guarantee_text"`
		UtilitiesResponsibility string `json:"utilities_responsibility"`
		SpecialTerms            string `json:"special_terms"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	start, startErr := time.Parse("2006-01-02", input.StartsOn)
	end, endErr := time.Parse("2006-01-02", input.EndsOn)
	input.LandlordName, input.LandlordIdentifier = strings.TrimSpace(input.LandlordName), strings.ToUpper(strings.TrimSpace(input.LandlordIdentifier))
	validAdditional := map[string]bool{"none": true, "co_debtor": true, "insurance": true, "security_check": true}
	if len(input.LandlordName) < 2 || !validChileRUT(input.LandlordIdentifier) || startErr != nil || endErr != nil || end.Before(start) ||
		input.RentAmountMinor <= 0 || input.PaymentDay < 1 || input.PaymentDay > 28 || input.DepositMonths < 1 || input.DepositMonths > 3 ||
		input.AdjustmentFrequency < 1 || input.AdjustmentFrequency > 36 || input.NoticeDays < 30 || input.NoticeDays > 365 || !validAdditional[input.AdditionalGuarantee] {
		writeError(w, http.StatusUnprocessableEntity, "invalid_contract_terms", "Revisa identidades, fechas, renta, garantía, reajuste y aviso.")
		return
	}
	if input.AdditionalGuarantee != "none" && len(strings.TrimSpace(input.AdditionalGuaranteeText)) < 5 {
		writeError(w, http.StatusUnprocessableEntity, "additional_guarantee_details_required", "Describe de manera precisa la garantía adicional.")
		return
	}
	if input.AdditionalGuarantee == "security_check" {
		writeError(w, http.StatusUnprocessableEntity, "security_check_requires_legal_review", "La primera versión no genera cheques en garantía: usa codeudor o seguro, o incorpora esa cláusula después de revisión jurídica individual.")
		return
	}

	var draft leaseContractDraftResponse
	var pdfStorageKey, docxStorageKey string
	err := pgx.BeginFunc(r.Context(), app.db, func(tx pgx.Tx) error {
		if _, err := tx.Exec(r.Context(), `SELECT pg_advisory_xact_lock(hashtext($1))`, applicationID); err != nil {
			return err
		}
		var application tenantApplicationResponse
		var propertyID, propertyName, propertyAddress, commune, region string
		if err := tx.QueryRow(r.Context(), `
			SELECT application.property_id,property.name,property.address_line,property.commune,property.region,
			       application.full_name,application.identifier,application.nationality,application.marital_status,
			       application.profession,application.current_address,application.email,application.phone,
			       application.authorized_occupants
			FROM tenant_applications application
			JOIN properties property ON property.id=application.property_id AND property.organization_id=application.organization_id
			WHERE application.id=$1 AND application.organization_id=$2 AND application.status NOT IN ('rejected','archived')
			FOR UPDATE OF application
		`, applicationID, person.OrganizationID).Scan(&propertyID, &propertyName, &propertyAddress, &commune, &region,
			&application.FullName, &application.Identifier, &application.Nationality, &application.MaritalStatus,
			&application.Profession, &application.CurrentAddress, &application.Email, &application.Phone,
			&application.AuthorizedOccupants); err != nil {
			return err
		}
		var version int
		if err := tx.QueryRow(r.Context(), `SELECT COALESCE(MAX(draft_version),0)+1 FROM lease_contract_drafts WHERE application_id=$1`, applicationID).Scan(&version); err != nil {
			return err
		}
		var commercialReportAttached bool
		if err := tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM tenant_application_documents WHERE application_id=$1 AND document_kind='commercial_report')`, applicationID).Scan(&commercialReportAttached); err != nil {
			return err
		}
		reportStatus := "No adjunto al momento de generar este borrador"
		if commercialReportAttached {
			reportStatus = "Adjunto al expediente confidencial del postulante"
		}
		landlordNationality := strings.TrimSpace(input.LandlordNationality)
		if landlordNationality == "" {
			landlordNationality = "Chilena"
		}
		snapshot := leaseContractSnapshot{
			DraftVersion: version, GeneratedAt: time.Now().UTC(), Organization: person.Organization,
			PropertyName: propertyName, PropertyAddress: propertyAddress, PropertyCommune: commune, PropertyRegion: region,
			LandlordName: input.LandlordName, LandlordIdentifier: input.LandlordIdentifier,
			LandlordNationality: landlordNationality, LandlordMaritalStatus: strings.TrimSpace(input.LandlordMaritalStatus),
			LandlordProfession: strings.TrimSpace(input.LandlordProfession), LandlordAddress: strings.TrimSpace(input.LandlordAddress), LandlordEmail: strings.TrimSpace(input.LandlordEmail),
			TenantName: application.FullName, TenantIdentifier: application.Identifier, TenantNationality: application.Nationality,
			TenantMaritalStatus: application.MaritalStatus, TenantProfession: application.Profession, TenantAddress: application.CurrentAddress,
			TenantEmail: application.Email, TenantPhone: application.Phone, AuthorizedOccupants: application.AuthorizedOccupants,
			StartsOn: input.StartsOn, EndsOn: input.EndsOn, RentAmountMinor: input.RentAmountMinor, PaymentDay: input.PaymentDay,
			DepositMonths: input.DepositMonths, DepositAmountMinor: int64(input.DepositMonths) * input.RentAmountMinor,
			AdjustmentFrequency: input.AdjustmentFrequency, NoticeDays: input.NoticeDays,
			AdditionalGuarantee: input.AdditionalGuarantee, AdditionalGuaranteeText: strings.TrimSpace(input.AdditionalGuaranteeText),
			UtilitiesResponsibility: strings.TrimSpace(input.UtilitiesResponsibility), SpecialTerms: strings.TrimSpace(input.SpecialTerms),
			CommercialReportStatus: reportStatus, TemplateVersion: leaseContractTemplateVersion, LegalBasisVersion: leaseContractLegalVersion,
		}
		pdfBytes, err := BuildLeaseContractDraftPDF(snapshot)
		if err != nil {
			return err
		}
		docxBytes, err := BuildLeaseContractDraftDOCX(snapshot)
		if err != nil {
			return err
		}
		pdfHashBytes, docxHashBytes := sha256.Sum256(pdfBytes), sha256.Sum256(docxBytes)
		pdfHash, docxHash := hex.EncodeToString(pdfHashBytes[:]), hex.EncodeToString(docxHashBytes[:])
		pdfStorageKey = fmt.Sprintf("organizations/%s/properties/%s/contracts/%s.pdf", person.OrganizationID, propertyID, randomHex(16))
		docxStorageKey = fmt.Sprintf("organizations/%s/properties/%s/contracts/%s.docx", person.OrganizationID, propertyID, randomHex(16))
		if err := app.storage.Put(r.Context(), pdfStorageKey, bytes.NewReader(pdfBytes), int64(len(pdfBytes)), "application/pdf"); err != nil {
			return err
		}
		if err := app.storage.Put(r.Context(), docxStorageKey, bytes.NewReader(docxBytes), int64(len(docxBytes)), "application/vnd.openxmlformats-officedocument.wordprocessingml.document"); err != nil {
			return err
		}
		var pdfDocumentID, docxDocumentID string
		displayName := fmt.Sprintf("Borrador contrato de arriendo - %s - v%d", application.FullName, version)
		if err := tx.QueryRow(r.Context(), `
			INSERT INTO documents (organization_id,property_id,document_type,display_name,original_name,storage_key,mime_type,size_bytes,sha256,tags,status,confidentiality,uploaded_by)
			VALUES ($1,$2,'lease',$3,$4,$5,'application/pdf',$6,$7,ARRAY['contrato','borrador','versionado'],'pending_review','restricted',$8)
			RETURNING id
		`, person.OrganizationID, propertyID, displayName, fmt.Sprintf("contrato-borrador-v%d.pdf", version), pdfStorageKey, len(pdfBytes), pdfHash, person.UserID).Scan(&pdfDocumentID); err != nil {
			return err
		}
		if err := tx.QueryRow(r.Context(), `
			INSERT INTO documents (organization_id,property_id,document_type,display_name,original_name,storage_key,mime_type,size_bytes,sha256,tags,status,confidentiality,uploaded_by)
			VALUES ($1,$2,'lease',$3,$4,$5,'application/vnd.openxmlformats-officedocument.wordprocessingml.document',$6,$7,ARRAY['contrato','borrador','editable','notaria','versionado'],'pending_review','restricted',$8)
			RETURNING id
		`, person.OrganizationID, propertyID, displayName+" - editable para notaría", fmt.Sprintf("contrato-borrador-v%d-editable.docx", version), docxStorageKey, len(docxBytes), docxHash, person.UserID).Scan(&docxDocumentID); err != nil {
			return err
		}
		snapshotJSON, err := json.Marshal(snapshot)
		if err != nil {
			return err
		}
		if err := tx.QueryRow(r.Context(), `
			INSERT INTO lease_contract_drafts (organization_id,property_id,application_id,draft_version,template_version,legal_basis_version,snapshot,pdf_document_id,docx_document_id,pdf_sha256,docx_sha256,created_by)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
			RETURNING id,application_id,draft_version,template_version,legal_basis_version,status,pdf_document_id,docx_document_id,COALESCE(signed_document_id::text,''),pdf_sha256,docx_sha256,COALESCE(legal_reviewed_by,''),to_char(created_at,'YYYY-MM-DD HH24:MI')
		`, person.OrganizationID, propertyID, applicationID, version, leaseContractTemplateVersion, leaseContractLegalVersion, snapshotJSON, pdfDocumentID, docxDocumentID, pdfHash, docxHash, person.UserID).Scan(
			&draft.ID, &draft.ApplicationID, &draft.DraftVersion, &draft.TemplateVersion, &draft.LegalBasisVersion,
			&draft.Status, &draft.PDFDocumentID, &draft.DOCXDocumentID, &draft.SignedDocumentID, &draft.PDFSHA256, &draft.DOCXSHA256, &draft.LegalReviewedBy, &draft.CreatedAt); err != nil {
			return err
		}
		if _, err := tx.Exec(r.Context(), `UPDATE tenant_applications SET status='under_review',updated_at=now(),version=version+1 WHERE id=$1`, applicationID); err != nil {
			return err
		}
		return audit(r.Context(), tx, person, "lease_contract.draft_created", "lease_contract_draft", draft.ID, nil, draft)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "application_not_found", "La postulación no existe.")
		return
	}
	if err != nil {
		if pdfStorageKey != "" {
			_ = app.storage.Delete(r.Context(), pdfStorageKey)
		}
		if docxStorageKey != "" {
			_ = app.storage.Delete(r.Context(), docxStorageKey)
		}
		app.logger.Error("create lease contract draft", "error", err)
		writeError(w, http.StatusInternalServerError, "contract_draft_failed", "No fue posible generar el borrador.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"draft": draft})
}

func (app *application) reviewLeaseContractDraft(w http.ResponseWriter, r *http.Request) {
	person := currentPrincipal(r)
	draftID := chi.URLParam(r, "draftID")
	var input struct {
		ReviewerName string `json:"reviewer_name"`
		Approved     bool   `json:"approved"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.ReviewerName = strings.TrimSpace(input.ReviewerName)
	if len(input.ReviewerName) < 3 {
		writeError(w, http.StatusUnprocessableEntity, "reviewer_required", "Indica quién revisó jurídicamente el borrador.")
		return
	}
	status := "reviewed"
	if input.Approved {
		status = "approved_for_signature"
	}
	command, err := app.db.Exec(r.Context(), `UPDATE lease_contract_drafts SET status=$1,legal_reviewed_by=$2,legal_reviewed_at=now(),updated_at=now() WHERE id=$3 AND organization_id=$4 AND status IN ('draft','reviewed','approved_for_signature')`, status, input.ReviewerName, draftID, person.OrganizationID)
	if err != nil || command.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "draft_not_found", "El borrador no está disponible para revisión.")
		return
	}
	_, _ = app.db.Exec(r.Context(), `INSERT INTO audit_events (organization_id,actor_type,actor_id,action,entity_type,entity_id,new_state) VALUES ($1,'user',$2,'lease_contract.reviewed','lease_contract_draft',$3,$4)`, person.OrganizationID, person.UserID, draftID, map[string]any{"status": status, "reviewer": input.ReviewerName})
	writeJSON(w, http.StatusOK, map[string]string{"status": status})
}

func (app *application) attachSignedLeaseContract(w http.ResponseWriter, r *http.Request) {
	person := currentPrincipal(r)
	draftID := chi.URLParam(r, "draftID")
	var input struct {
		DocumentID            string `json:"document_id"`
		VerificationConfirmed bool   `json:"verification_confirmed"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if !input.VerificationConfirmed {
		writeError(w, http.StatusUnprocessableEntity, "signed_contract_verification_required", "Confirma que revisaste identidad, firmas, certificación notarial, páginas y anexos.")
		return
	}
	err := pgx.BeginFunc(r.Context(), app.db, func(tx pgx.Tx) error {
		var applicationID string
		var exists bool
		if err := tx.QueryRow(r.Context(), `
			SELECT draft.application_id,EXISTS(SELECT 1 FROM documents document WHERE document.id=$2 AND document.organization_id=draft.organization_id AND document.property_id=draft.property_id AND document.document_type='lease' AND document.mime_type='application/pdf' AND document.status <> 'archived')
			FROM lease_contract_drafts draft WHERE draft.id=$1 AND draft.organization_id=$3 AND draft.status='approved_for_signature' FOR UPDATE
		`, draftID, input.DocumentID, person.OrganizationID).Scan(&applicationID, &exists); err != nil {
			return err
		}
		if !exists {
			return pgx.ErrNoRows
		}
		if _, err := tx.Exec(r.Context(), `UPDATE lease_contract_drafts SET signed_document_id=$2,status='signed_notarized',notarized_verified_by=$3,notarized_verified_at=now(),updated_at=now() WHERE id=$1`, draftID, input.DocumentID, person.UserID); err != nil {
			return err
		}
		if _, err := tx.Exec(r.Context(), `UPDATE documents SET tags=ARRAY(SELECT DISTINCT tag FROM unnest(tags || ARRAY['contrato','firmado','notariado']) tag),confidentiality='restricted',updated_at=now(),version=version+1 WHERE id=$1`, input.DocumentID); err != nil {
			return err
		}
		if _, err := tx.Exec(r.Context(), `UPDATE tenant_applications SET status='contracted',updated_at=now(),version=version+1 WHERE id=$1`, applicationID); err != nil {
			return err
		}
		return audit(r.Context(), tx, person, "lease_contract.signed_notarized_attached", "lease_contract_draft", draftID, nil, map[string]any{"document_id": input.DocumentID})
	})
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "draft_or_document_not_found", "Aprueba primero el borrador y sube un PDF de contrato de la misma propiedad.")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "signed_contract_attach_failed", "No fue posible registrar el contrato firmado.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "signed_notarized"})
}

// BuildLeaseContractDraftPDF creates the immutable printable representation of a contract draft.
func BuildLeaseContractDraftPDF(snapshot leaseContractSnapshot) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "Letter", "")
	pdf.SetMargins(24, 23, 24)
	pdf.SetAutoPageBreak(true, 22)
	tr := pdf.UnicodeTranslatorFromDescriptor("")
	pdf.SetHeaderFunc(func() {
		pdf.SetFont("Helvetica", "B", 8)
		pdf.SetTextColor(151, 51, 42)
		pdf.CellFormat(0, 5, tr(fmt.Sprintf("BORRADOR - NO FIRMADO - NO VIGENTE | VERSION %d", snapshot.DraftVersion)), "", 1, "C", false, 0, "")
		pdf.SetDrawColor(205, 164, 159)
		pdf.Line(24, 31, 192, 31)
		pdf.Ln(5)
	})
	pdf.SetFooterFunc(func() {
		pdf.SetY(-15)
		pdf.SetFont("Helvetica", "", 7)
		pdf.SetTextColor(105, 105, 105)
		pdf.CellFormat(0, 4, tr(fmt.Sprintf("Plantilla %s | Base %s | Pagina %d", snapshot.TemplateVersion, snapshot.LegalBasisVersion, pdf.PageNo())), "", 0, "C", false, 0, "")
	})
	pdf.AddPage()
	pdf.SetTextColor(27, 47, 36)
	pdf.SetFont("Helvetica", "B", 18)
	pdf.MultiCell(0, 9, tr("CONTRATO DE ARRENDAMIENTO DE INMUEBLE DESTINADO A HABITACION"), "", "C", false)
	pdf.Ln(5)
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(80, 91, 84)
	pdf.MultiCell(0, 5, tr("Documento generado para revisión de las partes y de un profesional jurídico chileno. Sólo la versión íntegramente firmada y formalizada reemplaza este borrador."), "", "C", false)
	pdf.Ln(7)

	heading := func(title string) {
		if pdf.GetY() > 235 {
			pdf.AddPage()
		}
		pdf.Ln(3)
		pdf.SetFont("Helvetica", "B", 11)
		pdf.SetTextColor(24, 72, 48)
		pdf.MultiCell(0, 6, tr(title), "", "L", false)
		pdf.SetTextColor(30, 38, 33)
	}
	paragraph := func(text string) {
		pdf.SetFont("Helvetica", "", 9)
		pdf.MultiCell(0, 5.2, tr(text), "", "J", false)
		pdf.Ln(1.5)
	}

	heading("PRIMERO: COMPARECIENTES")
	paragraph(fmt.Sprintf("Comparece como arrendador don/doña %s, RUT %s, nacionalidad %s, estado civil %s, profesión u oficio %s, domiciliado/a en %s, correo electrónico %s, en adelante el Arrendador.", fallback(snapshot.LandlordName), fallback(snapshot.LandlordIdentifier), fallback(snapshot.LandlordNationality), fallback(snapshot.LandlordMaritalStatus), fallback(snapshot.LandlordProfession), fallback(snapshot.LandlordAddress), fallback(snapshot.LandlordEmail)))
	paragraph(fmt.Sprintf("Comparece como arrendatario don/doña %s, RUT %s, nacionalidad %s, estado civil %s, profesión u oficio %s, domiciliado/a actualmente en %s, correo %s, teléfono %s, en adelante el Arrendatario.", fallback(snapshot.TenantName), fallback(snapshot.TenantIdentifier), fallback(snapshot.TenantNationality), fallback(snapshot.TenantMaritalStatus), fallback(snapshot.TenantProfession), fallback(snapshot.TenantAddress), fallback(snapshot.TenantEmail), fallback(snapshot.TenantPhone)))

	heading("SEGUNDO: INMUEBLE Y DESTINO")
	paragraph(fmt.Sprintf("El Arrendador entrega en arrendamiento el inmueble denominado %s, ubicado en %s, comuna de %s, Región de %s. Se destinará exclusivamente a habitación del Arrendatario y de los siguientes ocupantes autorizados: %s. El inventario, fotografías, lectura de medidores, llaves y estado de entrega deberán incorporarse en un anexo firmado por ambas partes.", snapshot.PropertyName, snapshot.PropertyAddress, snapshot.PropertyCommune, snapshot.PropertyRegion, fallback(snapshot.AuthorizedOccupants)))

	heading("TERCERO: VIGENCIA, ENTREGA Y RESTITUCION")
	paragraph(fmt.Sprintf("La vigencia se extenderá desde el %s hasta el %s. La renovación, prórroga o término deberá constar por un medio verificable. El aviso contractual se realizará con al menos %d días de anticipación, sin reducir los plazos o procedimientos imperativos que resulten aplicables. Al término, el Arrendatario deberá restituir materialmente el inmueble, entregar todas las llaves y acreditar el estado de las cuentas de servicios.", formatContractDate(snapshot.StartsOn), formatContractDate(snapshot.EndsOn), snapshot.NoticeDays))

	heading("CUARTO: RENTA, PAGO Y REAJUSTE")
	paragraph(fmt.Sprintf("La renta mensual será de %s, pagadera por mes anticipado hasta el día %d de cada mes mediante transferencia al medio informado formalmente por el Arrendador. Cada pago deberá individualizar período, monto, fecha y referencia. La renta se reajustará cada %d meses conforme a la variación del Índice de Precios al Consumidor, sin aplicación retroactiva y dejando constancia escrita del nuevo monto.", formatCLP(snapshot.RentAmountMinor), snapshot.PaymentDay, snapshot.AdjustmentFrequency))

	heading("QUINTO: GARANTIA")
	paragraph(fmt.Sprintf("El Arrendatario entregará una garantía en dinero equivalente a %d mes(es) de renta, esto es %s, separada del primer mes de arriendo. La garantía cauciona rentas, servicios, gastos y daños imputables acreditados, pero no podrá usarse unilateralmente como pago del último mes. Terminada la relación y restituido el inmueble, el Arrendador liquidará la garantía de forma documentada, identificando cada descuento y devolviendo el saldo que corresponda dentro del plazo pactado por las partes.", snapshot.DepositMonths, formatCLP(snapshot.DepositAmountMinor)))
	if snapshot.AdditionalGuarantee == "co_debtor" {
		paragraph("Garantía adicional - codeudor solidario: " + snapshot.AdditionalGuaranteeText + ". Su comparecencia, identificación, obligaciones y firma deberán incorporarse antes de aprobar el documento para firma.")
	} else if snapshot.AdditionalGuarantee == "insurance" {
		paragraph("Garantía adicional - seguro: " + snapshot.AdditionalGuaranteeText + ". La cobertura dependerá exclusivamente de la póliza, sus condiciones, vigencia, exclusiones y aceptación de la aseguradora.")
	}

	heading("SEXTO: SERVICIOS, GASTOS Y RESPALDOS")
	paragraph(fallbackWith(snapshot.UtilitiesResponsibility, "El Arrendatario será responsable del pago oportuno de agua potable, electricidad, gas, derechos de aseo y gastos comunes asociados a su ocupación. Deberá conservar y exhibir los comprobantes cuando sean requeridos razonablemente. Cualquier repactación, convenio o cambio de titularidad que pueda afectar al inmueble requerirá autorización previa y escrita del Arrendador."))

	heading("SEPTIMO: CONSERVACION, REPARACIONES Y HABITABILIDAD")
	paragraph("El Arrendatario deberá cuidar el inmueble, informar oportunamente daños o incidentes y responder por deterioros imputables a su hecho, culpa o la de sus ocupantes. El Arrendador mantendrá el inmueble en estado de servir para el fin convenido y ejecutará las reparaciones que legalmente le correspondan. Las reparaciones urgentes, hechos de fuerza mayor y afectaciones de habitabilidad se documentarán y resolverán conforme al contrato y la legislación aplicable.")

	heading("OCTAVO: INCUMPLIMIENTOS Y RESTITUCION JUDICIAL")
	paragraph("Constituirán incumplimientos relevantes, según su gravedad y acreditación, el no pago de renta o cuentas asumidas; el subarriendo o cesión no autorizados; el cambio de destino; los daños imputables; actividades ilícitas; la negativa injustificada a permitir inspecciones previamente coordinadas; y la falta de restitución al término. La terminación, cobro y restitución se ejercerán mediante notificaciones y procedimientos legales aplicables. Este contrato no autoriza cortar suministros, cambiar cerraduras, ingresar por la fuerza ni retirar bienes sin autorización legal o judicial.")

	heading("NOVENO: NOTIFICACIONES Y EVIDENCIA")
	paragraph("Las partes fijarán domicilios y correos para comunicaciones. Los avisos relevantes deberán conservar fecha, contenido, remitente, destinatario y constancia de entrega. El portal de Control de Propiedades podrá almacenar comprobantes, incidentes y documentos como respaldo privado, sin sustituir las notificaciones notariales, judiciales o por carta certificada que la ley o la póliza exijan.")

	heading("DECIMO: ANTECEDENTES Y SEGURO")
	paragraph("Estado del informe comercial en el expediente: " + snapshot.CommercialReportStatus + ". Dicho antecedente es confidencial, no forma parte del contrato entregado al arrendatario y no constituye por sí solo una garantía. Si se contrata un seguro, las partes deberán verificar que este contrato cumpla la duración, forma, pagos al día, avisos y demás condiciones exigidas por la póliza.")

	heading("MARCO LEGAL Y COMPETENCIA")
	paragraph("El contrato se regirá por el Código Civil, la Ley N° 18.101 sobre arrendamiento de predios urbanos, sus modificaciones -incluida la Ley N° 21.461- y las demás normas chilenas aplicables. Las controversias serán conocidas por los tribunales ordinarios competentes. No se incorpora cláusula arbitral en este borrador.")

	if strings.TrimSpace(snapshot.SpecialTerms) != "" {
		heading("UNDECIMO: CONDICIONES PARTICULARES SUJETAS A REVISION")
		paragraph(snapshot.SpecialTerms)
	}

	heading("CONTROL PREVIO A FIRMA")
	paragraph("Antes de firmar deben verificarse: título que habilita al Arrendador; identidad y capacidad de todos los comparecientes; fechas y montos; anexos e inventario; garantía adicional; póliza si existe; ausencia de espacios en blanco; numeración completa de páginas; y revisión jurídica de las cláusulas particulares. Las firmas y el título deberán presentarse al notario conforme corresponda.")

	pdf.Ln(8)
	pdf.SetFont("Helvetica", "", 9)
	pdf.CellFormat(82, 6, tr("________________________________"), "", 0, "C", false, 0, "")
	pdf.CellFormat(82, 6, tr("________________________________"), "", 1, "C", false, 0, "")
	pdf.CellFormat(82, 5, tr("ARRENDADOR: "+snapshot.LandlordName), "", 0, "C", false, 0, "")
	pdf.CellFormat(82, 5, tr("ARRENDATARIO: "+snapshot.TenantName), "", 1, "C", false, 0, "")
	pdf.CellFormat(82, 5, tr("RUT: "+snapshot.LandlordIdentifier), "", 0, "C", false, 0, "")
	pdf.CellFormat(82, 5, tr("RUT: "+snapshot.TenantIdentifier), "", 1, "C", false, 0, "")

	var output bytes.Buffer
	if err := pdf.Output(&output); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

type contractSection struct {
	Heading string
	Body    string
}

func leaseContractSections(snapshot leaseContractSnapshot) []contractSection {
	sections := []contractSection{
		{"PRIMERO: COMPARECIENTES", fmt.Sprintf("Comparece como arrendador don/doña %s, RUT %s, nacionalidad %s, estado civil %s, profesión u oficio %s, domiciliado/a en %s, correo electrónico %s, en adelante el Arrendador. Comparece como arrendatario don/doña %s, RUT %s, nacionalidad %s, estado civil %s, profesión u oficio %s, domiciliado/a actualmente en %s, correo %s, teléfono %s, en adelante el Arrendatario.", fallback(snapshot.LandlordName), fallback(snapshot.LandlordIdentifier), fallback(snapshot.LandlordNationality), fallback(snapshot.LandlordMaritalStatus), fallback(snapshot.LandlordProfession), fallback(snapshot.LandlordAddress), fallback(snapshot.LandlordEmail), fallback(snapshot.TenantName), fallback(snapshot.TenantIdentifier), fallback(snapshot.TenantNationality), fallback(snapshot.TenantMaritalStatus), fallback(snapshot.TenantProfession), fallback(snapshot.TenantAddress), fallback(snapshot.TenantEmail), fallback(snapshot.TenantPhone))},
		{"SEGUNDO: INMUEBLE Y DESTINO", fmt.Sprintf("El Arrendador entrega en arrendamiento el inmueble denominado %s, ubicado en %s, comuna de %s, Región de %s. Se destinará exclusivamente a habitación del Arrendatario y de los siguientes ocupantes autorizados: %s. El inventario, fotografías, lectura de medidores, llaves y estado de entrega deberán incorporarse en un anexo firmado por ambas partes.", snapshot.PropertyName, snapshot.PropertyAddress, snapshot.PropertyCommune, snapshot.PropertyRegion, fallback(snapshot.AuthorizedOccupants))},
		{"TERCERO: VIGENCIA, ENTREGA Y RESTITUCIÓN", fmt.Sprintf("La vigencia se extenderá desde el %s hasta el %s. La renovación, prórroga o término deberá constar por un medio verificable. El aviso contractual se realizará con al menos %d días de anticipación, sin reducir los plazos o procedimientos imperativos que resulten aplicables. Al término, el Arrendatario deberá restituir materialmente el inmueble, entregar todas las llaves y acreditar el estado de las cuentas de servicios.", formatContractDate(snapshot.StartsOn), formatContractDate(snapshot.EndsOn), snapshot.NoticeDays)},
		{"CUARTO: RENTA, PAGO Y REAJUSTE", fmt.Sprintf("La renta mensual será de %s, pagadera por mes anticipado hasta el día %d de cada mes mediante transferencia al medio informado formalmente por el Arrendador. Cada pago deberá individualizar período, monto, fecha y referencia. La renta se reajustará cada %d meses conforme a la variación del Índice de Precios al Consumidor, sin aplicación retroactiva y dejando constancia escrita del nuevo monto.", formatCLP(snapshot.RentAmountMinor), snapshot.PaymentDay, snapshot.AdjustmentFrequency)},
		{"QUINTO: GARANTÍA", fmt.Sprintf("El Arrendatario entregará una garantía en dinero equivalente a %d mes(es) de renta, esto es %s, separada del primer mes de arriendo. La garantía cauciona rentas, servicios, gastos y daños imputables acreditados, pero no podrá usarse unilateralmente como pago del último mes. Terminada la relación y restituido el inmueble, el Arrendador liquidará la garantía de forma documentada, identificando cada descuento y devolviendo el saldo que corresponda dentro del plazo pactado por las partes.", snapshot.DepositMonths, formatCLP(snapshot.DepositAmountMinor))},
		{"SEXTO: SERVICIOS, GASTOS Y RESPALDOS", fallbackWith(snapshot.UtilitiesResponsibility, "El Arrendatario será responsable del pago oportuno de agua potable, electricidad, gas, derechos de aseo y gastos comunes asociados a su ocupación. Deberá conservar y exhibir los comprobantes cuando sean requeridos razonablemente. Cualquier repactación, convenio o cambio de titularidad que pueda afectar al inmueble requerirá autorización previa y escrita del Arrendador.")},
		{"SÉPTIMO: CONSERVACIÓN, REPARACIONES Y HABITABILIDAD", "El Arrendatario deberá cuidar el inmueble, informar oportunamente daños o incidentes y responder por deterioros imputables a su hecho, culpa o la de sus ocupantes. El Arrendador mantendrá el inmueble en estado de servir para el fin convenido y ejecutará las reparaciones que legalmente le correspondan. Las reparaciones urgentes, hechos de fuerza mayor y afectaciones de habitabilidad se documentarán y resolverán conforme al contrato y la legislación aplicable."},
		{"OCTAVO: INCUMPLIMIENTOS Y RESTITUCIÓN JUDICIAL", "Constituirán incumplimientos relevantes, según su gravedad y acreditación, el no pago de renta o cuentas asumidas; el subarriendo o cesión no autorizados; el cambio de destino; los daños imputables; actividades ilícitas; la negativa injustificada a permitir inspecciones previamente coordinadas; y la falta de restitución al término. La terminación, cobro y restitución se ejercerán mediante notificaciones y procedimientos legales aplicables. Este contrato no autoriza cortar suministros, cambiar cerraduras, ingresar por la fuerza ni retirar bienes sin autorización legal o judicial."},
		{"NOVENO: NOTIFICACIONES Y EVIDENCIA", "Las partes fijarán domicilios y correos para comunicaciones. Los avisos relevantes deberán conservar fecha, contenido, remitente, destinatario y constancia de entrega. El portal de Control de Propiedades podrá almacenar comprobantes, incidentes y documentos como respaldo privado, sin sustituir las notificaciones notariales, judiciales o por carta certificada que la ley o la póliza exijan."},
		{"DÉCIMO: ANTECEDENTES Y SEGURO", "Estado del informe comercial en el expediente: " + snapshot.CommercialReportStatus + ". Dicho antecedente es confidencial, no forma parte del contrato entregado al arrendatario y no constituye por sí solo una garantía. Si se contrata un seguro, las partes deberán verificar que este contrato cumpla la duración, forma, pagos al día, avisos y demás condiciones exigidas por la póliza."},
		{"MARCO LEGAL Y COMPETENCIA", "El contrato se regirá por el Código Civil, la Ley N° 18.101 sobre arrendamiento de predios urbanos, sus modificaciones -incluida la Ley N° 21.461- y las demás normas chilenas aplicables. Las controversias serán conocidas por los tribunales ordinarios competentes. No se incorpora cláusula arbitral en este borrador."},
	}
	if snapshot.AdditionalGuarantee == "co_debtor" {
		sections = append(sections, contractSection{"GARANTÍA ADICIONAL: CODEUDOR SOLIDARIO", snapshot.AdditionalGuaranteeText + ". Su comparecencia, identificación, obligaciones y firma deberán incorporarse antes de aprobar el documento para firma."})
	} else if snapshot.AdditionalGuarantee == "insurance" {
		sections = append(sections, contractSection{"GARANTÍA ADICIONAL: SEGURO", snapshot.AdditionalGuaranteeText + ". La cobertura dependerá exclusivamente de la póliza, sus condiciones, vigencia, exclusiones y aceptación de la aseguradora."})
	}
	if strings.TrimSpace(snapshot.SpecialTerms) != "" {
		sections = append(sections, contractSection{"CONDICIONES PARTICULARES SUJETAS A REVISIÓN", snapshot.SpecialTerms})
	}
	sections = append(sections, contractSection{"CONTROL PREVIO A FIRMA", "Antes de firmar deben verificarse: título que habilita al Arrendador; identidad y capacidad de todos los comparecientes; fechas y montos; anexos e inventario; garantía adicional; póliza si existe; ausencia de espacios en blanco; numeración completa de páginas; y revisión jurídica de las cláusulas particulares. Las firmas y el título deberán presentarse al notario conforme corresponda."})
	return sections
}

// BuildLeaseContractDraftDOCX creates an editable Word document for legal and notarial review.
func BuildLeaseContractDraftDOCX(snapshot leaseContractSnapshot) ([]byte, error) {
	var output bytes.Buffer
	archive := zip.NewWriter(&output)
	addFile := func(name, content string) error {
		file, err := archive.Create(name)
		if err != nil {
			return err
		}
		_, err = file.Write([]byte(content))
		return err
	}

	contentTypes := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
 <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
 <Default Extension="xml" ContentType="application/xml"/>
 <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
 <Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/>
 <Override PartName="/word/settings.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.settings+xml"/>
 <Override PartName="/word/header1.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.header+xml"/>
 <Override PartName="/word/footer1.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.footer+xml"/>
 <Override PartName="/docProps/core.xml" ContentType="application/vnd.openxmlformats-package.core-properties+xml"/>
 <Override PartName="/docProps/app.xml" ContentType="application/vnd.openxmlformats-officedocument.extended-properties+xml"/>
</Types>`
	rootRels := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/><Relationship Id="rId2" Type="http://schemas.openxmlformats.org/package/2006/relationships/metadata/core-properties" Target="docProps/core.xml"/><Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/extended-properties" Target="docProps/app.xml"/></Relationships>`
	documentRels := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/><Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/settings" Target="settings.xml"/><Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/header" Target="header1.xml"/><Relationship Id="rId4" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/footer" Target="footer1.xml"/></Relationships>`
	styles := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
 <w:docDefaults><w:rPrDefault><w:rPr><w:rFonts w:ascii="Calibri" w:hAnsi="Calibri"/><w:sz w:val="22"/><w:szCs w:val="22"/><w:color w:val="000000"/></w:rPr></w:rPrDefault><w:pPrDefault><w:pPr><w:spacing w:before="0" w:after="120" w:line="264" w:lineRule="auto"/></w:pPr></w:pPrDefault></w:docDefaults>
 <w:style w:type="paragraph" w:default="1" w:styleId="Normal"><w:name w:val="Normal"/><w:qFormat/><w:pPr><w:jc w:val="both"/><w:spacing w:before="0" w:after="120" w:line="264" w:lineRule="auto"/></w:pPr><w:rPr><w:rFonts w:ascii="Calibri" w:hAnsi="Calibri"/><w:sz w:val="22"/><w:color w:val="000000"/></w:rPr></w:style>
 <w:style w:type="paragraph" w:styleId="ContractTitle"><w:name w:val="Contract Title"/><w:basedOn w:val="Normal"/><w:qFormat/><w:pPr><w:jc w:val="center"/><w:spacing w:before="0" w:after="160"/><w:keepNext/></w:pPr><w:rPr><w:rFonts w:ascii="Calibri" w:hAnsi="Calibri"/><w:b/><w:sz w:val="32"/><w:color w:val="000000"/></w:rPr></w:style>
 <w:style w:type="paragraph" w:styleId="Heading1"><w:name w:val="heading 1"/><w:basedOn w:val="Normal"/><w:next w:val="Normal"/><w:qFormat/><w:pPr><w:keepNext/><w:keepLines/><w:spacing w:before="240" w:after="120"/></w:pPr><w:rPr><w:rFonts w:ascii="Calibri" w:hAnsi="Calibri"/><w:b/><w:sz w:val="26"/><w:color w:val="000000"/></w:rPr></w:style>
 <w:style w:type="paragraph" w:styleId="DraftNotice"><w:name w:val="Draft Notice"/><w:basedOn w:val="Normal"/><w:pPr><w:jc w:val="center"/><w:spacing w:before="0" w:after="120"/></w:pPr><w:rPr><w:rFonts w:ascii="Calibri" w:hAnsi="Calibri"/><w:b/><w:color w:val="9B1C1C"/><w:sz w:val="20"/></w:rPr></w:style>
 <w:style w:type="paragraph" w:styleId="Signature"><w:name w:val="Signature"/><w:basedOn w:val="Normal"/><w:pPr><w:jc w:val="left"/><w:spacing w:before="160" w:after="80"/></w:pPr><w:rPr><w:rFonts w:ascii="Calibri" w:hAnsi="Calibri"/><w:sz w:val="22"/><w:color w:val="000000"/></w:rPr></w:style>
</w:styles>`
	settings := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><w:settings xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:zoom w:percent="100"/><w:defaultTabStop w:val="720"/><w:displayBackgroundShape/></w:settings>`
	header := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><w:hdr xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:p><w:pPr><w:jc w:val="center"/><w:spacing w:after="80"/></w:pPr><w:r><w:rPr><w:b/><w:color w:val="9B1C1C"/><w:sz w:val="18"/></w:rPr><w:t>` + wordXMLEscape(fmt.Sprintf("BORRADOR EDITABLE PARA REVISIÓN Y NOTARÍA - VERSIÓN %d", snapshot.DraftVersion)) + `</w:t></w:r></w:p></w:hdr>`
	footer := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><w:ftr xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:p><w:pPr><w:jc w:val="center"/></w:pPr><w:r><w:rPr><w:color w:val="666666"/><w:sz w:val="16"/></w:rPr><w:t>` + wordXMLEscape("Plantilla "+snapshot.TemplateVersion+" | Página ") + `</w:t></w:r><w:r><w:fldChar w:fldCharType="begin"/></w:r><w:r><w:instrText xml:space="preserve"> PAGE </w:instrText></w:r><w:r><w:fldChar w:fldCharType="end"/></w:r></w:p></w:ftr>`

	paragraph := func(style, text string) string {
		return `<w:p><w:pPr><w:pStyle w:val="` + style + `"/></w:pPr><w:r><w:t xml:space="preserve">` + wordXMLEscape(text) + `</w:t></w:r></w:p>`
	}
	var body strings.Builder
	body.WriteString(paragraph("ContractTitle", "CONTRATO DE ARRENDAMIENTO DE INMUEBLE DESTINADO A HABITACIÓN"))
	body.WriteString(paragraph("DraftNotice", "BORRADOR - NO FIRMADO - NO VIGENTE"))
	body.WriteString(paragraph("Normal", "Documento editable generado para revisión de las partes, de un profesional jurídico chileno y de la notaría. Toda modificación debe dar origen a una nueva versión en el portal antes de firmarse."))
	for _, section := range leaseContractSections(snapshot) {
		body.WriteString(paragraph("Heading1", section.Heading))
		body.WriteString(paragraph("Normal", section.Body))
	}
	body.WriteString(paragraph("Heading1", "FIRMAS"))
	body.WriteString(paragraph("Signature", "ARRENDADOR: _______________________________________    RUT: "+snapshot.LandlordIdentifier))
	body.WriteString(paragraph("Signature", "ARRENDATARIO: ____________________________________    RUT: "+snapshot.TenantIdentifier))
	if snapshot.AdditionalGuarantee == "co_debtor" {
		body.WriteString(paragraph("Signature", "CODEUDOR SOLIDARIO: _______________________________    RUT: __________________"))
	}
	body.WriteString(`<w:sectPr><w:headerReference w:type="default" r:id="rId3"/><w:footerReference w:type="default" r:id="rId4"/><w:pgSz w:w="12240" w:h="15840"/><w:pgMar w:top="1440" w:right="1440" w:bottom="1440" w:left="1440" w:header="708" w:footer="708" w:gutter="0"/></w:sectPr>`)
	document := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><w:body>` + body.String() + `</w:body></w:document>`
	created := snapshot.GeneratedAt.UTC().Format(time.RFC3339)
	core := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties" xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:dcterms="http://purl.org/dc/terms/" xmlns:dcmitype="http://purl.org/dc/dcmitype/" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"><dc:title>` + wordXMLEscape("Borrador contrato de arriendo - "+snapshot.PropertyName) + `</dc:title><dc:creator>Control de Propiedades</dc:creator><cp:lastModifiedBy>Control de Propiedades</cp:lastModifiedBy><dcterms:created xsi:type="dcterms:W3CDTF">` + created + `</dcterms:created><dcterms:modified xsi:type="dcterms:W3CDTF">` + created + `</dcterms:modified></cp:coreProperties>`
	appProperties := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Properties xmlns="http://schemas.openxmlformats.org/officeDocument/2006/extended-properties" xmlns:vt="http://schemas.openxmlformats.org/officeDocument/2006/docPropsVTypes"><Application>Control de Propiedades</Application><AppVersion>1.0</AppVersion></Properties>`

	files := []struct{ name, content string }{
		{"[Content_Types].xml", contentTypes}, {"_rels/.rels", rootRels}, {"docProps/core.xml", core}, {"docProps/app.xml", appProperties},
		{"word/document.xml", document}, {"word/styles.xml", styles}, {"word/settings.xml", settings}, {"word/header1.xml", header}, {"word/footer1.xml", footer}, {"word/_rels/document.xml.rels", documentRels},
	}
	for _, file := range files {
		if err := addFile(file.name, file.content); err != nil {
			_ = archive.Close()
			return nil, err
		}
	}
	if err := archive.Close(); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func wordXMLEscape(value string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&apos;").Replace(value)
}

func validChileRUT(value string) bool {
	compact := strings.ToUpper(strings.NewReplacer(".", "", "-", "", " ", "").Replace(value))
	if len(compact) < 2 {
		return false
	}
	number, verifier := compact[:len(compact)-1], compact[len(compact)-1:]
	if !regexp.MustCompile(`^[0-9]{7,8}$`).MatchString(number) || !regexp.MustCompile(`^[0-9K]$`).MatchString(verifier) {
		return false
	}
	sum, multiplier := 0, 2
	for index := len(number) - 1; index >= 0; index-- {
		digit, _ := strconv.Atoi(number[index : index+1])
		sum += digit * multiplier
		multiplier++
		if multiplier > 7 {
			multiplier = 2
		}
	}
	result := 11 - sum%11
	expected := strconv.Itoa(result)
	if result == 11 {
		expected = "0"
	} else if result == 10 {
		expected = "K"
	}
	return verifier == expected
}

func fallback(value string) string {
	if strings.TrimSpace(value) == "" {
		return "[COMPLETAR]"
	}
	return strings.TrimSpace(value)
}

func fallbackWith(value, fallbackValue string) string {
	if strings.TrimSpace(value) == "" {
		return fallbackValue
	}
	return strings.TrimSpace(value)
}

func formatContractDate(value string) string {
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return value
	}
	return parsed.Format("02-01-2006")
}
