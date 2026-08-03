package httpserver

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

func (app *application) createPaymentFromDocument(w http.ResponseWriter, r *http.Request) {
	person := currentPrincipal(r)
	propertyID := chi.URLParam(r, "propertyID")
	var input struct {
		ChargeID      string `json:"charge_id"`
		AmountMinor   int64  `json:"amount_minor"`
		PaymentDate   string `json:"payment_date"`
		PaymentMethod string `json:"payment_method"`
		BankReference string `json:"bank_reference"`
		Observations  string `json:"observations"`
		DocumentID    string `json:"document_id"`
		Confirm       bool   `json:"confirm"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.ChargeID = strings.TrimSpace(input.ChargeID)
	input.DocumentID = strings.TrimSpace(input.DocumentID)
	input.BankReference = strings.TrimSpace(input.BankReference)
	input.Observations = strings.TrimSpace(input.Observations)
	if person.AuthKind == "agent" && !input.Confirm {
		writeError(w, http.StatusUnprocessableEntity, "explicit_confirmation_required", "El registro del comprobante requiere confirm=true.")
		return
	}
	paymentDate, dateErr := time.Parse("2006-01-02", input.PaymentDate)
	allowedMethods := map[string]bool{"bank_transfer": true, "deposit": true, "cash": true, "other": true}
	if input.AmountMinor <= 0 || dateErr != nil || !allowedMethods[input.PaymentMethod] || input.ChargeID == "" || input.DocumentID == "" {
		writeError(w, http.StatusUnprocessableEntity, "invalid_payment", "Revisa cobro, fecha, monto, medio de pago y comprobante.")
		return
	}

	var payment paymentResponse
	err := pgx.BeginFunc(r.Context(), app.db, func(tx pgx.Tx) error {
		var leaseID, tenantID, tenantName, period string
		var balance int64
		if err := tx.QueryRow(r.Context(), `
			SELECT c.lease_id, c.tenant_id, t.name, c.period, c.balance_minor
			FROM charges c JOIN tenants t ON t.id = c.tenant_id
			WHERE c.id = $1 AND c.organization_id = $2 AND c.property_id = $3
			  AND c.status NOT IN ('paid', 'cancelled')
			FOR UPDATE OF c
		`, input.ChargeID, person.OrganizationID, propertyID).Scan(&leaseID, &tenantID, &tenantName, &period, &balance); err != nil {
			return err
		}
		if input.AmountMinor > balance {
			return errAllocationTooLarge
		}
		var documentID string
		if err := tx.QueryRow(r.Context(), `
			UPDATE documents document SET
				display_name = $4::text, status = 'pending_review', document_date = $5::date,
				period = $6::text, amount_minor = $7::bigint, currency_code = 'CLP',
				confidentiality = 'restricted', tags = ARRAY['pago'::text, $6::text],
				updated_by = $8, updated_at = now(), version = document.version + 1
			WHERE document.id = $1 AND document.organization_id = $2 AND document.property_id = $3
			  AND document.document_type = 'bank_receipt' AND document.status = 'uploaded'
			  AND NOT EXISTS (SELECT 1 FROM payments WHERE supporting_document_id = document.id)
			RETURNING document.id
		`, input.DocumentID, person.OrganizationID, propertyID, "Comprobante de pago "+period,
			paymentDate, period, input.AmountMinor, person.UserID).Scan(&documentID); err != nil {
			return err
		}
		var existingPayments int
		if err := tx.QueryRow(r.Context(), `
			SELECT count(*) FROM payments payment
			JOIN payment_allocations allocation ON allocation.payment_id = payment.id
			WHERE allocation.charge_id = $1 AND payment.status NOT IN ('rejected', 'reversed', 'duplicate')
		`, input.ChargeID).Scan(&existingPayments); err != nil {
			return err
		}
		var installmentNumber, dueDay int16
		var expectedAmount int64
		if err := tx.QueryRow(r.Context(), `
			SELECT installment_number, due_day, expected_amount_minor
			FROM lease_payment_schedules
			WHERE lease_id = $1
			  AND installment_number = LEAST($2 + 1, (SELECT payment_installments FROM leases WHERE id = $1))
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
			INSERT INTO payments (
				organization_id, property_id, lease_id, tenant_id, payer_name, payment_date,
				amount_minor, payment_method, bank_reference, supporting_document_id,
				timing_status, observations, registered_by, planned_installment_number, planned_due_date
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NULLIF($9,''),$10,$11,NULLIF($12,''),$13,$14,$15)
			RETURNING id, to_char(payment_date,'YYYY-MM-DD'), currency_code, status,
			          to_char(created_at,'YYYY-MM-DD HH24:MI'), version
		`, person.OrganizationID, propertyID, leaseID, tenantID, payer, paymentDate,
			input.AmountMinor, input.PaymentMethod, input.BankReference, documentID, timing,
			input.Observations, person.UserID, installmentNumber, plannedDueDate).Scan(
			&payment.ID, &payment.PaymentDate, &payment.CurrencyCode, &payment.Status,
			&payment.CreatedAt, &payment.Version,
		); err != nil {
			return err
		}
		payment.ChargeID, payment.PayerName = input.ChargeID, payer
		payment.AmountMinor, payment.PaymentMethod = input.AmountMinor, input.PaymentMethod
		payment.BankReference, payment.TimingStatus = input.BankReference, timing
		payment.Observations, payment.DocumentID = input.Observations, documentID
		payment.InstallmentNumber = installmentNumber
		payment.PlannedDueDate, payment.ExpectedAmountMinor = plannedDueDate.Format("2006-01-02"), expectedAmount
		if _, err := tx.Exec(r.Context(), `
			INSERT INTO payment_allocations (organization_id, payment_id, charge_id, amount_minor)
			VALUES ($1, $2, $3, $4)
		`, person.OrganizationID, payment.ID, input.ChargeID, input.AmountMinor); err != nil {
			return err
		}
		if _, err := tx.Exec(r.Context(), `
			INSERT INTO notifications (
				organization_id, property_id, notification_type, title, message,
				entity_type, entity_id, due_at, dedup_key
			) VALUES ($1,$2,'payment_review','Pago por revisar',$3,'payment',$4,now(),$5)
		`, person.OrganizationID, propertyID,
			fmt.Sprintf("%s registró un abono de %s para %s.", payer, formatCLP(input.AmountMinor), period),
			payment.ID, "payment-review-"+payment.ID); err != nil {
			return err
		}
		return audit(r.Context(), tx, person, "payment.registered", "payment", payment.ID, nil, payment)
	})
	if errors.Is(err, errAllocationTooLarge) {
		writeError(w, http.StatusConflict, "payment_exceeds_balance", "El abono supera el saldo pendiente del cobro.")
		return
	}
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "payment_source_not_found", "El cobro o comprobante no existe, ya fue utilizado o dejó de estar disponible.")
		return
	}
	if err != nil {
		app.logger.Error("create payment from direct document", "error", err)
		writeError(w, http.StatusInternalServerError, "payment_create_failed", "No fue posible registrar el pago.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"payment": payment})
}
