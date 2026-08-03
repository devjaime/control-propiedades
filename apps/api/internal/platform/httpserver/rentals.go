package httpserver

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/phpdave11/gofpdf"
	qrcode "github.com/skip2/go-qrcode"
)

type leaseResponse struct {
	ID                  string                     `json:"id"`
	TenantID            string                     `json:"tenant_id"`
	TenantName          string                     `json:"tenant_name"`
	TenantEmail         string                     `json:"tenant_email"`
	StartsOn            string                     `json:"starts_on"`
	EndsOn              string                     `json:"ends_on"`
	Status              string                     `json:"status"`
	RentAmountMinor     int64                      `json:"rent_amount_minor"`
	CurrencyCode        string                     `json:"currency_code"`
	PaymentDay          int16                      `json:"payment_day"`
	PaymentInstallments int16                      `json:"payment_installments"`
	ScheduleKind        string                     `json:"schedule_kind"`
	AgreementNotes      string                     `json:"agreement_notes"`
	Installments        []leaseInstallmentResponse `json:"installments"`
	RenewalNoticeDays   int16                      `json:"renewal_notice_days"`
	RenewalReviewDays   int16                      `json:"renewal_review_days"`
	VacateNoticeMonths  int16                      `json:"vacate_notice_months"`
	VacateNoticeDays    int16                      `json:"vacate_notice_days"`
	AdjustmentMethod    string                     `json:"adjustment_method"`
	AdjustmentFrequency int16                      `json:"adjustment_frequency_months"`
	NextAdjustmentOn    string                     `json:"next_adjustment_on"`
	AdjustmentEffective string                     `json:"adjustment_effective_on"`
	Version             int64                      `json:"version"`
}

type leaseInstallmentResponse struct {
	InstallmentNumber   int16 `json:"installment_number"`
	DueDay              int16 `json:"due_day"`
	ExpectedAmountMinor int64 `json:"expected_amount_minor"`
}

type chargeResponse struct {
	ID                  string `json:"id"`
	Period              string `json:"period"`
	ChargeType          string `json:"charge_type"`
	Concept             string `json:"concept"`
	DueDate             string `json:"due_date"`
	OriginalAmountMinor int64  `json:"original_amount_minor"`
	BalanceMinor        int64  `json:"balance_minor"`
	CurrencyCode        string `json:"currency_code"`
	Status              string `json:"status"`
	Notes               string `json:"notes"`
	Version             int64  `json:"version"`
}

type paymentResponse struct {
	ID                  string `json:"id"`
	ChargeID            string `json:"charge_id"`
	PayerName           string `json:"payer_name"`
	PaymentDate         string `json:"payment_date"`
	AmountMinor         int64  `json:"amount_minor"`
	CurrencyCode        string `json:"currency_code"`
	PaymentMethod       string `json:"payment_method"`
	BankReference       string `json:"bank_reference"`
	Status              string `json:"status"`
	TimingStatus        string `json:"timing_status"`
	Observations        string `json:"observations"`
	DocumentID          string `json:"document_id"`
	CreatedAt           string `json:"created_at"`
	Version             int64  `json:"version"`
	InstallmentNumber   int16  `json:"installment_number"`
	PlannedDueDate      string `json:"planned_due_date"`
	ExpectedAmountMinor int64  `json:"expected_amount_minor"`
}

type notificationResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Title   string `json:"title"`
	Message string `json:"message"`
	DueAt   string `json:"due_at"`
	Status  string `json:"status"`
}

type observationResponse struct {
	ID         string `json:"id"`
	Category   string `json:"category"`
	Content    string `json:"content"`
	ObservedAt string `json:"observed_at"`
	CreatedBy  string `json:"created_by"`
}

type maintenanceScheduleResponse struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Category        string `json:"category"`
	FrequencyMonths int16  `json:"frequency_months"`
	NextDueOn       string `json:"next_due_on"`
	ReminderDays    int16  `json:"reminder_days"`
	LastCompletedOn string `json:"last_completed_on"`
	Notes           string `json:"notes"`
	Status          string `json:"status"`
	Version         int64  `json:"version"`
}

type reviewCalendarItemResponse struct {
	ID          string `json:"id"`
	Period      string `json:"period"`
	Kind        string `json:"kind"`
	Type        string `json:"type"`
	Title       string `json:"title"`
	Message     string `json:"message"`
	ScheduledOn string `json:"scheduled_on"`
	Status      string `json:"status"`
}

type receiptResponse struct {
	ID               string `json:"id"`
	DocumentID       string `json:"document_id"`
	Folio            string `json:"folio"`
	PublicToken      string `json:"public_token"`
	VerificationCode string `json:"verification_code"`
	Status           string `json:"status"`
	IssuedAt         string `json:"issued_at"`
}

type receiptHistoryResponse struct {
	ID          string `json:"id"`
	DocumentID  string `json:"document_id"`
	Folio       string `json:"folio"`
	PublicToken string `json:"public_token"`
	Period      string `json:"period"`
	Status      string `json:"status"`
	IssuedAt    string `json:"issued_at"`
}

func (app *application) createLease(w http.ResponseWriter, r *http.Request) {
	person := currentPrincipal(r)
	propertyID := chi.URLParam(r, "propertyID")
	var input struct {
		TenantName          string `json:"tenant_name"`
		TenantIdentifier    string `json:"tenant_identifier"`
		TenantEmail         string `json:"tenant_email"`
		TenantPhone         string `json:"tenant_phone"`
		StartsOn            string `json:"starts_on"`
		EndsOn              string `json:"ends_on"`
		RentAmountMinor     int64  `json:"rent_amount_minor"`
		PaymentDay          int16  `json:"payment_day"`
		SpecialTerms        string `json:"special_terms"`
		RenewalNoticeDays   int16  `json:"renewal_notice_days"`
		RenewalReviewDays   int16  `json:"renewal_review_days"`
		VacateNoticeMonths  int16  `json:"vacate_notice_months"`
		VacateNoticeDays    int16  `json:"vacate_notice_days"`
		AdjustmentMethod    string `json:"adjustment_method"`
		AdjustmentFrequency int16  `json:"adjustment_frequency_months"`
		NextAdjustmentOn    string `json:"next_adjustment_on"`
		AdjustmentEffective string `json:"adjustment_effective_on"`
		ScheduleKind        string `json:"schedule_kind"`
		AgreementNotes      string `json:"agreement_notes"`
		Installments        []struct {
			DueDay              int16 `json:"due_day"`
			ExpectedAmountMinor int64 `json:"expected_amount_minor"`
		} `json:"installments"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.TenantName = strings.TrimSpace(input.TenantName)
	if input.RenewalNoticeDays == 0 {
		input.RenewalNoticeDays = 60
	}
	if input.RenewalReviewDays == 0 {
		input.RenewalReviewDays = 45
	}
	if input.AdjustmentMethod == "" {
		input.AdjustmentMethod = "none"
	}
	if len(input.Installments) == 0 && input.PaymentDay >= 1 && input.PaymentDay <= 28 {
		input.Installments = append(input.Installments, struct {
			DueDay              int16 `json:"due_day"`
			ExpectedAmountMinor int64 `json:"expected_amount_minor"`
		}{DueDay: input.PaymentDay, ExpectedAmountMinor: input.RentAmountMinor})
	}
	if input.ScheduleKind == "" {
		if len(input.Installments) > 1 {
			input.ScheduleKind = "special_agreement"
		} else {
			input.ScheduleKind = "contractual"
		}
	}
	start, startErr := time.Parse("2006-01-02", input.StartsOn)
	if input.TenantName == "" || startErr != nil || input.RentAmountMinor <= 0 || input.PaymentDay < 1 || input.PaymentDay > 28 || len(input.Installments) < 1 || len(input.Installments) > 12 {
		writeError(w, http.StatusUnprocessableEntity, "invalid_lease", "Completa arrendatario, inicio, renta y calendario de cuotas.")
		return
	}
	var scheduledTotal int64
	seenDays := make(map[int16]bool, len(input.Installments))
	for _, installment := range input.Installments {
		if installment.DueDay < 1 || installment.DueDay > 28 || installment.ExpectedAmountMinor <= 0 || seenDays[installment.DueDay] {
			writeError(w, http.StatusUnprocessableEntity, "invalid_installment_schedule", "Cada cuota debe tener un día distinto entre 1 y 28 y un monto positivo.")
			return
		}
		seenDays[installment.DueDay] = true
		scheduledTotal += installment.ExpectedAmountMinor
	}
	if scheduledTotal != input.RentAmountMinor {
		writeError(w, http.StatusUnprocessableEntity, "installment_total_mismatch", "La suma de las cuotas debe coincidir con la renta mensual.")
		return
	}
	if input.ScheduleKind == "contractual" && (len(input.Installments) != 1 || input.Installments[0].DueDay != input.PaymentDay) {
		writeError(w, http.StatusUnprocessableEntity, "invalid_contractual_schedule", "Sin acuerdo especial, el calendario debe contener un solo pago por la renta completa en el día contractual.")
		return
	}
	if input.ScheduleKind != "contractual" && input.ScheduleKind != "special_agreement" {
		writeError(w, http.StatusUnprocessableEntity, "invalid_schedule_kind", "El tipo de calendario de pago no es válido.")
		return
	}
	if input.ScheduleKind == "special_agreement" && len(strings.TrimSpace(input.AgreementNotes)) < 3 {
		writeError(w, http.StatusUnprocessableEntity, "agreement_notes_required", "Describe el acuerdo especial de pago con el arrendatario.")
		return
	}
	if input.EndsOn != "" {
		if end, err := time.Parse("2006-01-02", input.EndsOn); err != nil || end.Before(start) {
			writeError(w, http.StatusUnprocessableEntity, "invalid_lease_end", "La fecha de término no es válida.")
			return
		}
	}
	if input.RenewalNoticeDays < 0 || input.RenewalNoticeDays > 365 || input.RenewalReviewDays < 0 || input.RenewalReviewDays > 365 || input.VacateNoticeMonths < 0 || input.VacateNoticeMonths > 24 || input.VacateNoticeDays < 0 || input.VacateNoticeDays > 90 {
		writeError(w, http.StatusUnprocessableEntity, "invalid_lease_alerts", "Revisa los plazos de renovación y aviso de restitución.")
		return
	}
	if input.AdjustmentMethod == "ipc" {
		if input.AdjustmentFrequency < 1 || input.AdjustmentFrequency > 36 {
			writeError(w, http.StatusUnprocessableEntity, "invalid_adjustment", "El reajuste IPC necesita una periodicidad entre 1 y 36 meses.")
			return
		}
		if _, err := time.Parse("2006-01-02", input.NextAdjustmentOn); err != nil {
			writeError(w, http.StatusUnprocessableEntity, "invalid_adjustment_date", "Indica la fecha del próximo reajuste IPC.")
			return
		}
		if input.AdjustmentEffective != "" {
			if _, err := time.Parse("2006-01-02", input.AdjustmentEffective); err != nil {
				writeError(w, http.StatusUnprocessableEntity, "invalid_adjustment_effective_date", "La fecha de aplicación del reajuste no es válida.")
				return
			}
		}
	} else if input.AdjustmentMethod != "none" {
		writeError(w, http.StatusUnprocessableEntity, "invalid_adjustment", "El mecanismo de reajuste no es válido.")
		return
	} else {
		input.AdjustmentFrequency = 0
		input.NextAdjustmentOn = ""
		input.AdjustmentEffective = ""
	}

	var lease leaseResponse
	err := pgx.BeginFunc(r.Context(), app.db, func(tx pgx.Tx) error {
		var propertyExists bool
		if err := tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM properties WHERE id=$1 AND organization_id=$2 AND status <> 'archived')`, propertyID, person.OrganizationID).Scan(&propertyExists); err != nil || !propertyExists {
			if err != nil {
				return err
			}
			return pgx.ErrNoRows
		}
		var tenantID string
		if err := tx.QueryRow(r.Context(), `
			INSERT INTO tenants (organization_id, name, identifier, email, phone)
			VALUES ($1,$2,NULLIF($3,''),NULLIF($4,''),NULLIF($5,'')) RETURNING id
		`, person.OrganizationID, input.TenantName, strings.TrimSpace(input.TenantIdentifier), strings.TrimSpace(input.TenantEmail), strings.TrimSpace(input.TenantPhone)).Scan(&tenantID); err != nil {
			return err
		}
		if err := tx.QueryRow(r.Context(), `
			INSERT INTO leases (organization_id, property_id, tenant_id, starts_on, ends_on, status,
				rent_amount_minor, payment_day, payment_installments, special_terms,
				renewal_notice_days,renewal_review_days,vacate_notice_months,vacate_notice_days,adjustment_method,
				adjustment_frequency_months,next_adjustment_on,adjustment_effective_on,created_by,updated_by)
			VALUES ($1,$2,$3,$4,NULLIF($5,'')::date,'active',$6,$7,$8,NULLIF($9,''),$10,$11,$12,$13,$14,
				NULLIF($15,0),NULLIF($16,'')::date,NULLIF($17,'')::date,$18,$18)
			RETURNING id, tenant_id, to_char(starts_on,'YYYY-MM-DD'), COALESCE(to_char(ends_on,'YYYY-MM-DD'),''), status,
				rent_amount_minor, currency_code, payment_day, payment_installments,renewal_notice_days,vacate_notice_months,
				vacate_notice_days,adjustment_method,COALESCE(adjustment_frequency_months,0),COALESCE(to_char(next_adjustment_on,'YYYY-MM-DD'),''),
				renewal_review_days,COALESCE(to_char(adjustment_effective_on,'YYYY-MM-DD'),''),version
		`, person.OrganizationID, propertyID, tenantID, input.StartsOn, input.EndsOn, input.RentAmountMinor, input.PaymentDay, len(input.Installments), strings.TrimSpace(input.SpecialTerms), input.RenewalNoticeDays, input.RenewalReviewDays, input.VacateNoticeMonths, input.VacateNoticeDays, input.AdjustmentMethod, input.AdjustmentFrequency, input.NextAdjustmentOn, input.AdjustmentEffective, person.UserID).Scan(
			&lease.ID, &lease.TenantID, &lease.StartsOn, &lease.EndsOn, &lease.Status, &lease.RentAmountMinor, &lease.CurrencyCode, &lease.PaymentDay, &lease.PaymentInstallments,
			&lease.RenewalNoticeDays, &lease.VacateNoticeMonths, &lease.VacateNoticeDays, &lease.AdjustmentMethod, &lease.AdjustmentFrequency, &lease.NextAdjustmentOn,
			&lease.RenewalReviewDays, &lease.AdjustmentEffective, &lease.Version,
		); err != nil {
			return err
		}
		lease.Installments = make([]leaseInstallmentResponse, 0, len(input.Installments))
		for index, installment := range input.Installments {
			item := leaseInstallmentResponse{InstallmentNumber: int16(index + 1), DueDay: installment.DueDay, ExpectedAmountMinor: installment.ExpectedAmountMinor}
			if _, err := tx.Exec(r.Context(), `INSERT INTO lease_payment_schedules (organization_id,lease_id,installment_number,due_day,expected_amount_minor,schedule_kind,agreement_notes) VALUES ($1,$2,$3,$4,$5,$6,NULLIF($7,''))`, person.OrganizationID, lease.ID, item.InstallmentNumber, item.DueDay, item.ExpectedAmountMinor, input.ScheduleKind, strings.TrimSpace(input.AgreementNotes)); err != nil {
				return err
			}
			lease.Installments = append(lease.Installments, item)
		}
		lease.TenantName, lease.TenantEmail = input.TenantName, strings.TrimSpace(input.TenantEmail)
		lease.ScheduleKind, lease.AgreementNotes = input.ScheduleKind, strings.TrimSpace(input.AgreementNotes)
		return audit(r.Context(), tx, person, "lease.created", "lease", lease.ID, nil, lease)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "property_not_found", "La propiedad no existe.")
		return
	}
	if err != nil {
		writeError(w, http.StatusConflict, "lease_create_failed", "No fue posible crear el arriendo. Verifica que no exista otro contrato vigente.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"lease": lease})
}

func (app *application) createCharge(w http.ResponseWriter, r *http.Request) {
	person := currentPrincipal(r)
	propertyID := chi.URLParam(r, "propertyID")
	var input struct {
		Period      string `json:"period"`
		ChargeType  string `json:"charge_type"`
		Concept     string `json:"concept"`
		DueDate     string `json:"due_date"`
		AmountMinor int64  `json:"amount_minor"`
		Notes       string `json:"notes"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	allowedTypes := map[string]bool{"rent": true, "electricity": true, "water": true, "trash": true, "common_expense": true, "adjustment": true, "other": true}
	if _, err := time.Parse("2006-01", input.Period); err != nil || !allowedTypes[input.ChargeType] || input.AmountMinor <= 0 {
		writeError(w, http.StatusUnprocessableEntity, "invalid_charge", "Revisa período, tipo y monto del cobro.")
		return
	}
	if _, err := time.Parse("2006-01-02", input.DueDate); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid_due_date", "La fecha de vencimiento no es válida.")
		return
	}
	if strings.TrimSpace(input.Concept) == "" {
		input.Concept = chargeTypeLabel(input.ChargeType)
	}
	var charge chargeResponse
	err := pgx.BeginFunc(r.Context(), app.db, func(tx pgx.Tx) error {
		var leaseID, tenantID string
		if err := tx.QueryRow(r.Context(), `SELECT id, tenant_id FROM leases WHERE organization_id=$1 AND property_id=$2 AND status IN ('active','ending')`, person.OrganizationID, propertyID).Scan(&leaseID, &tenantID); err != nil {
			return err
		}
		if err := tx.QueryRow(r.Context(), `
			INSERT INTO charges (organization_id, property_id, lease_id, tenant_id, period, charge_type, concept,
				due_date, original_amount_minor, balance_minor, notes, created_by)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$9,NULLIF($10,''),$11)
			RETURNING id, period, charge_type, concept, to_char(due_date,'YYYY-MM-DD'), original_amount_minor,
				balance_minor, currency_code, status, COALESCE(notes,''), version
		`, person.OrganizationID, propertyID, leaseID, tenantID, input.Period, input.ChargeType, strings.TrimSpace(input.Concept), input.DueDate, input.AmountMinor, strings.TrimSpace(input.Notes), person.UserID).Scan(
			&charge.ID, &charge.Period, &charge.ChargeType, &charge.Concept, &charge.DueDate, &charge.OriginalAmountMinor,
			&charge.BalanceMinor, &charge.CurrencyCode, &charge.Status, &charge.Notes, &charge.Version,
		); err != nil {
			return err
		}
		_, err := tx.Exec(r.Context(), `
			INSERT INTO notifications (organization_id, property_id, notification_type, title, message, entity_type, entity_id, due_at, dedup_key)
			VALUES ($1,$2,'charge_due',$3,$4,'charge',$5,$6::date + interval '9 hours',$7) ON CONFLICT DO NOTHING
		`, person.OrganizationID, propertyID, "Cobro próximo: "+input.Concept, fmt.Sprintf("Revisar %s del período %s.", input.Concept, input.Period), charge.ID, input.DueDate, "charge-due-"+charge.ID)
		if err != nil {
			return err
		}
		return audit(r.Context(), tx, person, "charge.created", "charge", charge.ID, nil, charge)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusConflict, "active_lease_required", "Primero configura el arrendatario y el contrato vigente.")
		return
	}
	if err != nil {
		writeError(w, http.StatusConflict, "charge_create_failed", "Ese tipo de cobro ya existe para el período o los datos no son válidos.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"charge": charge})
}

func (app *application) createPayment(w http.ResponseWriter, r *http.Request) {
	person := currentPrincipal(r)
	propertyID := chi.URLParam(r, "propertyID")
	r.Body = http.MaxBytesReader(w, r.Body, app.options.MaxUploadBytes+1024*1024)
	if err := r.ParseMultipartForm(app.options.MaxUploadBytes + 1024*1024); err != nil {
		writeError(w, http.StatusRequestEntityTooLarge, "upload_too_large", uploadTooLargeMessage(app.options.MaxUploadBytes))
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	amount, err := strconv.ParseInt(strings.TrimSpace(r.FormValue("amount_minor")), 10, 64)
	paymentDate, dateErr := time.Parse("2006-01-02", r.FormValue("payment_date"))
	method := r.FormValue("payment_method")
	allowedMethods := map[string]bool{"bank_transfer": true, "deposit": true, "cash": true, "other": true}
	chargeID := strings.TrimSpace(r.FormValue("charge_id"))
	if err != nil || amount <= 0 || dateErr != nil || !allowedMethods[method] || chargeID == "" {
		writeError(w, http.StatusUnprocessableEntity, "invalid_payment", "Revisa cobro, fecha, monto y medio de pago.")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil || header.Size <= 0 || header.Size > app.options.MaxUploadBytes {
		writeError(w, http.StatusUnprocessableEntity, "payment_proof_required", "Adjunta un comprobante PDF o imagen válido.")
		return
	}
	defer file.Close()
	buffer := make([]byte, 512)
	n, readErr := file.Read(buffer)
	if readErr != nil && !errors.Is(readErr, io.EOF) {
		writeError(w, http.StatusBadRequest, "file_read_failed", "No fue posible leer el archivo.")
		return
	}
	contentType := http.DetectContentType(buffer[:n])
	extensions := map[string]string{"application/pdf": ".pdf", "image/jpeg": ".jpg", "image/png": ".png", "image/webp": ".webp"}
	extension, ok := extensions[contentType]
	if !ok {
		writeError(w, http.StatusUnsupportedMediaType, "unsupported_file", "Solo se permiten PDF, JPG, PNG y WebP.")
		return
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		writeError(w, http.StatusBadRequest, "file_read_failed", "No fue posible procesar el archivo.")
		return
	}
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		writeError(w, http.StatusBadRequest, "file_read_failed", "No fue posible procesar el archivo.")
		return
	}
	fileHash := hex.EncodeToString(hasher.Sum(nil))
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		writeError(w, http.StatusBadRequest, "file_read_failed", "No fue posible procesar el archivo.")
		return
	}
	storageKey := fmt.Sprintf("organizations/%s/properties/%s/payments/%s%s", person.OrganizationID, propertyID, randomHex(16), extension)
	if err := app.storage.Put(r.Context(), storageKey, file, header.Size, contentType); err != nil {
		writeError(w, http.StatusBadGateway, "storage_failed", "No fue posible guardar el comprobante.")
		return
	}

	var payment paymentResponse
	err = pgx.BeginFunc(r.Context(), app.db, func(tx pgx.Tx) error {
		var leaseID, tenantID, tenantName, period string
		var balance int64
		if err := tx.QueryRow(r.Context(), `
			SELECT c.lease_id, c.tenant_id, t.name, c.period, c.balance_minor
			FROM charges c JOIN tenants t ON t.id=c.tenant_id
			WHERE c.id=$1 AND c.organization_id=$2 AND c.property_id=$3 AND c.status NOT IN ('paid','cancelled')
		`, chargeID, person.OrganizationID, propertyID).Scan(&leaseID, &tenantID, &tenantName, &period, &balance); err != nil {
			return err
		}
		if amount > balance {
			return errAllocationTooLarge
		}
		var documentID string
		originalName := filepath.Base(header.Filename)
		if err := tx.QueryRow(r.Context(), `
			INSERT INTO documents (organization_id, property_id, document_type, display_name, original_name, storage_key,
				mime_type, size_bytes, sha256, tags, status, uploaded_by, document_date, period, amount_minor, currency_code,confidentiality)
			VALUES ($1,$2,'bank_receipt',$3,$4,$5,$6,$7,$8,$9,'pending_review',$10,$11,$12,$13,'CLP','restricted') RETURNING id
		`, person.OrganizationID, propertyID, "Comprobante de pago "+period, originalName, storageKey, contentType, header.Size, fileHash,
			[]string{"pago", period}, person.UserID, paymentDate.Format("2006-01-02"), period, amount).Scan(&documentID); err != nil {
			return err
		}
		var existingPayments int
		if err := tx.QueryRow(r.Context(), `SELECT count(*) FROM payments p JOIN payment_allocations a ON a.payment_id=p.id WHERE a.charge_id=$1 AND p.status NOT IN ('rejected','reversed','duplicate')`, chargeID).Scan(&existingPayments); err != nil {
			return err
		}
		var installmentNumber, dueDay int16
		var expectedAmount int64
		if err := tx.QueryRow(r.Context(), `
			SELECT installment_number,due_day,expected_amount_minor
			FROM lease_payment_schedules
			WHERE lease_id=$1 AND installment_number=LEAST($2 + 1,(SELECT payment_installments FROM leases WHERE id=$1))
		`, leaseID, existingPayments).Scan(&installmentNumber, &dueDay, &expectedAmount); err != nil {
			return err
		}
		periodDate, err := time.Parse("2006-01", period)
		if err != nil {
			return err
		}
		plannedDueDate := time.Date(periodDate.Year(), periodDate.Month(), int(dueDay), 0, 0, 0, 0, time.UTC)
		timing := "on_time"
		if paymentDate.After(plannedDueDate) {
			timing = "late"
		}
		payer := tenantName
		if err := tx.QueryRow(r.Context(), `
			INSERT INTO payments (organization_id, property_id, lease_id, tenant_id, payer_name, payment_date, amount_minor,
				payment_method, bank_reference, supporting_document_id, timing_status, observations, registered_by,
				planned_installment_number,planned_due_date)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NULLIF($9,''),$10,$11,NULLIF($12,''),$13,$14,$15)
			RETURNING id, to_char(payment_date,'YYYY-MM-DD'), currency_code, status, to_char(created_at,'YYYY-MM-DD HH24:MI'), version
		`, person.OrganizationID, propertyID, leaseID, tenantID, payer, paymentDate, amount, method, strings.TrimSpace(r.FormValue("bank_reference")), documentID, timing, strings.TrimSpace(r.FormValue("observations")), person.UserID, installmentNumber, plannedDueDate).Scan(
			&payment.ID, &payment.PaymentDate, &payment.CurrencyCode, &payment.Status, &payment.CreatedAt, &payment.Version,
		); err != nil {
			return err
		}
		payment.ChargeID, payment.PayerName, payment.AmountMinor, payment.PaymentMethod = chargeID, payer, amount, method
		payment.BankReference, payment.TimingStatus, payment.Observations, payment.DocumentID = strings.TrimSpace(r.FormValue("bank_reference")), timing, strings.TrimSpace(r.FormValue("observations")), documentID
		payment.InstallmentNumber, payment.PlannedDueDate, payment.ExpectedAmountMinor = installmentNumber, plannedDueDate.Format("2006-01-02"), expectedAmount
		if _, err := tx.Exec(r.Context(), `INSERT INTO payment_allocations (organization_id,payment_id,charge_id,amount_minor) VALUES ($1,$2,$3,$4)`, person.OrganizationID, payment.ID, chargeID, amount); err != nil {
			return err
		}
		if _, err := tx.Exec(r.Context(), `
			INSERT INTO notifications (organization_id, property_id, notification_type, title, message, entity_type, entity_id, due_at, dedup_key)
			VALUES ($1,$2,'payment_review','Pago por revisar',$3,'payment',$4,now(),$5)
		`, person.OrganizationID, propertyID, fmt.Sprintf("%s registró un abono de %s para %s.", payer, formatCLP(amount), period), payment.ID, "payment-review-"+payment.ID); err != nil {
			return err
		}
		return audit(r.Context(), tx, person, "payment.registered", "payment", payment.ID, nil, payment)
	})
	if err != nil {
		_ = app.storage.Delete(r.Context(), storageKey)
		if errors.Is(err, errAllocationTooLarge) {
			writeError(w, http.StatusConflict, "payment_exceeds_balance", "El abono supera el saldo pendiente del cobro.")
			return
		}
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "charge_not_found", "El cobro no existe o ya está pagado.")
			return
		}
		writeError(w, http.StatusInternalServerError, "payment_create_failed", "No fue posible registrar el pago.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"payment": payment})
}

var errAllocationTooLarge = errors.New("allocation exceeds balance")
var errPaymentNotPending = errors.New("payment is not pending")

func (app *application) confirmPayment(w http.ResponseWriter, r *http.Request) {
	person := currentPrincipal(r)
	paymentID := chi.URLParam(r, "paymentID")
	if person.Role != "organization_admin" && person.Role != "property_manager" {
		writeError(w, http.StatusForbidden, "confirmation_forbidden", "No tienes permisos para confirmar pagos.")
		return
	}
	var input struct {
		Version int64 `json:"version"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	var receipt *receiptResponse
	var receiptStorageKey string
	err := pgx.BeginFunc(r.Context(), app.db, func(tx pgx.Tx) error {
		var organizationID, propertyID, leaseID, tenantID, chargeID, paymentStatus, allocationStatus string
		var payerName, tenantName, propertyName, propertyAddress, propertyCommune, period, chargeType, method, paymentDate string
		var amount, balance, originalAmount int64
		var version int64
		if err := tx.QueryRow(r.Context(), `
			SELECT p.organization_id,p.property_id,p.lease_id,p.tenant_id,a.charge_id,p.status,a.status,p.payer_name,t.name,
				pr.name,pr.address_line,pr.commune,c.period,c.charge_type,p.payment_method,to_char(p.payment_date,'YYYY-MM-DD'),
				p.amount_minor,c.balance_minor,c.original_amount_minor,p.version
			FROM payments p JOIN payment_allocations a ON a.payment_id=p.id JOIN charges c ON c.id=a.charge_id
			JOIN tenants t ON t.id=p.tenant_id JOIN properties pr ON pr.id=p.property_id
			WHERE p.id=$1 AND p.organization_id=$2 FOR UPDATE OF p,c,a
		`, paymentID, person.OrganizationID).Scan(&organizationID, &propertyID, &leaseID, &tenantID, &chargeID, &paymentStatus, &allocationStatus, &payerName, &tenantName, &propertyName, &propertyAddress, &propertyCommune, &period, &chargeType, &method, &paymentDate, &amount, &balance, &originalAmount, &version); err != nil {
			return err
		}
		if paymentStatus != "pending_reconciliation" || allocationStatus != "proposed" || version != input.Version {
			return errPaymentNotPending
		}
		if amount > balance {
			return errAllocationTooLarge
		}
		newBalance := balance - amount
		newStatus := "partially_paid"
		if newBalance == 0 {
			newStatus = "paid"
		}
		if _, err := tx.Exec(r.Context(), `UPDATE payments SET status='reconciled',confirmed_by=$2,confirmed_at=now(),updated_at=now(),version=version+1 WHERE id=$1`, paymentID, person.UserID); err != nil {
			return err
		}
		if _, err := tx.Exec(r.Context(), `UPDATE payment_allocations SET status='confirmed',confirmed_by=$2,confirmed_at=now() WHERE payment_id=$1`, paymentID, person.UserID); err != nil {
			return err
		}
		if _, err := tx.Exec(r.Context(), `UPDATE charges SET balance_minor=$2,status=$3,updated_at=now(),version=version+1 WHERE id=$1`, chargeID, newBalance, newStatus); err != nil {
			return err
		}
		if _, err := tx.Exec(r.Context(), `UPDATE documents SET status='verified',verified_at=now(),verified_by=$2,updated_by=$2,updated_at=now(),version=version+1 WHERE id=(SELECT supporting_document_id FROM payments WHERE id=$1)`, paymentID, person.UserID); err != nil {
			return err
		}
		if _, err := tx.Exec(r.Context(), `UPDATE notifications SET status='read',read_at=now() WHERE organization_id=$1 AND entity_type='payment' AND entity_id=$2 AND status='unread'`, person.OrganizationID, paymentID); err != nil {
			return err
		}
		if err := audit(r.Context(), tx, person, "payment.confirmed", "payment", paymentID, map[string]any{"status": paymentStatus}, map[string]any{"status": "reconciled", "charge_balance_minor": newBalance}); err != nil {
			return err
		}
		if chargeType == "rent" && newBalance == 0 {
			paymentLines := make([]receiptPaymentLine, 0)
			rows, err := tx.Query(r.Context(), `
				SELECT COALESCE(p.planned_installment_number,1),to_char(p.payment_date,'YYYY-MM-DD'),p.amount_minor,p.payment_method,COALESCE(p.bank_reference,'')
				FROM payments p JOIN payment_allocations a ON a.payment_id=p.id
				WHERE a.charge_id=$1 AND p.status='reconciled' ORDER BY planned_installment_number,payment_date,p.created_at
			`, chargeID)
			if err != nil {
				return err
			}
			for rows.Next() {
				var line receiptPaymentLine
				if err := rows.Scan(&line.InstallmentNumber, &line.PaymentDate, &line.AmountMinor, &line.PaymentMethod, &line.BankReference); err != nil {
					rows.Close()
					return err
				}
				paymentLines = append(paymentLines, line)
			}
			rows.Close()
			issued, storageKey, err := app.issueRentReceipt(r, tx, person, receiptIssueData{OrganizationID: organizationID, PropertyID: propertyID, LeaseID: leaseID, TenantID: tenantID, ChargeID: chargeID, TenantName: tenantName, PropertyName: propertyName, PropertyAddress: propertyAddress, PropertyCommune: propertyCommune, Period: period, AmountMinor: originalAmount, PaymentLines: paymentLines})
			if err != nil {
				return err
			}
			receipt, receiptStorageKey = issued, storageKey
		}
		return nil
	})
	if err != nil {
		if receiptStorageKey != "" {
			_ = app.storage.Delete(r.Context(), receiptStorageKey)
		}
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "payment_not_found", "El pago no existe.")
			return
		}
		if errors.Is(err, errPaymentNotPending) {
			writeError(w, http.StatusConflict, "payment_conflict", "El pago ya fue revisado o cambió en otra sesión.")
			return
		}
		if errors.Is(err, errAllocationTooLarge) {
			writeError(w, http.StatusConflict, "allocation_conflict", "El saldo del cobro cambió. Revisa antes de confirmar.")
			return
		}
		app.logger.Error("confirm payment", "error", err)
		writeError(w, http.StatusInternalServerError, "payment_confirm_failed", "No fue posible confirmar el pago.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "reconciled", "receipt": receipt})
}

type receiptIssueData struct {
	OrganizationID, PropertyID, LeaseID, TenantID, ChargeID, TenantName, PropertyName, PropertyAddress, PropertyCommune, Period string
	AmountMinor                                                                                                                 int64
	PaymentLines                                                                                                                []receiptPaymentLine
}

type receiptPaymentLine struct {
	InstallmentNumber int16  `json:"installment_number"`
	PaymentDate       string `json:"payment_date"`
	AmountMinor       int64  `json:"amount_minor"`
	PaymentMethod     string `json:"payment_method"`
	BankReference     string `json:"bank_reference,omitempty"`
}

func (app *application) issueRentReceipt(r *http.Request, tx pgx.Tx, person principal, data receiptIssueData) (*receiptResponse, string, error) {
	year, _ := strconv.Atoi(data.Period[:4])
	var sequence int64
	if err := tx.QueryRow(r.Context(), `
		INSERT INTO receipt_sequences (organization_id,sequence_year,last_value) VALUES ($1,$2,1)
		ON CONFLICT (organization_id,sequence_year) DO UPDATE SET last_value=receipt_sequences.last_value+1 RETURNING last_value
	`, person.OrganizationID, year).Scan(&sequence); err != nil {
		return nil, "", err
	}
	folio := fmt.Sprintf("CP-%d-%06d", year, sequence)
	token, verificationCode := randomHex(24), strings.ToUpper(randomHex(4))
	verificationURL := strings.TrimSuffix(app.options.PublicWebURL, "/") + "/verificar/comprobante/" + token
	signedAt := time.Now()
	snapshot := map[string]any{"folio": folio, "organization": person.Organization, "landlord": person.Name, "tenant": data.TenantName, "property": data.PropertyName, "address": data.PropertyAddress, "commune": data.PropertyCommune, "period": data.Period, "payment_lines": data.PaymentLines, "amount_minor": data.AmountMinor, "total_received_minor": data.AmountMinor, "balance_minor": int64(0), "payment_status": "paid", "currency_code": "CLP", "verification_code": verificationCode, "verification_url": verificationURL, "signature_name": person.Name, "signed_at": signedAt}
	pdfBytes, err := buildRentReceiptPDF(snapshot, verificationURL)
	if err != nil {
		return nil, "", err
	}
	hash := sha256.Sum256(pdfBytes)
	hashText := hex.EncodeToString(hash[:])
	storageKey := fmt.Sprintf("organizations/%s/properties/%s/receipts/%s.pdf", person.OrganizationID, data.PropertyID, token)
	if err := app.storage.Put(r.Context(), storageKey, bytes.NewReader(pdfBytes), int64(len(pdfBytes)), "application/pdf"); err != nil {
		return nil, "", err
	}
	var documentID string
	if err := tx.QueryRow(r.Context(), `
		INSERT INTO documents (organization_id,property_id,document_type,display_name,original_name,storage_key,mime_type,size_bytes,sha256,tags,status,uploaded_by,document_date,period,amount_minor,currency_code,verified_at,verified_by,updated_by)
		VALUES ($1,$2,'rent_receipt',$3,$4,$5,'application/pdf',$6,$7,$8,'verified',$9,current_date,$10,$11,'CLP',now(),$9,$9) RETURNING id
	`, person.OrganizationID, data.PropertyID, "Comprobante de arriendo "+data.Period, folio+".pdf", storageKey, len(pdfBytes), hashText, []string{"arriendo", "comprobante", data.Period}, person.UserID, data.Period, data.AmountMinor).Scan(&documentID); err != nil {
		_ = app.storage.Delete(r.Context(), storageKey)
		return nil, "", err
	}
	snapshotJSON, _ := json.Marshal(snapshot)
	var receipt receiptResponse
	if err := tx.QueryRow(r.Context(), `
		INSERT INTO rent_receipts (organization_id,property_id,lease_id,tenant_id,charge_id,document_id,folio,public_token,verification_code,snapshot,pdf_sha256,template_version,signature_name,signed_at,issued_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,'rent-receipt-v1.3-cl',$12,$13,$14)
		RETURNING id,document_id,folio,public_token,verification_code,status,to_char(issued_at,'YYYY-MM-DD HH24:MI')
	`, person.OrganizationID, data.PropertyID, data.LeaseID, data.TenantID, data.ChargeID, documentID, folio, token, verificationCode, snapshotJSON, hashText, person.Name, signedAt, person.UserID).Scan(&receipt.ID, &receipt.DocumentID, &receipt.Folio, &receipt.PublicToken, &receipt.VerificationCode, &receipt.Status, &receipt.IssuedAt); err != nil {
		_ = app.storage.Delete(r.Context(), storageKey)
		return nil, "", err
	}
	_, err = tx.Exec(r.Context(), `INSERT INTO notifications (organization_id,property_id,notification_type,title,message,entity_type,entity_id,due_at,dedup_key) VALUES ($1,$2,'rent_completed','Arriendo pagado',$3,'receipt',$4,now(),$5)`, person.OrganizationID, data.PropertyID, "Se emitió el comprobante "+folio+" para "+data.Period+".", receipt.ID, "rent-completed-"+data.ChargeID)
	if err != nil {
		_ = app.storage.Delete(r.Context(), storageKey)
		return nil, "", err
	}
	if err := audit(r.Context(), tx, person, "receipt.issued", "rent_receipt", receipt.ID, nil, receipt); err != nil {
		_ = app.storage.Delete(r.Context(), storageKey)
		return nil, "", err
	}
	return &receipt, storageKey, nil
}

func buildRentReceiptPDF(snapshot map[string]any, verificationURL string) ([]byte, error) {
	qrPNG, err := qrcode.Encode(verificationURL, qrcode.Medium, 256)
	if err != nil {
		return nil, err
	}
	pdf := gofpdf.New("P", "mm", "A4", "")
	tr := pdf.UnicodeTranslatorFromDescriptor("")
	pdf.SetTitle("Comprobante de pago de arriendo "+fmt.Sprint(snapshot["folio"]), false)
	pdf.SetMargins(18, 18, 18)
	pdf.AddPage()
	pdf.SetTextColor(23, 35, 29)
	pdf.SetFont("Helvetica", "B", 20)
	pdf.CellFormat(0, 12, tr("Comprobante de pago de arriendo"), "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)
	pdf.SetTextColor(36, 107, 75)
	pdf.CellFormat(0, 7, "VERIFICADO POR CONTROL DE PROPIEDADES", "", 1, "L", false, 0, "")
	if reviewerName := strings.TrimSpace(fmt.Sprint(snapshot["reviewer_name"])); reviewerName != "" && reviewerName != "<nil>" {
		reviewerRole := strings.TrimSpace(fmt.Sprint(snapshot["reviewer_role"]))
		pdf.CellFormat(0, 6, tr("REVISIÓN DOCUMENTAL: "+reviewerName+" - "+reviewerRole), "", 1, "L", false, 0, "")
	}
	pdf.Ln(4)
	pdf.SetTextColor(23, 35, 29)
	pdf.SetFont("Helvetica", "", 11)
	amount := snapshot["amount_minor"].(int64)
	propertyLabel := fmt.Sprintf("%s, %s", snapshot["property"], snapshot["commune"])
	if address := strings.TrimSpace(fmt.Sprint(snapshot["address"])); address != "" && address != "<nil>" {
		propertyLabel = fmt.Sprintf("%s - %s, %s", snapshot["property"], address, snapshot["commune"])
	}
	rows := [][2]string{{"Folio", fmt.Sprint(snapshot["folio"])}, {"Emisor", fmt.Sprint(snapshot["organization"])}, {"Arrendador", fmt.Sprint(snapshot["landlord"])}, {"Arrendatario", fmt.Sprint(snapshot["tenant"])}, {"Propiedad", propertyLabel}, {"Período", fmt.Sprint(snapshot["period"])}, {"Renta del período", formatCLP(amount)}, {"Total recibido", formatCLP(amount)}, {"Saldo registrado", formatCLP(0)}, {"Estado", "Renta íntegramente pagada según registros conciliados"}, {"Código", fmt.Sprint(snapshot["verification_code"])}}
	for _, row := range rows {
		pdf.SetFont("Helvetica", "B", 10)
		pdf.CellFormat(45, 8, tr(row[0]), "B", 0, "L", false, 0, "")
		pdf.SetFont("Helvetica", "", 10)
		pdf.CellFormat(0, 8, tr(row[1]), "B", 1, "L", false, 0, "")
	}
	pdf.Ln(5)
	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(0, 7, tr("Abonos confirmados"), "", 1, "L", false, 0, "")
	if paymentLines, ok := snapshot["payment_lines"].([]receiptPaymentLine); ok {
		for _, line := range paymentLines {
			reference := ""
			if line.BankReference != "" {
				reference = " · Ref. " + line.BankReference
			}
			pdf.SetFont("Helvetica", "", 9)
			pdf.CellFormat(0, 6, tr(fmt.Sprintf("Cuota %d · %s · %s · %s%s", line.InstallmentNumber, line.PaymentDate, formatCLP(line.AmountMinor), paymentMethodLabel(line.PaymentMethod), reference)), "", 1, "L", false, 0, "")
		}
	}
	if pdf.GetY() > 220 {
		pdf.AddPage()
	}
	pdf.Ln(7)
	pdf.SetFont("Helvetica", "I", 10)
	pdf.MultiCell(125, 6, tr("Firma electrónica simple: "+fmt.Sprint(snapshot["signature_name"])+" · "+snapshot["signed_at"].(time.Time).Format("02-01-2006 15:04")), "", "L", false)
	pdf.SetFont("Helvetica", "", 8)
	pdf.SetTextColor(88, 100, 94)
	pdf.MultiCell(125, 4.5, tr("Constancia privada de recepción. No es un documento tributario ni altera por sí sola el contrato, sus reajustes, intereses, reparaciones u otros acuerdos entre las partes."), "", "L", false)
	if reviewerName := strings.TrimSpace(fmt.Sprint(snapshot["reviewer_name"])); reviewerName != "" && reviewerName != "<nil>" {
		pdf.MultiCell(125, 4.5, tr("La revisión documental registra el cotejo de los pagos y respaldos indicados; no constituye auditoría externa, fe pública ni certificación tributaria."), "", "L", false)
	}
	pdf.RegisterImageOptionsReader("receipt-qr", gofpdf.ImageOptions{ImageType: "PNG", ReadDpi: true}, bytes.NewReader(qrPNG))
	qrY := pdf.GetY() + 4
	pdf.ImageOptions("receipt-qr", 155, qrY, 35, 35, false, gofpdf.ImageOptions{ImageType: "PNG", ReadDpi: true}, 0, "")
	pdf.SetY(qrY + 40)
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(88, 100, 94)
	pdf.MultiCell(0, 5, tr("Escanea el QR para validar folio, estado e integridad del comprobante. La validación pública no expone RUT, datos bancarios ni la dirección completa."), "", "L", false)
	var output bytes.Buffer
	if err := pdf.Output(&output); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

// RebuildRentReceiptPDF regenerates a stored receipt with a corrected public verification URL.
func RebuildRentReceiptPDF(snapshotJSON []byte, verificationURL string, repairedAt time.Time) ([]byte, []byte, error) {
	var stored struct {
		Folio            string               `json:"folio"`
		Organization     string               `json:"organization"`
		Landlord         string               `json:"landlord"`
		Tenant           string               `json:"tenant"`
		Property         string               `json:"property"`
		Address          string               `json:"address"`
		Commune          string               `json:"commune"`
		Period           string               `json:"period"`
		PaymentLines     []receiptPaymentLine `json:"payment_lines"`
		AmountMinor      int64                `json:"amount_minor"`
		CurrencyCode     string               `json:"currency_code"`
		VerificationCode string               `json:"verification_code"`
		SignatureName    string               `json:"signature_name"`
		SignedAt         time.Time            `json:"signed_at"`
		ReviewerName     string               `json:"reviewer_name"`
		ReviewerRole     string               `json:"reviewer_role"`
		ReviewedAt       *time.Time           `json:"reviewed_at"`
	}
	if err := json.Unmarshal(snapshotJSON, &stored); err != nil {
		return nil, nil, err
	}
	snapshot := map[string]any{
		"folio": stored.Folio, "organization": stored.Organization, "landlord": stored.Landlord,
		"tenant": stored.Tenant, "property": stored.Property, "address": stored.Address, "commune": stored.Commune,
		"period": stored.Period, "payment_lines": stored.PaymentLines, "amount_minor": stored.AmountMinor,
		"currency_code": stored.CurrencyCode, "verification_code": stored.VerificationCode,
		"verification_url": verificationURL, "signature_name": stored.SignatureName, "signed_at": stored.SignedAt,
		"reviewer_name": stored.ReviewerName, "reviewer_role": stored.ReviewerRole, "reviewed_at": stored.ReviewedAt,
		"qr_repaired_at": repairedAt, "qr_repair_reason": "public_url_correction",
	}
	pdfBytes, err := buildRentReceiptPDF(snapshot, verificationURL)
	if err != nil {
		return nil, nil, err
	}
	updatedSnapshot, err := json.Marshal(snapshot)
	if err != nil {
		return nil, nil, err
	}
	return pdfBytes, updatedSnapshot, nil
}

// BuildRentReceiptPDF renders a receipt from its canonical JSON snapshot.
func BuildRentReceiptPDF(snapshotJSON []byte, verificationURL string) ([]byte, error) {
	var stored struct {
		Folio            string               `json:"folio"`
		Organization     string               `json:"organization"`
		Landlord         string               `json:"landlord"`
		Tenant           string               `json:"tenant"`
		Property         string               `json:"property"`
		Address          string               `json:"address"`
		Commune          string               `json:"commune"`
		Period           string               `json:"period"`
		PaymentLines     []receiptPaymentLine `json:"payment_lines"`
		AmountMinor      int64                `json:"amount_minor"`
		CurrencyCode     string               `json:"currency_code"`
		VerificationCode string               `json:"verification_code"`
		SignatureName    string               `json:"signature_name"`
		SignedAt         time.Time            `json:"signed_at"`
		ReviewerName     string               `json:"reviewer_name"`
		ReviewerRole     string               `json:"reviewer_role"`
		ReviewedAt       *time.Time           `json:"reviewed_at"`
	}
	if err := json.Unmarshal(snapshotJSON, &stored); err != nil {
		return nil, err
	}
	snapshot := map[string]any{
		"folio": stored.Folio, "organization": stored.Organization, "landlord": stored.Landlord,
		"tenant": stored.Tenant, "property": stored.Property, "address": stored.Address, "commune": stored.Commune,
		"period": stored.Period, "payment_lines": stored.PaymentLines, "amount_minor": stored.AmountMinor,
		"currency_code": stored.CurrencyCode, "verification_code": stored.VerificationCode,
		"verification_url": verificationURL, "signature_name": stored.SignatureName, "signed_at": stored.SignedAt,
		"reviewer_name": stored.ReviewerName, "reviewer_role": stored.ReviewerRole, "reviewed_at": stored.ReviewedAt,
	}
	return buildRentReceiptPDF(snapshot, verificationURL)
}

func (app *application) ensureMonthlyRentCharge(r *http.Request, person principal, propertyID, period string) error {
	periodDate, err := time.Parse("2006-01", period)
	if err != nil {
		return err
	}
	return pgx.BeginFunc(r.Context(), app.db, func(tx pgx.Tx) error {
		var leaseID, tenantID string
		var rentAmount int64
		var paymentInstallments, contractualPaymentDay int16
		var startsOn time.Time
		var endsOn *time.Time
		if err := tx.QueryRow(r.Context(), `
			SELECT id,tenant_id,rent_amount_minor,payment_installments,payment_day,starts_on,ends_on
			FROM leases WHERE organization_id=$1 AND property_id=$2 AND status IN ('active','ending')
		`, person.OrganizationID, propertyID).Scan(&leaseID, &tenantID, &rentAmount, &paymentInstallments, &contractualPaymentDay, &startsOn, &endsOn); err != nil {
			return err
		}
		monthStart := time.Date(periodDate.Year(), periodDate.Month(), 1, 0, 0, 0, 0, time.UTC)
		leaseStart := time.Date(startsOn.Year(), startsOn.Month(), 1, 0, 0, 0, 0, time.UTC)
		if monthStart.Before(leaseStart) || (endsOn != nil && monthStart.After(time.Date(endsOn.Year(), endsOn.Month(), 1, 0, 0, 0, 0, time.UTC))) {
			return nil
		}
		dueDate := time.Date(periodDate.Year(), periodDate.Month(), int(contractualPaymentDay), 0, 0, 0, 0, time.UTC)
		var chargeID string
		inserted := true
		err := tx.QueryRow(r.Context(), `
			INSERT INTO charges (organization_id,property_id,lease_id,tenant_id,period,charge_type,concept,due_date,
				original_amount_minor,balance_minor,origin,notes,created_by)
			VALUES ($1,$2,$3,$4,$5,'rent',$6,$7,$8,$8,'automatic','Generado desde el calendario del contrato',$9)
			ON CONFLICT (lease_id,period,charge_type) DO NOTHING RETURNING id
		`, person.OrganizationID, propertyID, leaseID, tenantID, period, "Arriendo "+period, dueDate, rentAmount, person.UserID).Scan(&chargeID)
		if errors.Is(err, pgx.ErrNoRows) {
			inserted = false
			err = tx.QueryRow(r.Context(), `SELECT id FROM charges WHERE lease_id=$1 AND period=$2 AND charge_type='rent'`, leaseID, period).Scan(&chargeID)
		}
		if err != nil {
			return err
		}
		rows, err := tx.Query(r.Context(), `SELECT installment_number,due_day,expected_amount_minor,schedule_kind FROM lease_payment_schedules WHERE lease_id=$1 ORDER BY installment_number`, leaseID)
		if err != nil {
			return err
		}
		type scheduledInstallment struct {
			number   int
			day      int
			expected int64
			kind     string
		}
		schedule := make([]scheduledInstallment, 0, paymentInstallments)
		for rows.Next() {
			var item scheduledInstallment
			if err := rows.Scan(&item.number, &item.day, &item.expected, &item.kind); err != nil {
				return err
			}
			schedule = append(schedule, item)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return err
		}
		rows.Close()
		for _, item := range schedule {
			installmentDue := time.Date(periodDate.Year(), periodDate.Month(), item.day, 9, 0, 0, 0, time.UTC)
			title := "Pago contractual del arriendo"
			message := fmt.Sprintf("Revisar el pago contractual por %s, exigible hasta el día %d.", formatCLP(item.expected), item.day)
			if item.kind == "special_agreement" {
				title = fmt.Sprintf("Acuerdo especial de pago · cuota %d", item.number)
				message = fmt.Sprintf("Revisar cuota %d del acuerdo especial por %s, programada para el día %d. La obligación contractual original vence el día %d.", item.number, formatCLP(item.expected), item.day, contractualPaymentDay)
			}
			if _, err := tx.Exec(r.Context(), `
				INSERT INTO notifications (organization_id,property_id,notification_type,title,message,entity_type,entity_id,due_at,dedup_key)
				VALUES ($1,$2,'charge_due',$3,$4,'charge',$5,$6,$7) ON CONFLICT DO NOTHING
			`, person.OrganizationID, propertyID, title, message, chargeID, installmentDue, fmt.Sprintf("rent-installment-%s-%d", chargeID, item.number)); err != nil {
				return err
			}
		}
		if inserted {
			_, err = tx.Exec(r.Context(), `
				INSERT INTO audit_events (organization_id,actor_type,action,entity_type,entity_id,new_state)
				VALUES ($1,'system','charge.generated','charge',$2,$3)
			`, person.OrganizationID, chargeID, map[string]any{"period": period, "amount_minor": rentAmount, "installments": paymentInstallments})
		}
		return err
	})
}

func (app *application) ensureLifecycleNotifications(r *http.Request, person principal, propertyID string) error {
	_, err := app.db.Exec(r.Context(), `
		INSERT INTO notifications (organization_id,property_id,notification_type,title,message,entity_type,entity_id,due_at,dedup_key)
		SELECT l.organization_id,l.property_id,'lease_expiry','Preparar decisión de renovación',
			'El contrato de '||t.name||' vence el '||to_char(l.ends_on,'DD/MM/YYYY')||'. Revisar renovación o término.',
			'lease',l.id,(l.ends_on - l.renewal_review_days)::timestamp,
			'lease-expiry-'||l.id||'-'||to_char(l.ends_on,'YYYYMMDD')
		FROM leases l JOIN tenants t ON t.id=l.tenant_id
		WHERE l.organization_id=$1 AND l.property_id=$2 AND l.status IN ('active','ending') AND l.ends_on IS NOT NULL
		ON CONFLICT DO NOTHING
	`, person.OrganizationID, propertyID)
	if err != nil {
		return err
	}
	_, err = app.db.Exec(r.Context(), `
		INSERT INTO notifications (organization_id,property_id,notification_type,title,message,entity_type,entity_id,due_at,dedup_key)
		SELECT l.organization_id,l.property_id,'vacate_notice','Fecha para solicitar restitución',
			'Si la propiedad no continuará arrendada, el aviso debe enviarse a más tardar el '||
			to_char(l.ends_on - make_interval(months => l.vacate_notice_months, days => l.vacate_notice_days),'DD/MM/YYYY')||
			' ('||l.vacate_notice_months||' meses y '||l.vacate_notice_days||' días de anticipación), usando los medios formales que correspondan.',
			'lease',l.id,(l.ends_on - make_interval(months => l.vacate_notice_months, days => l.vacate_notice_days))::timestamp,
			'vacate-notice-'||l.id||'-'||to_char(l.ends_on,'YYYYMMDD')
		FROM leases l
		WHERE l.organization_id=$1 AND l.property_id=$2 AND l.status IN ('active','ending') AND l.ends_on IS NOT NULL
		ON CONFLICT DO NOTHING
	`, person.OrganizationID, propertyID)
	if err != nil {
		return err
	}
	_, err = app.db.Exec(r.Context(), `
		INSERT INTO notifications (organization_id,property_id,notification_type,title,message,entity_type,entity_id,due_at,dedup_key)
		SELECT organization_id,property_id,'rent_adjustment','Revisar reajuste de arriendo por IPC',
			'Corresponde revisar el IPC el '||to_char(next_adjustment_on,'DD/MM/YYYY')||
			CASE WHEN adjustment_effective_on IS NULL THEN '. ' ELSE '. Si el contrato continúa, la aplicación está prevista desde el '||to_char(adjustment_effective_on,'DD/MM/YYYY')||'. ' END||
			'El monto no cambiará sin aprobación humana.',
			'lease',id,next_adjustment_on::timestamp,'rent-adjustment-'||id||'-'||to_char(next_adjustment_on,'YYYYMMDD')
		FROM leases
		WHERE organization_id=$1 AND property_id=$2 AND status IN ('active','ending') AND adjustment_method='ipc'
		ON CONFLICT DO NOTHING
	`, person.OrganizationID, propertyID)
	if err != nil {
		return err
	}
	_, err = app.db.Exec(r.Context(), `
		INSERT INTO notifications (organization_id,property_id,notification_type,title,message,entity_type,entity_id,due_at,dedup_key)
		SELECT organization_id,property_id,'exit_inspection','Inspección y entrega de la propiedad',
			'Realizar inspección de salida, entrega de llaves, revisión de servicios, fotografías, cálculo de daños y liquidación de garantía.',
			'lease',id,ends_on::timestamp,'exit-inspection-'||id||'-'||to_char(ends_on,'YYYYMMDD')
		FROM leases
		WHERE organization_id=$1 AND property_id=$2 AND status IN ('active','ending') AND ends_on IS NOT NULL
		ON CONFLICT DO NOTHING
	`, person.OrganizationID, propertyID)
	if err != nil {
		return err
	}
	_, err = app.db.Exec(r.Context(), `
		INSERT INTO notifications (organization_id,property_id,notification_type,title,message,entity_type,entity_id,due_at,dedup_key)
		SELECT organization_id,property_id,'maintenance_due','Mantención: '||name,
			'Programada para el '||to_char(next_due_on,'DD/MM/YYYY')||'. '||COALESCE(notes,''),
			'maintenance_schedule',id,(next_due_on-reminder_days)::timestamp,
			'maintenance-'||id||'-'||to_char(next_due_on,'YYYYMMDD')
		FROM maintenance_schedules
		WHERE organization_id=$1 AND property_id=$2 AND status='active'
		ON CONFLICT DO NOTHING
	`, person.OrganizationID, propertyID)
	return err
}

func (app *application) quarterlyReviewCalendar(r *http.Request, person principal, propertyID, period string) ([]reviewCalendarItemResponse, error) {
	quarterStart, err := time.Parse("2006-01", period)
	if err != nil {
		return nil, err
	}
	quarterEnd := quarterStart.AddDate(0, 3, 0)
	items := make([]reviewCalendarItemResponse, 0)
	rows, err := app.db.Query(r.Context(), `
		SELECT id,notification_type,title,message,to_char(due_at,'YYYY-MM-DD'),status
		FROM notifications
		WHERE organization_id=$1 AND property_id=$2 AND due_at >= $3 AND due_at < $4
		  AND status IN ('unread','read')
		ORDER BY due_at,created_at
	`, person.OrganizationID, propertyID, quarterStart, quarterEnd)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var item reviewCalendarItemResponse
		if err := rows.Scan(&item.ID, &item.Type, &item.Title, &item.Message, &item.ScheduledOn, &item.Status); err != nil {
			rows.Close()
			return nil, err
		}
		item.Period = item.ScheduledOn[:7]
		item.Kind = "alert"
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	suggestions := []struct {
		day     int
		typeID  string
		title   string
		message string
	}{
		{7, "financial_review", "Revisión financiera del mes", "Verificar cobros, abonos pendientes de conciliación, saldos y respaldos bancarios."},
		{14, "contract_review", "Revisión contractual y documental", "Revisar vigencia, comunicaciones del arrendatario, reajustes próximos y documentos pendientes."},
		{21, "property_review", "Revisión de propiedad y mantenciones", "Revisar reportes, mantenciones próximas, trabajos pendientes y necesidades del trimestre siguiente."},
	}
	for index, suggestion := range suggestions {
		month := quarterStart.AddDate(0, index, 0)
		scheduled := time.Date(month.Year(), month.Month(), suggestion.day, 0, 0, 0, 0, time.UTC)
		items = append(items, reviewCalendarItemResponse{
			ID:          fmt.Sprintf("suggestion-%s-%s", month.Format("2006-01"), suggestion.typeID),
			Period:      month.Format("2006-01"),
			Kind:        "suggestion",
			Type:        suggestion.typeID,
			Title:       suggestion.title,
			Message:     suggestion.message,
			ScheduledOn: scheduled.Format("2006-01-02"),
			Status:      "suggested",
		})
	}
	return items, nil
}

func (app *application) amendLeaseTerms(w http.ResponseWriter, r *http.Request) {
	person := currentPrincipal(r)
	leaseID := chi.URLParam(r, "leaseID")
	var input struct {
		EffectiveOn         string `json:"effective_on"`
		Reason              string `json:"reason"`
		Version             int64  `json:"version"`
		EndsOn              string `json:"ends_on"`
		RentAmountMinor     int64  `json:"rent_amount_minor"`
		PaymentDay          int16  `json:"payment_day"`
		ScheduleKind        string `json:"schedule_kind"`
		AgreementNotes      string `json:"agreement_notes"`
		RenewalNoticeDays   int16  `json:"renewal_notice_days"`
		RenewalReviewDays   int16  `json:"renewal_review_days"`
		VacateNoticeMonths  int16  `json:"vacate_notice_months"`
		VacateNoticeDays    int16  `json:"vacate_notice_days"`
		AdjustmentMethod    string `json:"adjustment_method"`
		AdjustmentFrequency int16  `json:"adjustment_frequency_months"`
		NextAdjustmentOn    string `json:"next_adjustment_on"`
		AdjustmentEffective string `json:"adjustment_effective_on"`
		Installments        []struct {
			DueDay              int16 `json:"due_day"`
			ExpectedAmountMinor int64 `json:"expected_amount_minor"`
		} `json:"installments"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	effective, effectiveErr := time.Parse("2006-01-02", input.EffectiveOn)
	input.Reason = strings.TrimSpace(input.Reason)
	input.AgreementNotes = strings.TrimSpace(input.AgreementNotes)
	if effectiveErr != nil || len(input.Reason) < 3 || input.Version < 1 || input.RentAmountMinor <= 0 || input.PaymentDay < 1 || input.PaymentDay > 28 || len(input.Installments) < 1 || len(input.Installments) > 12 {
		writeError(w, http.StatusUnprocessableEntity, "invalid_amendment", "Completa la fecha efectiva, el motivo y las condiciones corregidas.")
		return
	}
	if input.EndsOn != "" {
		if _, err := time.Parse("2006-01-02", input.EndsOn); err != nil {
			writeError(w, http.StatusUnprocessableEntity, "invalid_lease_end", "La fecha de término no es válida.")
			return
		}
	}
	var total int64
	seenDays := map[int16]bool{}
	for _, installment := range input.Installments {
		if installment.DueDay < 1 || installment.DueDay > 28 || installment.ExpectedAmountMinor <= 0 || seenDays[installment.DueDay] {
			writeError(w, http.StatusUnprocessableEntity, "invalid_installment_schedule", "Cada fecha del calendario debe ser distinta y usar días entre 1 y 28.")
			return
		}
		seenDays[installment.DueDay] = true
		total += installment.ExpectedAmountMinor
	}
	if total != input.RentAmountMinor {
		writeError(w, http.StatusUnprocessableEntity, "installment_total_mismatch", "El calendario de pagos debe sumar la renta contractual.")
		return
	}
	if input.ScheduleKind == "contractual" {
		if len(input.Installments) != 1 || input.Installments[0].DueDay != input.PaymentDay || input.Installments[0].ExpectedAmountMinor != input.RentAmountMinor {
			writeError(w, http.StatusUnprocessableEntity, "invalid_contractual_schedule", "La obligación contractual debe ser un pago por la renta completa en el día límite del contrato.")
			return
		}
	} else if input.ScheduleKind != "special_agreement" || len(input.AgreementNotes) < 3 {
		writeError(w, http.StatusUnprocessableEntity, "invalid_special_agreement", "Para dividir el pago debes indicar que existe un acuerdo especial y describirlo.")
		return
	}
	if input.RenewalNoticeDays < 0 || input.RenewalNoticeDays > 365 || input.RenewalReviewDays < 0 || input.RenewalReviewDays > 365 || input.VacateNoticeMonths < 0 || input.VacateNoticeMonths > 24 || input.VacateNoticeDays < 0 || input.VacateNoticeDays > 90 {
		writeError(w, http.StatusUnprocessableEntity, "invalid_lease_alerts", "Revisa las anticipaciones configuradas.")
		return
	}
	if input.AdjustmentMethod == "ipc" {
		if input.AdjustmentFrequency < 1 || input.AdjustmentFrequency > 36 {
			writeError(w, http.StatusUnprocessableEntity, "invalid_adjustment", "Indica la periodicidad del reajuste IPC.")
			return
		}
		if _, err := time.Parse("2006-01-02", input.NextAdjustmentOn); err != nil {
			writeError(w, http.StatusUnprocessableEntity, "invalid_adjustment_date", "La fecha de revisión IPC no es válida.")
			return
		}
		if input.AdjustmentEffective != "" {
			if _, err := time.Parse("2006-01-02", input.AdjustmentEffective); err != nil {
				writeError(w, http.StatusUnprocessableEntity, "invalid_adjustment_effective_date", "La fecha de aplicación IPC no es válida.")
				return
			}
		}
	} else if input.AdjustmentMethod == "none" {
		input.AdjustmentFrequency, input.NextAdjustmentOn, input.AdjustmentEffective = 0, "", ""
	} else {
		writeError(w, http.StatusUnprocessableEntity, "invalid_adjustment", "El mecanismo de reajuste no es válido.")
		return
	}

	var propertyID string
	err := pgx.BeginFunc(r.Context(), app.db, func(tx pgx.Tx) error {
		var previous map[string]any
		var startsOn time.Time
		if err := tx.QueryRow(r.Context(), `
			SELECT property_id,starts_on,jsonb_build_object(
				'rent_amount_minor',rent_amount_minor,'payment_day',payment_day,'ends_on',ends_on,
				'renewal_notice_days',renewal_notice_days,'renewal_review_days',renewal_review_days,
				'vacate_notice_months',vacate_notice_months,'vacate_notice_days',vacate_notice_days,
				'adjustment_method',adjustment_method,'adjustment_frequency_months',adjustment_frequency_months,
				'next_adjustment_on',next_adjustment_on,'adjustment_effective_on',adjustment_effective_on,'version',version)
			FROM leases WHERE id=$1 AND organization_id=$2 AND status IN ('active','ending') AND version=$3 FOR UPDATE
		`, leaseID, person.OrganizationID, input.Version).Scan(&propertyID, &startsOn, &previous); err != nil {
			return err
		}
		if effective.Before(startsOn) {
			return fmt.Errorf("effective date before contract")
		}
		effectivePeriod := effective.Format("2006-01")
		var hasPayments bool
		if err := tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM payments p JOIN payment_allocations a ON a.payment_id=p.id JOIN charges c ON c.id=a.charge_id WHERE p.lease_id=$1 AND c.period >= $2 AND p.status NOT IN ('rejected','reversed','duplicate'))`, leaseID, effectivePeriod).Scan(&hasPayments); err != nil {
			return err
		}
		if hasPayments {
			return fmt.Errorf("payments exist after effective date")
		}
		newState := map[string]any{"rent_amount_minor": input.RentAmountMinor, "payment_day": input.PaymentDay, "ends_on": input.EndsOn, "schedule_kind": input.ScheduleKind, "installments": input.Installments, "renewal_notice_days": input.RenewalNoticeDays, "renewal_review_days": input.RenewalReviewDays, "vacate_notice_months": input.VacateNoticeMonths, "vacate_notice_days": input.VacateNoticeDays, "adjustment_method": input.AdjustmentMethod, "next_adjustment_on": input.NextAdjustmentOn, "adjustment_effective_on": input.AdjustmentEffective}
		if _, err := tx.Exec(r.Context(), `
			UPDATE leases SET ends_on=NULLIF($1,'')::date,rent_amount_minor=$2,payment_day=$3,payment_installments=$4,
				renewal_notice_days=$5,renewal_review_days=$6,vacate_notice_months=$7,vacate_notice_days=$8,
				adjustment_method=$9,adjustment_frequency_months=NULLIF($10,0),next_adjustment_on=NULLIF($11,'')::date,
				adjustment_effective_on=NULLIF($12,'')::date,updated_by=$13,updated_at=now(),version=version+1
			WHERE id=$14
		`, input.EndsOn, input.RentAmountMinor, input.PaymentDay, len(input.Installments), input.RenewalNoticeDays, input.RenewalReviewDays, input.VacateNoticeMonths, input.VacateNoticeDays, input.AdjustmentMethod, input.AdjustmentFrequency, input.NextAdjustmentOn, input.AdjustmentEffective, person.UserID, leaseID); err != nil {
			return err
		}
		if _, err := tx.Exec(r.Context(), `DELETE FROM lease_payment_schedules WHERE lease_id=$1`, leaseID); err != nil {
			return err
		}
		for index, installment := range input.Installments {
			if _, err := tx.Exec(r.Context(), `INSERT INTO lease_payment_schedules (organization_id,lease_id,installment_number,due_day,expected_amount_minor,schedule_kind,agreement_notes) VALUES ($1,$2,$3,$4,$5,$6,NULLIF($7,''))`, person.OrganizationID, leaseID, index+1, installment.DueDay, installment.ExpectedAmountMinor, input.ScheduleKind, input.AgreementNotes); err != nil {
				return err
			}
		}
		if _, err := tx.Exec(r.Context(), `
			UPDATE charges SET original_amount_minor=$1,balance_minor=$1,due_date=make_date(split_part(period,'-',1)::int,split_part(period,'-',2)::int,$2),updated_at=now(),version=version+1
			WHERE lease_id=$3 AND charge_type='rent' AND origin='automatic' AND period >= $4
		`, input.RentAmountMinor, input.PaymentDay, leaseID, effectivePeriod); err != nil {
			return err
		}
		if _, err := tx.Exec(r.Context(), `DELETE FROM notifications WHERE organization_id=$1 AND ((entity_type='lease' AND entity_id=$2) OR (entity_type='charge' AND entity_id IN (SELECT id FROM charges WHERE lease_id=$2 AND period >= $3)))`, person.OrganizationID, leaseID, effectivePeriod); err != nil {
			return err
		}
		if _, err := tx.Exec(r.Context(), `INSERT INTO lease_amendments (organization_id,lease_id,effective_on,reason,previous_state,new_state,approved_by) VALUES ($1,$2,$3,$4,$5,$6,$7)`, person.OrganizationID, leaseID, input.EffectiveOn, input.Reason, previous, newState, person.UserID); err != nil {
			return err
		}
		return audit(r.Context(), tx, person, "lease.terms_amended", "lease", leaseID, previous, newState)
	})
	if err != nil {
		writeError(w, http.StatusConflict, "lease_amendment_failed", "No fue posible aplicar la corrección. Revisa la versión y que no existan pagos registrados desde la fecha efectiva.")
		return
	}
	_ = app.ensureLifecycleNotifications(r, person, propertyID)
	writeJSON(w, http.StatusOK, map[string]any{"lease_id": leaseID, "property_id": propertyID, "effective_on": input.EffectiveOn})
}

func (app *application) endLease(w http.ResponseWriter, r *http.Request) {
	person := currentPrincipal(r)
	leaseID := chi.URLParam(r, "leaseID")
	var input struct {
		EndedOn string `json:"ended_on"`
		Reason  string `json:"reason"`
		Status  string `json:"status"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	endedOn, err := time.Parse("2006-01-02", input.EndedOn)
	input.Reason = strings.TrimSpace(input.Reason)
	if err != nil || len(input.Reason) < 3 || (input.Status != "ended" && input.Status != "terminated") {
		writeError(w, http.StatusUnprocessableEntity, "invalid_lease_end", "Indica fecha, motivo y si el contrato terminó normalmente o fue rescindido.")
		return
	}
	var tenantID, propertyID, startsOn string
	err = pgx.BeginFunc(r.Context(), app.db, func(tx pgx.Tx) error {
		if err := tx.QueryRow(r.Context(), `SELECT tenant_id,property_id,to_char(starts_on,'YYYY-MM-DD') FROM leases WHERE id=$1 AND organization_id=$2 AND status IN ('active','ending') FOR UPDATE`, leaseID, person.OrganizationID).Scan(&tenantID, &propertyID, &startsOn); err != nil {
			return err
		}
		start, _ := time.Parse("2006-01-02", startsOn)
		if endedOn.Before(start) {
			return fmt.Errorf("end before start")
		}
		previous := map[string]any{"status": "active", "tenant_id": tenantID}
		tag, err := tx.Exec(r.Context(), `UPDATE leases SET status=$1,ended_on=$2,ends_on=COALESCE(ends_on,$2),end_reason=$3,updated_by=$4,updated_at=now(),version=version+1 WHERE id=$5 AND organization_id=$6 AND status IN ('active','ending')`, input.Status, input.EndedOn, input.Reason, person.UserID, leaseID, person.OrganizationID)
		if err != nil || tag.RowsAffected() != 1 {
			return err
		}
		_, _ = tx.Exec(r.Context(), `UPDATE notifications SET status='dismissed' WHERE organization_id=$1 AND entity_type='lease' AND entity_id=$2 AND status='unread'`, person.OrganizationID, leaseID)
		_, _ = tx.Exec(r.Context(), `UPDATE tenants SET status='inactive',updated_at=now() WHERE id=$1 AND NOT EXISTS (SELECT 1 FROM leases WHERE tenant_id=$1 AND status IN ('active','ending'))`, tenantID)
		return audit(r.Context(), tx, person, "lease.ended", "lease", leaseID, previous, map[string]any{"status": input.Status, "ended_on": input.EndedOn, "reason": input.Reason})
	})
	if err != nil {
		writeError(w, http.StatusConflict, "lease_end_failed", "No fue posible cerrar el contrato vigente.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"lease_id": leaseID, "property_id": propertyID, "status": input.Status})
}

func (app *application) createMaintenanceSchedule(w http.ResponseWriter, r *http.Request) {
	person := currentPrincipal(r)
	propertyID := chi.URLParam(r, "propertyID")
	var input struct {
		Name            string `json:"name"`
		Category        string `json:"category"`
		FrequencyMonths int16  `json:"frequency_months"`
		NextDueOn       string `json:"next_due_on"`
		ReminderDays    int16  `json:"reminder_days"`
		Notes           string `json:"notes"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	allowed := map[string]bool{"inspection": true, "plumbing": true, "electrical": true, "gas": true, "roof": true, "painting": true, "appliance": true, "garden": true, "pest_control": true, "insurance": true, "other": true}
	input.Name = strings.TrimSpace(input.Name)
	if len(input.Name) < 2 || !allowed[input.Category] || input.FrequencyMonths < 1 || input.FrequencyMonths > 120 || input.ReminderDays < 0 || input.ReminderDays > 365 {
		writeError(w, http.StatusUnprocessableEntity, "invalid_maintenance", "Revisa el nombre, categoría, frecuencia y anticipación.")
		return
	}
	if _, err := time.Parse("2006-01-02", input.NextDueOn); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid_maintenance_date", "La próxima fecha de mantención no es válida.")
		return
	}
	var item maintenanceScheduleResponse
	err := app.db.QueryRow(r.Context(), `
		INSERT INTO maintenance_schedules (organization_id,property_id,lease_id,name,category,frequency_months,next_due_on,reminder_days,notes,created_by,updated_by)
		VALUES ($1,$2,(SELECT id FROM leases WHERE organization_id=$1 AND property_id=$2 AND status IN ('active','ending') LIMIT 1),$3,$4,$5,$6,$7,NULLIF($8,''),$9,$9)
		RETURNING id,name,category,frequency_months,to_char(next_due_on,'YYYY-MM-DD'),reminder_days,'',COALESCE(notes,''),status,version
	`, person.OrganizationID, propertyID, input.Name, input.Category, input.FrequencyMonths, input.NextDueOn, input.ReminderDays, strings.TrimSpace(input.Notes), person.UserID).Scan(&item.ID, &item.Name, &item.Category, &item.FrequencyMonths, &item.NextDueOn, &item.ReminderDays, &item.LastCompletedOn, &item.Notes, &item.Status, &item.Version)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "maintenance_create_failed", "No fue posible programar la mantención.")
		return
	}
	_ = app.ensureLifecycleNotifications(r, person, propertyID)
	writeJSON(w, http.StatusCreated, map[string]any{"maintenance": item})
}

func (app *application) completeMaintenanceSchedule(w http.ResponseWriter, r *http.Request) {
	person := currentPrincipal(r)
	id := chi.URLParam(r, "maintenanceID")
	var input struct {
		CompletedOn string `json:"completed_on"`
		Notes       string `json:"notes"`
		Version     int64  `json:"version"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if _, err := time.Parse("2006-01-02", input.CompletedOn); err != nil || input.Version < 1 {
		writeError(w, http.StatusUnprocessableEntity, "invalid_completion", "Indica la fecha de ejecución de la mantención.")
		return
	}
	var propertyID string
	err := pgx.BeginFunc(r.Context(), app.db, func(tx pgx.Tx) error {
		var frequency int
		if err := tx.QueryRow(r.Context(), `SELECT property_id,frequency_months FROM maintenance_schedules WHERE id=$1 AND organization_id=$2 AND status='active' AND version=$3 FOR UPDATE`, id, person.OrganizationID, input.Version).Scan(&propertyID, &frequency); err != nil {
			return err
		}
		if _, err := tx.Exec(r.Context(), `UPDATE maintenance_schedules SET last_completed_on=$1,next_due_on=$1::date+make_interval(months=>$2),notes=CASE WHEN $3='' THEN notes ELSE $3 END,updated_by=$4,updated_at=now(),version=version+1 WHERE id=$5`, input.CompletedOn, frequency, strings.TrimSpace(input.Notes), person.UserID, id); err != nil {
			return err
		}
		_, _ = tx.Exec(r.Context(), `UPDATE notifications SET status='read',read_at=now() WHERE organization_id=$1 AND entity_type='maintenance_schedule' AND entity_id=$2 AND status='unread'`, person.OrganizationID, id)
		return audit(r.Context(), tx, person, "maintenance.completed", "maintenance_schedule", id, nil, map[string]any{"completed_on": input.CompletedOn})
	})
	if err != nil {
		writeError(w, http.StatusConflict, "maintenance_complete_failed", "La mantención cambió o ya no está activa.")
		return
	}
	_ = app.ensureLifecycleNotifications(r, person, propertyID)
	writeJSON(w, http.StatusOK, map[string]any{"maintenance_id": id})
}

func (app *application) getRentalLedger(w http.ResponseWriter, r *http.Request) {
	person := currentPrincipal(r)
	propertyID := chi.URLParam(r, "propertyID")
	period := strings.TrimSpace(r.URL.Query().Get("period"))
	if period == "" {
		period = time.Now().Format("2006-01")
	}
	if _, err := time.Parse("2006-01", period); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid_period", "El período debe usar formato AAAA-MM.")
		return
	}
	_, _ = app.db.Exec(r.Context(), `UPDATE charges SET status='overdue',updated_at=now(),version=version+1 WHERE organization_id=$1 AND property_id=$2 AND due_date<current_date AND balance_minor>0 AND status IN ('pending','partially_paid')`, person.OrganizationID, propertyID)
	_, _ = app.db.Exec(r.Context(), `
		INSERT INTO notifications (organization_id,property_id,notification_type,title,message,entity_type,entity_id,due_at,dedup_key)
		SELECT organization_id,property_id,'charge_overdue','Cobro vencido','Hay un saldo vencido de '||concept,'charge',id,now(),'charge-overdue-'||id
		FROM charges WHERE organization_id=$1 AND property_id=$2 AND status='overdue' ON CONFLICT DO NOTHING
	`, person.OrganizationID, propertyID)
	var lease leaseResponse
	err := app.db.QueryRow(r.Context(), `SELECT l.id,l.tenant_id,t.name,COALESCE(t.email,''),to_char(l.starts_on,'YYYY-MM-DD'),COALESCE(to_char(l.ends_on,'YYYY-MM-DD'),''),l.status,l.rent_amount_minor,l.currency_code,l.payment_day,l.payment_installments,l.renewal_notice_days,l.vacate_notice_months,l.vacate_notice_days,l.adjustment_method,COALESCE(l.adjustment_frequency_months,0),COALESCE(to_char(l.next_adjustment_on,'YYYY-MM-DD'),''),l.renewal_review_days,COALESCE(to_char(l.adjustment_effective_on,'YYYY-MM-DD'),''),l.version FROM leases l JOIN tenants t ON t.id=l.tenant_id WHERE l.organization_id=$1 AND l.property_id=$2 AND l.status IN ('active','ending')`, person.OrganizationID, propertyID).Scan(&lease.ID, &lease.TenantID, &lease.TenantName, &lease.TenantEmail, &lease.StartsOn, &lease.EndsOn, &lease.Status, &lease.RentAmountMinor, &lease.CurrencyCode, &lease.PaymentDay, &lease.PaymentInstallments, &lease.RenewalNoticeDays, &lease.VacateNoticeMonths, &lease.VacateNoticeDays, &lease.AdjustmentMethod, &lease.AdjustmentFrequency, &lease.NextAdjustmentOn, &lease.RenewalReviewDays, &lease.AdjustmentEffective, &lease.Version)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusInternalServerError, "ledger_read_failed", "No fue posible consultar el arriendo.")
		return
	}
	var leaseValue any
	if err == nil {
		lease.Installments = make([]leaseInstallmentResponse, 0, lease.PaymentInstallments)
		scheduleRows, scheduleErr := app.db.Query(r.Context(), `SELECT installment_number,due_day,expected_amount_minor,schedule_kind,COALESCE(agreement_notes,'') FROM lease_payment_schedules WHERE lease_id=$1 ORDER BY installment_number`, lease.ID)
		if scheduleErr != nil {
			writeError(w, http.StatusInternalServerError, "schedule_read_failed", "No fue posible consultar el calendario de cuotas.")
			return
		}
		for scheduleRows.Next() {
			var item leaseInstallmentResponse
			var scheduleKind, agreementNotes string
			if scheduleRows.Scan(&item.InstallmentNumber, &item.DueDay, &item.ExpectedAmountMinor, &scheduleKind, &agreementNotes) == nil {
				lease.Installments = append(lease.Installments, item)
				if lease.ScheduleKind == "" {
					lease.ScheduleKind, lease.AgreementNotes = scheduleKind, agreementNotes
				}
			}
		}
		scheduleRows.Close()
		calendarStart, _ := time.Parse("2006-01", period)
		for monthOffset := 0; monthOffset < 3; monthOffset++ {
			calendarPeriod := calendarStart.AddDate(0, monthOffset, 0).Format("2006-01")
			if err := app.ensureMonthlyRentCharge(r, person, propertyID, calendarPeriod); err != nil {
				app.logger.Error("ensure monthly rent charge", "error", err, "period", calendarPeriod)
				writeError(w, http.StatusInternalServerError, "rent_charge_prepare_failed", "No fue posible preparar el calendario trimestral del arriendo.")
				return
			}
		}
		if err := app.ensureLifecycleNotifications(r, person, propertyID); err != nil {
			app.logger.Error("ensure lifecycle notifications", "error", err)
		}
		_, _ = app.db.Exec(r.Context(), `UPDATE charges SET status='overdue',updated_at=now(),version=version+1 WHERE organization_id=$1 AND property_id=$2 AND due_date<current_date AND balance_minor>0 AND status IN ('pending','partially_paid')`, person.OrganizationID, propertyID)
		leaseValue = lease
	}
	charges := []chargeResponse{}
	rows, err := app.db.Query(r.Context(), `SELECT id,period,charge_type,concept,to_char(due_date,'YYYY-MM-DD'),original_amount_minor,balance_minor,currency_code,status,COALESCE(notes,''),version FROM charges WHERE organization_id=$1 AND property_id=$2 AND period=$3 ORDER BY due_date,charge_type`, person.OrganizationID, propertyID, period)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "ledger_read_failed", "No fue posible consultar los cobros.")
		return
	}
	for rows.Next() {
		var c chargeResponse
		if rows.Scan(&c.ID, &c.Period, &c.ChargeType, &c.Concept, &c.DueDate, &c.OriginalAmountMinor, &c.BalanceMinor, &c.CurrencyCode, &c.Status, &c.Notes, &c.Version) == nil {
			charges = append(charges, c)
		}
	}
	rows.Close()
	payments := []paymentResponse{}
	payRows, err := app.db.Query(r.Context(), `SELECT p.id,a.charge_id,p.payer_name,to_char(p.payment_date,'YYYY-MM-DD'),p.amount_minor,p.currency_code,p.payment_method,COALESCE(p.bank_reference,''),p.status,p.timing_status,COALESCE(p.observations,''),COALESCE(p.supporting_document_id::text,''),to_char(p.created_at,'YYYY-MM-DD HH24:MI'),p.version,COALESCE(p.planned_installment_number,1),COALESCE(to_char(p.planned_due_date,'YYYY-MM-DD'),''),COALESCE(s.expected_amount_minor,p.amount_minor) FROM payments p JOIN payment_allocations a ON a.payment_id=p.id JOIN charges c ON c.id=a.charge_id LEFT JOIN lease_payment_schedules s ON s.lease_id=p.lease_id AND s.installment_number=p.planned_installment_number WHERE p.organization_id=$1 AND p.property_id=$2 AND c.period=$3 ORDER BY p.planned_installment_number,p.payment_date,p.created_at`, person.OrganizationID, propertyID, period)
	if err == nil {
		for payRows.Next() {
			var p paymentResponse
			if payRows.Scan(&p.ID, &p.ChargeID, &p.PayerName, &p.PaymentDate, &p.AmountMinor, &p.CurrencyCode, &p.PaymentMethod, &p.BankReference, &p.Status, &p.TimingStatus, &p.Observations, &p.DocumentID, &p.CreatedAt, &p.Version, &p.InstallmentNumber, &p.PlannedDueDate, &p.ExpectedAmountMinor) == nil {
				payments = append(payments, p)
			}
		}
		payRows.Close()
	}
	notifications := []notificationResponse{}
	nRows, _ := app.db.Query(r.Context(), `SELECT id,notification_type,title,message,COALESCE(to_char(due_at,'YYYY-MM-DD HH24:MI'),''),status FROM notifications WHERE organization_id=$1 AND property_id=$2 AND status='unread' ORDER BY due_at NULLS LAST,created_at DESC`, person.OrganizationID, propertyID)
	if nRows != nil {
		for nRows.Next() {
			var n notificationResponse
			if nRows.Scan(&n.ID, &n.Type, &n.Title, &n.Message, &n.DueAt, &n.Status) == nil {
				notifications = append(notifications, n)
			}
		}
		nRows.Close()
	}
	observations := []observationResponse{}
	oRows, _ := app.db.Query(r.Context(), `SELECT ro.id,ro.category,ro.content,to_char(ro.observed_at,'YYYY-MM-DD HH24:MI'),u.name FROM rental_observations ro JOIN users u ON u.id=ro.created_by WHERE ro.organization_id=$1 AND ro.property_id=$2 ORDER BY ro.observed_at DESC LIMIT 30`, person.OrganizationID, propertyID)
	if oRows != nil {
		for oRows.Next() {
			var o observationResponse
			if oRows.Scan(&o.ID, &o.Category, &o.Content, &o.ObservedAt, &o.CreatedBy) == nil {
				observations = append(observations, o)
			}
		}
		oRows.Close()
	}
	maintenance := []maintenanceScheduleResponse{}
	mRows, _ := app.db.Query(r.Context(), `SELECT id,name,category,frequency_months,to_char(next_due_on,'YYYY-MM-DD'),reminder_days,COALESCE(to_char(last_completed_on,'YYYY-MM-DD'),''),COALESCE(notes,''),status,version FROM maintenance_schedules WHERE organization_id=$1 AND property_id=$2 AND status <> 'archived' ORDER BY next_due_on,name`, person.OrganizationID, propertyID)
	if mRows != nil {
		for mRows.Next() {
			var item maintenanceScheduleResponse
			if mRows.Scan(&item.ID, &item.Name, &item.Category, &item.FrequencyMonths, &item.NextDueOn, &item.ReminderDays, &item.LastCompletedOn, &item.Notes, &item.Status, &item.Version) == nil {
				maintenance = append(maintenance, item)
			}
		}
		mRows.Close()
	}
	var receipt receiptResponse
	var receiptValue any
	if app.db.QueryRow(r.Context(), `SELECT rr.id,rr.document_id,rr.folio,rr.public_token,rr.verification_code,rr.status,to_char(rr.issued_at,'YYYY-MM-DD HH24:MI') FROM rent_receipts rr JOIN charges c ON c.id=rr.charge_id WHERE rr.organization_id=$1 AND rr.property_id=$2 AND c.period=$3 AND rr.status='valid'`, person.OrganizationID, propertyID, period).Scan(&receipt.ID, &receipt.DocumentID, &receipt.Folio, &receipt.PublicToken, &receipt.VerificationCode, &receipt.Status, &receipt.IssuedAt) == nil {
		receiptValue = receipt
	}
	receiptHistory := []receiptHistoryResponse{}
	rRows, _ := app.db.Query(r.Context(), `SELECT rr.id,rr.document_id,rr.folio,rr.public_token,c.period,rr.status,to_char(rr.issued_at,'YYYY-MM-DD HH24:MI') FROM rent_receipts rr JOIN charges c ON c.id=rr.charge_id WHERE rr.organization_id=$1 AND rr.property_id=$2 ORDER BY c.period DESC,rr.issued_at DESC`, person.OrganizationID, propertyID)
	if rRows != nil {
		for rRows.Next() {
			var item receiptHistoryResponse
			if rRows.Scan(&item.ID, &item.DocumentID, &item.Folio, &item.PublicToken, &item.Period, &item.Status, &item.IssuedAt) == nil {
				receiptHistory = append(receiptHistory, item)
			}
		}
		rRows.Close()
	}
	reviewCalendar, calendarErr := app.quarterlyReviewCalendar(r, person, propertyID, period)
	if calendarErr != nil {
		app.logger.Error("quarterly review calendar", "error", calendarErr)
		writeError(w, http.StatusInternalServerError, "review_calendar_failed", "No fue posible preparar el calendario trimestral.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"period": period, "lease": leaseValue, "charges": charges, "payments": payments, "notifications": notifications, "observations": observations, "maintenance": maintenance, "review_calendar": reviewCalendar, "receipt": receiptValue, "receipt_history": receiptHistory})
}

func (app *application) createRentalObservation(w http.ResponseWriter, r *http.Request) {
	person := currentPrincipal(r)
	propertyID := chi.URLParam(r, "propertyID")
	var input struct {
		Category  string `json:"category"`
		Content   string `json:"content"`
		PaymentID string `json:"payment_id"`
		ChargeID  string `json:"charge_id"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, 400, "invalid_request", err.Error())
		return
	}
	allowed := map[string]bool{"payment": true, "tenant_report": true, "property": true, "service": true, "general": true}
	input.Content = strings.TrimSpace(input.Content)
	if !allowed[input.Category] || len(input.Content) < 2 {
		writeError(w, 422, "invalid_observation", "Selecciona una categoría y escribe la observación.")
		return
	}
	var leaseID string
	if err := app.db.QueryRow(r.Context(), `SELECT id FROM leases WHERE organization_id=$1 AND property_id=$2 AND status IN ('active','ending')`, person.OrganizationID, propertyID).Scan(&leaseID); err != nil {
		writeError(w, 409, "active_lease_required", "Primero configura el arriendo.")
		return
	}
	var result observationResponse
	err := app.db.QueryRow(r.Context(), `INSERT INTO rental_observations (organization_id,property_id,lease_id,payment_id,charge_id,category,content,created_by) VALUES ($1,$2,$3,NULLIF($4,'')::uuid,NULLIF($5,'')::uuid,$6,$7,$8) RETURNING id,category,content,to_char(observed_at,'YYYY-MM-DD HH24:MI')`, person.OrganizationID, propertyID, leaseID, input.PaymentID, input.ChargeID, input.Category, input.Content, person.UserID).Scan(&result.ID, &result.Category, &result.Content, &result.ObservedAt)
	if err != nil {
		writeError(w, 422, "observation_create_failed", "No fue posible guardar la observación.")
		return
	}
	result.CreatedBy = person.Name
	if input.Category == "tenant_report" {
		_, _ = app.db.Exec(r.Context(), `INSERT INTO notifications (organization_id,property_id,notification_type,title,message,entity_type,entity_id,due_at,dedup_key) VALUES ($1,$2,'tenant_report','Reporte del arrendatario',$3,'observation',$4,now(),$5)`, person.OrganizationID, propertyID, input.Content, result.ID, "tenant-report-"+result.ID)
	}
	writeJSON(w, 201, map[string]any{"observation": result})
}

func (app *application) readNotification(w http.ResponseWriter, r *http.Request) {
	person := currentPrincipal(r)
	tag, err := app.db.Exec(r.Context(), `UPDATE notifications SET status='read',read_at=now() WHERE id=$1 AND organization_id=$2 AND status='unread'`, chi.URLParam(r, "notificationID"), person.OrganizationID)
	if err != nil || tag.RowsAffected() == 0 {
		writeError(w, 404, "notification_not_found", "La notificación no existe.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (app *application) verifyRentReceipt(w http.ResponseWriter, r *http.Request) {
	var folio, status, period, propertyName, commune, tenantName, amount, currency, issuedAt, code, pdfSHA256, templateVersion string
	err := app.db.QueryRow(r.Context(), `SELECT rr.folio,rr.status,c.period,p.name,p.commune,t.name,(rr.snapshot->>'amount_minor'),(rr.snapshot->>'currency_code'),to_char(rr.issued_at,'YYYY-MM-DD HH24:MI'),rr.verification_code,rr.pdf_sha256,rr.template_version FROM rent_receipts rr JOIN charges c ON c.id=rr.charge_id JOIN properties p ON p.id=rr.property_id JOIN tenants t ON t.id=rr.tenant_id WHERE rr.public_token=$1`, chi.URLParam(r, "token")).Scan(&folio, &status, &period, &propertyName, &commune, &tenantName, &amount, &currency, &issuedAt, &code, &pdfSHA256, &templateVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, 404, "receipt_not_found", "El comprobante no existe.")
		return
	}
	if err != nil {
		writeError(w, 500, "receipt_verify_failed", "No fue posible verificar el comprobante.")
		return
	}
	w.Header().Set("X-Robots-Tag", "noindex, nofollow")
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, 200, map[string]any{"verified": status == "valid", "verification_label": "Verificado por Control de Propiedades", "folio": folio, "status": status, "period": period, "property": propertyName, "commune": commune, "tenant": maskName(tenantName), "amount_minor": amount, "currency_code": currency, "issued_at": issuedAt, "verification_code": code, "pdf_sha256": pdfSHA256, "template_version": templateVersion})
}

func chargeTypeLabel(value string) string {
	labels := map[string]string{"rent": "Arriendo", "electricity": "Luz", "water": "Agua", "trash": "Basura", "common_expense": "Gastos comunes", "adjustment": "Ajuste", "other": "Otro"}
	return labels[value]
}
func paymentMethodLabel(value string) string {
	labels := map[string]string{"bank_transfer": "Transferencia bancaria", "deposit": "Depósito", "cash": "Efectivo", "other": "Otro"}
	return labels[value]
}
func formatCLP(value int64) string {
	text := strconv.FormatInt(value, 10)
	for i := len(text) - 3; i > 0; i -= 3 {
		text = text[:i] + "." + text[i:]
	}
	return "$" + text + " CLP"
}
func maskName(value string) string {
	parts := strings.Fields(value)
	for i := range parts {
		if len([]rune(parts[i])) > 1 {
			r := []rune(parts[i])
			parts[i] = string(r[0]) + strings.Repeat("*", len(r)-1)
		}
	}
	return strings.Join(parts, " ")
}
