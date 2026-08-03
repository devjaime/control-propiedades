"use client";

import { FormEvent, useCallback, useEffect, useMemo, useState } from "react";
import { APIError, api, formatBytes, uploadDocumentDirect } from "@/lib/api";
import type { RentalLedger as RentalLedgerData } from "@/lib/types";

const today = new Date().toISOString().slice(0, 10);
const currentPeriod = today.slice(0, 7);
const money = new Intl.NumberFormat("es-CL", { style: "currency", currency: "CLP", maximumFractionDigits: 0 });

function dueDate(period: string, day: number) {
  return `${period}-${String(Math.min(28, Math.max(1, day))).padStart(2, "0")}`;
}

function installmentAmount(total: number, count: number, index: number) {
  const base = Math.floor(total / count);
  return index === count - 1 ? total - base * index : base;
}

function addMonths(period: string, offset: number) {
  const [year, month] = period.split("-").map(Number);
  const date = new Date(Date.UTC(year, month - 1 + offset, 1));
  return `${date.getUTCFullYear()}-${String(date.getUTCMonth() + 1).padStart(2, "0")}`;
}

function monthLabel(period: string) {
  const [year, month] = period.split("-").map(Number);
  return new Intl.DateTimeFormat("es-CL", { month: "long", year: "numeric", timeZone: "UTC" }).format(new Date(Date.UTC(year, month - 1, 1)));
}

export function RentalLedger({ propertyID }: { propertyID: string }) {
  const [period, setPeriod] = useState(currentPeriod);
  const [ledger, setLedger] = useState<RentalLedgerData | null>(null);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [working, setWorking] = useState(false);
  const [installmentCount, setInstallmentCount] = useState(1);
  const [rentAmount, setRentAmount] = useState(0);
  const [adjustmentMethod, setAdjustmentMethod] = useState<"none" | "ipc">("none");
  const [specialAgreement, setSpecialAgreement] = useState(false);
  const [amending, setAmending] = useState(false);
  const [amendRentAmount, setAmendRentAmount] = useState(0);
  const [amendInstallmentCount, setAmendInstallmentCount] = useState(1);
  const [amendScheduleKind, setAmendScheduleKind] = useState<"contractual" | "special_agreement">("contractual");
  const [amendAdjustmentMethod, setAmendAdjustmentMethod] = useState<"none" | "ipc">("none");

  const load = useCallback(async () => {
    try {
      const result = await api<RentalLedgerData>(`/properties/${propertyID}/rental-ledger?period=${period}`);
      setLedger(result);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "No fue posible cargar el arriendo.");
    }
  }, [period, propertyID]);

  useEffect(() => {
    let active = true;
    api<RentalLedgerData>(`/properties/${propertyID}/rental-ledger?period=${period}`)
      .then((result) => { if (active) setLedger(result); })
      .catch((caught) => { if (active) setError(caught instanceof Error ? caught.message : "No fue posible cargar el arriendo."); });
    return () => { active = false; };
  }, [period, propertyID]);

  const total = useMemo(() => ledger?.charges.reduce((sum, charge) => sum + charge.original_amount_minor, 0) ?? 0, [ledger]);
  const balance = useMemo(() => ledger?.charges.reduce((sum, charge) => sum + charge.balance_minor, 0) ?? 0, [ledger]);
  const rentCharge = ledger?.charges.find((charge) => charge.charge_type === "rent");
  const rentPayments = ledger?.payments.filter((payment) => payment.charge_id === rentCharge?.id && !["rejected", "reversed", "duplicate"].includes(payment.status)) ?? [];
  const nextInstallment = ledger?.lease?.installments[Math.min(rentPayments.length, Math.max(0, (ledger.lease?.installments.length ?? 1) - 1))];
  const suggestedAmount = Math.min(nextInstallment?.expected_amount_minor ?? 0, rentCharge?.balance_minor ?? 0);
  const calendarItems = ledger?.review_calendar;
  const reviewMonths = useMemo(() => Array.from({ length: 3 }, (_, index) => {
    const monthPeriod = addMonths(period, index);
    return {
      period: monthPeriod,
      label: monthLabel(monthPeriod),
      items: (calendarItems ?? []).filter((item) => item.period === monthPeriod).toSorted((left, right) => left.scheduled_on.localeCompare(right.scheduled_on)),
    };
  }), [calendarItems, period]);

  async function submitJSON(path: string, body: unknown, message: string) {
    setWorking(true); setError(""); setSuccess("");
    try {
      await api(path, { method: "POST", body: JSON.stringify(body) });
      setSuccess(message); await load();
    } catch (caught) {
      setError(caught instanceof APIError ? caught.message : "No fue posible guardar la información.");
    } finally { setWorking(false); }
  }

  async function createLease(event: FormEvent<HTMLFormElement>) {
    event.preventDefault(); const form = event.currentTarget; const data = new FormData(form);
    const amount = Number(data.get("rent_amount_minor"));
    const paymentDay = Number(data.get("payment_day"));
    const baseInstallment = Math.floor(amount / installmentCount);
    const installments = specialAgreement ? Array.from({ length: installmentCount }, (_, index) => ({ due_day: Number(data.get(`installment_day_${index}`)), expected_amount_minor: index === installmentCount - 1 ? amount - baseInstallment * index : baseInstallment })) : [{ due_day: paymentDay, expected_amount_minor: amount }];
    await submitJSON(`/properties/${propertyID}/lease`, {
      tenant_name: data.get("tenant_name"), tenant_identifier: data.get("tenant_identifier"),
      tenant_email: data.get("tenant_email"), tenant_phone: data.get("tenant_phone"),
      starts_on: data.get("starts_on"), ends_on: data.get("ends_on"),
      rent_amount_minor: amount, payment_day: paymentDay,
      schedule_kind: specialAgreement ? "special_agreement" : "contractual",
      agreement_notes: specialAgreement ? data.get("agreement_notes") : "",
      installments, special_terms: data.get("special_terms"),
      renewal_notice_days: Number(data.get("renewal_notice_days")),
      renewal_review_days: Number(data.get("renewal_review_days")),
      vacate_notice_months: Number(data.get("vacate_notice_months")),
      vacate_notice_days: Number(data.get("vacate_notice_days")),
      adjustment_method: data.get("adjustment_method"),
      adjustment_frequency_months: adjustmentMethod === "ipc" ? Number(data.get("adjustment_frequency_months")) : 0,
      next_adjustment_on: adjustmentMethod === "ipc" ? data.get("next_adjustment_on") : "",
      adjustment_effective_on: adjustmentMethod === "ipc" ? data.get("adjustment_effective_on") : "",
    }, "Arriendo configurado. El cobro mensual y sus cuotas se prepararán automáticamente.");
  }

  async function createCharge(event: FormEvent<HTMLFormElement>) {
    event.preventDefault(); const form = event.currentTarget; const data = new FormData(form);
    await submitJSON(`/properties/${propertyID}/charges`, {
      period, charge_type: data.get("charge_type"), concept: data.get("concept"),
      due_date: data.get("due_date"), amount_minor: Number(data.get("amount_minor")), notes: data.get("notes"),
    }, "Cobro agregado al período.");
    form.reset();
  }

  async function createPayment(event: FormEvent<HTMLFormElement>) {
    event.preventDefault(); setWorking(true); setError(""); setSuccess("");
    const form = event.currentTarget; const data = new FormData(form); const file = data.get("file");
    if (!(file instanceof File) || file.size === 0) { setError("Adjunta el comprobante del depósito."); setWorking(false); return; }
    if (file.size > 50 * 1024 * 1024) { setError(`El archivo pesa ${formatBytes(file.size)}. El máximo es 50 MB.`); setWorking(false); return; }
    try {
      const uploaded = await uploadDocumentDirect<{ document: { id: string } }>(propertyID, file, {
        displayName: "Comprobante de pago",
        documentType: "bank_receipt",
        tags: ["pago", period],
      });
      await api(`/properties/${propertyID}/payments/from-document`, {
        method: "POST",
        body: JSON.stringify({
          charge_id: data.get("charge_id"),
          amount_minor: Number(data.get("amount_minor")),
          payment_date: data.get("payment_date"),
          payment_method: data.get("payment_method"),
          bank_reference: String(data.get("bank_reference") ?? "").trim(),
          observations: String(data.get("observations") ?? "").trim(),
          document_id: uploaded.document.id,
        }),
      });
      form.reset(); setSuccess("Abono registrado y pendiente de revisión. Aún no se considera pagado."); await load();
    } catch (caught) {
      setError(caught instanceof APIError ? caught.message : "No fue posible registrar el abono.");
    } finally { setWorking(false); }
  }

  async function confirmPayment(paymentID: string, version: number) {
    await submitJSON(`/payments/${paymentID}/confirm`, { version }, "Pago confirmado y saldo actualizado.");
  }

  async function createObservation(event: FormEvent<HTMLFormElement>) {
    event.preventDefault(); const form = event.currentTarget; const data = new FormData(form);
    await submitJSON(`/properties/${propertyID}/observations`, { category: data.get("category"), content: data.get("content"), payment_id: "", charge_id: "" }, "Observación guardada para seguimiento y análisis.");
    form.reset();
  }

  function openAmendment() {
    if (!ledger?.lease) return;
    setAmendRentAmount(ledger.lease.rent_amount_minor);
    setAmendInstallmentCount(ledger.lease.payment_installments);
    setAmendScheduleKind(ledger.lease.schedule_kind);
    setAmendAdjustmentMethod(ledger.lease.adjustment_method);
    setAmending(true);
  }

  async function amendLease(event: FormEvent<HTMLFormElement>) {
    event.preventDefault(); if (!ledger?.lease) return;
    const data = new FormData(event.currentTarget);
    const paymentDay = Number(data.get("payment_day"));
    const baseInstallment = Math.floor(amendRentAmount / amendInstallmentCount);
    const installments = amendScheduleKind === "special_agreement" ? Array.from({ length: amendInstallmentCount }, (_, index) => ({ due_day: Number(data.get(`amend_installment_day_${index}`)), expected_amount_minor: index === amendInstallmentCount - 1 ? amendRentAmount - baseInstallment * index : baseInstallment })) : [{ due_day: paymentDay, expected_amount_minor: amendRentAmount }];
    await submitJSON(`/leases/${ledger.lease.id}/terms`, {
      effective_on: data.get("effective_on"), reason: data.get("reason"), version: ledger.lease.version,
      ends_on: data.get("ends_on"), rent_amount_minor: amendRentAmount, payment_day: paymentDay,
      schedule_kind: amendScheduleKind, agreement_notes: amendScheduleKind === "special_agreement" ? data.get("agreement_notes") : "",
      installments, renewal_notice_days: Number(data.get("renewal_notice_days")), renewal_review_days: Number(data.get("renewal_review_days")),
      vacate_notice_months: Number(data.get("vacate_notice_months")), vacate_notice_days: Number(data.get("vacate_notice_days")),
      adjustment_method: amendAdjustmentMethod, adjustment_frequency_months: amendAdjustmentMethod === "ipc" ? Number(data.get("adjustment_frequency_months")) : 0,
      next_adjustment_on: amendAdjustmentMethod === "ipc" ? data.get("next_adjustment_on") : "", adjustment_effective_on: amendAdjustmentMethod === "ipc" ? data.get("adjustment_effective_on") : "",
    }, "Condiciones corregidas mediante una enmienda auditada. El arrendatario no fue modificado.");
    setAmending(false);
  }

  async function endLease(event: FormEvent<HTMLFormElement>) {
    event.preventDefault(); const form = event.currentTarget; const data = new FormData(form);
    await submitJSON(`/leases/${ledger?.lease?.id}/end`, {
      ended_on: data.get("ended_on"), reason: data.get("reason"), status: data.get("status"),
    }, "Contrato cerrado. Ya puedes registrar un contrato nuevo y, si corresponde, otro arrendatario.");
  }

  async function createMaintenance(event: FormEvent<HTMLFormElement>) {
    event.preventDefault(); const form = event.currentTarget; const data = new FormData(form);
    await submitJSON(`/properties/${propertyID}/maintenance-schedules`, {
      name: data.get("name"), category: data.get("category"),
      frequency_months: Number(data.get("frequency_months")), next_due_on: data.get("next_due_on"),
      reminder_days: Number(data.get("reminder_days")), notes: data.get("notes"),
    }, "Mantención programada y alerta creada.");
    form.reset();
  }

  async function completeMaintenance(id: string, version: number) {
    await submitJSON(`/maintenance-schedules/${id}/complete`, {
      completed_on: today, notes: "", version,
    }, "Mantención registrada. La siguiente fecha fue calculada automáticamente.");
  }

  async function readNotification(id: string) {
    try { await api(`/notifications/${id}/read`, { method: "PATCH", body: JSON.stringify({}) }); await load(); }
    catch (caught) { setError(caught instanceof Error ? caught.message : "No fue posible actualizar la alerta."); }
  }

  if (!ledger) return <section className="rental-shell" id="arriendo"><p>Cargando arriendo y pagos…</p></section>;

  return <section className="rental-shell" id="arriendo">
    <div className="section-heading"><div><p className="eyebrow">Arriendo y pagos</p><h2>Control mensual</h2></div><label className="period-picker">Período<input type="month" value={period} onChange={(event) => setPeriod(event.target.value)} /></label></div>
    {error ? <p className="form-error" role="alert">{error}</p> : null}{success ? <p className="form-success">{success}</p> : null}

    {!ledger.lease ? <div className="rental-card"><h3>Configurar arrendatario y contrato</h3><p>Estos datos se usarán en cobros, análisis y comprobantes verificables.</p><form className="rental-form" onSubmit={createLease}>
      <label>Nombre del arrendatario<input name="tenant_name" required /></label><label>RUT o identificador<input name="tenant_identifier" /></label>
      <label>Correo<input name="tenant_email" type="email" /></label><label>Teléfono<input name="tenant_phone" /></label>
      <label>Inicio<input name="starts_on" type="date" defaultValue={today} required /></label><label>Término opcional<input name="ends_on" type="date" /></label>
      <label>Renta contractual mensual en CLP<input name="rent_amount_minor" type="number" min="1" value={rentAmount || ""} onChange={(event) => setRentAmount(Number(event.target.value))} required /></label><label>Día límite según contrato<input name="payment_day" type="number" min="1" max="28" defaultValue="3" required /></label>
      <label className="full-field checkbox-field"><input type="checkbox" checked={specialAgreement} onChange={(event) => { setSpecialAgreement(event.target.checked); if (!event.target.checked) setInstallmentCount(1); }} /> Existe un acuerdo especial para dividir el pago</label>
      {specialAgreement ? <div className="installment-builder full-field"><div><strong>Acuerdo especial de pago</strong><span>No reemplaza la obligación contractual original.</span></div><label>Número de cuotas<input type="number" min="2" max="12" value={installmentCount} onChange={(event) => setInstallmentCount(Math.min(12, Math.max(2, Number(event.target.value))))} required /></label>{Array.from({ length: installmentCount }, (_, index) => <label key={index}>Cuota {index + 1} · {money.format(installmentAmount(rentAmount, installmentCount, index))}<input name={`installment_day_${index}`} type="number" min="1" max="28" defaultValue={index === 0 ? 5 : index === 1 ? 15 : undefined} placeholder="Día del mes" required /></label>)}<label className="full-field">Descripción del acuerdo<input name="agreement_notes" placeholder="Ej. Acuerdo escrito para pagar en dos abonos" required /></label></div> : null}
      <div className="installment-builder full-field"><div><strong>Alertas del contrato</strong><span>Fechas estructuradas para seguimiento y agentes.</span></div><input name="renewal_notice_days" type="hidden" value="30" /><label>Preparar decisión con días de anticipación<input name="renewal_review_days" type="number" min="0" max="365" defaultValue="45" required /></label><label>Anticipación para enviar aviso (meses)<input name="vacate_notice_months" type="number" min="0" max="24" defaultValue="1" required /></label><label>Días adicionales de anticipación<input name="vacate_notice_days" type="number" min="0" max="90" defaultValue="0" required /></label></div>
      <label>Mecanismo de reajuste<select name="adjustment_method" value={adjustmentMethod} onChange={(event) => setAdjustmentMethod(event.target.value as "none" | "ipc")}><option value="none">Sin reajuste programado</option><option value="ipc">IPC</option></select></label>{adjustmentMethod === "ipc" ? <><label>Reajustar cada cuántos meses<input name="adjustment_frequency_months" type="number" min="1" max="36" defaultValue="12" required /></label><label>Fecha de revisión/publicación IPC<input name="next_adjustment_on" type="date" required /></label><label>Aplicar nuevo monto desde<input name="adjustment_effective_on" type="date" /></label></> : null}
      <label className="full-field">Condiciones u observaciones<input name="special_terms" /></label><button className="primary-button full-field" disabled={working}>Guardar arriendo</button>
    </form></div> : <>
      <div className="lease-summary"><div><span>Arrendatario del contrato</span><strong>{ledger.lease.tenant_name}</strong><small>Fijo durante esta vigencia</small></div><div><span>Obligación contractual</span><strong>{money.format(ledger.lease.rent_amount_minor)}</strong><small>Pago completo hasta el día {ledger.lease.payment_day}</small></div><div><span>{ledger.lease.schedule_kind === "special_agreement" ? "Acuerdo especial" : "Forma contractual"}</span><strong>{ledger.lease.payment_installments} {ledger.lease.payment_installments === 1 ? "pago" : "cuotas"}</strong><small>{ledger.lease.installments.map((item) => `día ${item.due_day}`).join(" · ")}</small></div><div><span>Contrato</span><strong>{ledger.lease.status}</strong><small>{ledger.lease.ends_on ? `Vence ${ledger.lease.ends_on}` : "Sin término informado"}</small></div></div>
      <div className="rental-card amendment-card"><div className="amendment-heading"><div><h3>Condiciones del contrato</h3><p>Si detectas un error, registra una corrección auditada. El arrendatario y los movimientos históricos no se pueden sobrescribir.</p></div>{!amending ? <button className="secondary-button" type="button" onClick={openAmendment}>Corregir condiciones</button> : <button className="text-button" type="button" onClick={() => setAmending(false)}>Cancelar</button>}</div>{amending ? <form className="rental-form" onSubmit={amendLease}>
        <label>Fecha efectiva de la corrección<input name="effective_on" type="date" defaultValue={ledger.lease.starts_on} required /></label><label>Motivo de la corrección<input name="reason" placeholder="Ej. Calendario ingresado distinto al contrato firmado" required /></label>
        <label>Fecha de término<input name="ends_on" type="date" defaultValue={ledger.lease.ends_on} /></label><label>Renta contractual CLP<input name="rent_amount_minor" type="number" min="1" value={amendRentAmount || ""} onChange={(event) => setAmendRentAmount(Number(event.target.value))} required /></label>
        <label>Día límite contractual<input name="payment_day" type="number" min="1" max="28" defaultValue={ledger.lease.payment_day} required /></label><label>Forma de pago<select value={amendScheduleKind} onChange={(event) => { const value = event.target.value as "contractual" | "special_agreement"; setAmendScheduleKind(value); setAmendInstallmentCount(value === "contractual" ? 1 : Math.max(2, ledger.lease?.payment_installments ?? 2)); }}><option value="contractual">Pago único según contrato</option><option value="special_agreement">Acuerdo especial en cuotas</option></select></label>
        {amendScheduleKind === "special_agreement" ? <div className="installment-builder full-field"><div><strong>Acuerdo especial</strong><span>La renta y el vencimiento contractual permanecen visibles.</span></div><label>Número de cuotas<input type="number" min="2" max="12" value={amendInstallmentCount} onChange={(event) => setAmendInstallmentCount(Math.min(12, Math.max(2, Number(event.target.value))))} /></label>{Array.from({ length: amendInstallmentCount }, (_, index) => <label key={index}>Cuota {index + 1} · {money.format(installmentAmount(amendRentAmount, amendInstallmentCount, index))}<input name={`amend_installment_day_${index}`} type="number" min="1" max="28" defaultValue={ledger.lease?.installments[index]?.due_day ?? (index === 0 ? 5 : index === 1 ? 15 : undefined)} required /></label>)}<label className="full-field">Descripción del acuerdo<input name="agreement_notes" defaultValue={ledger.lease.agreement_notes} required /></label></div> : null}
        <input name="renewal_notice_days" type="hidden" value={ledger.lease.renewal_notice_days} /><label>Preparar renovación con días de anticipación<input name="renewal_review_days" type="number" min="0" max="365" defaultValue={ledger.lease.renewal_review_days} required /></label><label>Aviso de restitución: meses antes<input name="vacate_notice_months" type="number" min="0" max="24" defaultValue={ledger.lease.vacate_notice_months} required /></label><label>Días adicionales<input name="vacate_notice_days" type="number" min="0" max="90" defaultValue={ledger.lease.vacate_notice_days} required /></label>
        <label>Reajuste<select value={amendAdjustmentMethod} onChange={(event) => setAmendAdjustmentMethod(event.target.value as "none" | "ipc")}><option value="none">Sin reajuste programado</option><option value="ipc">IPC</option></select></label>{amendAdjustmentMethod === "ipc" ? <><label>Frecuencia en meses<input name="adjustment_frequency_months" type="number" min="1" max="36" defaultValue={ledger.lease.adjustment_frequency_months || 12} required /></label><label>Revisar/publicación IPC<input name="next_adjustment_on" type="date" defaultValue={ledger.lease.next_adjustment_on} required /></label><label>Aplicar desde<input name="adjustment_effective_on" type="date" defaultValue={ledger.lease.adjustment_effective_on} /></label></> : null}
        <button className="primary-button full-field" disabled={working}>Guardar corrección auditada</button>
      </form> : null}</div>
      {ledger.notifications.length ? <div className="notification-stack"><h3>Requiere tu atención</h3>{ledger.notifications.map((notification) => <article key={notification.id}><div><strong>{notification.title}</strong><p>{notification.message}</p><small>{notification.due_at}</small></div><button className="text-button" onClick={() => readNotification(notification.id)}>Marcar revisada</button></article>)}</div> : null}
      <div className="quarter-calendar"><div className="calendar-heading"><div><p className="eyebrow">Próximos tres meses</p><h3>Calendario de alertas y revisiones</h3></div><p>Se actualiza desde el mes seleccionado y conserva las alertas revisadas.</p></div><div className="calendar-months">{reviewMonths.map((month) => <section key={month.period}><header><span>{month.period}</span><strong>{month.label}</strong></header><div className="calendar-items">{month.items.length ? month.items.map((item) => <article className={`calendar-item calendar-${item.kind} calendar-${item.status}`} key={item.id}><div><span>{item.scheduled_on.slice(8, 10)}</span><small>{item.kind === "alert" ? "Alerta" : "Revisión"}</small></div><div><strong>{item.title}</strong><p>{item.message}</p>{item.status === "read" ? <small>Revisada</small> : null}</div>{item.kind === "alert" && item.status === "unread" ? <button className="text-button" type="button" onClick={() => readNotification(item.id)}>Revisar</button> : null}</article>) : <p className="empty-copy">Sin actividades programadas.</p>}</div></section>)}</div></div>
      <div className="ledger-metrics"><article><span>Cobrado</span><strong>{money.format(total)}</strong></article><article><span>Confirmado</span><strong>{money.format(total - balance)}</strong></article><article><span>Saldo del mes</span><strong>{money.format(balance)}</strong></article></div>

      <div className="rental-grid"><div className="rental-card"><h3>Agregar servicios opcionales</h3><p>El arriendo del mes se crea automáticamente desde el contrato. Aquí puedes sumar luz, agua u otros cobros.</p><form className="compact-form" onSubmit={createCharge}>
        <label>Tipo<select name="charge_type" required defaultValue="electricity"><option value="electricity">Luz</option><option value="water">Agua</option><option value="trash">Basura</option><option value="common_expense">Gastos comunes</option><option value="other">Otro</option></select></label>
        <label>Concepto<input name="concept" placeholder="Ej. Luz julio" /></label><label>Monto CLP<input name="amount_minor" type="number" min="1" required /></label>
        <label>Vencimiento<input name="due_date" type="date" defaultValue={dueDate(period, 20)} required /></label><label className="full-field">Notas<input name="notes" /></label><button className="secondary-button full-field" disabled={working}>Agregar servicio</button>
      </form></div>
      <div className="rental-card"><h3>Subir comprobante del arrendatario</h3><p>Los datos de {ledger.lease.tenant_name} ya están guardados para todo el contrato. Solo registra cada depósito hasta completar el mes.</p>{nextInstallment && rentCharge?.balance_minor ? <div className="next-installment"><span>Próxima cuota planificada</span><strong>Cuota {nextInstallment.installment_number} de {ledger.lease.payment_installments} · día {nextInstallment.due_day} · {money.format(suggestedAmount)}</strong></div> : null}<form className="compact-form" onSubmit={createPayment}>
        <label className="full-field">Aplicar a<select key={`${period}-${rentPayments.length}-${rentCharge?.id ?? ""}`} name="charge_id" defaultValue={rentCharge?.id ?? ""} required><option value="">Selecciona un cobro</option>{ledger.charges.filter((charge) => charge.balance_minor > 0).map((charge) => <option key={charge.id} value={charge.id}>{charge.concept} · saldo {money.format(charge.balance_minor)}</option>)}</select></label>
        <label>Monto CLP<input key={`${period}-${rentPayments.length}-${suggestedAmount}`} name="amount_minor" type="number" min="1" defaultValue={suggestedAmount || undefined} required /></label><label>Fecha recibida<input name="payment_date" type="date" defaultValue={today} required /></label>
        <label>Medio<select name="payment_method" defaultValue="bank_transfer"><option value="bank_transfer">Transferencia</option><option value="deposit">Depósito</option><option value="cash">Efectivo</option><option value="other">Otro</option></select></label><label>Referencia bancaria<input name="bank_reference" /></label>
        <label className="full-field">Comprobante de depósito<input name="file" type="file" accept="application/pdf,image/jpeg,image/png,image/webp" required /></label>
        <label className="full-field">Observaciones<input name="observations" placeholder="Pago atrasado, diferencia informada, etc." /></label><button className="primary-button full-field" disabled={working}>Registrar abono para revisión</button>
      </form></div></div>

      <div className="rental-card ledger-list"><h3>Estado del período</h3>{ledger.charges.length ? ledger.charges.map((charge) => <article key={charge.id}><div><strong>{charge.concept}</strong><span>Vence {charge.due_date} · {charge.status}</span></div><div className="amount-copy"><strong>{money.format(charge.original_amount_minor - charge.balance_minor)} / {money.format(charge.original_amount_minor)}</strong><span>Saldo {money.format(charge.balance_minor)}</span></div></article>) : <p className="empty-copy">Agrega el cobro de arriendo y los servicios que correspondan.</p>}</div>
      <div className="rental-card ledger-list"><h3>Cuotas y abonos recibidos</h3>{ledger.payments.length ? ledger.payments.map((payment) => <article key={payment.id}><div><strong>Cuota {payment.installment_number} · {money.format(payment.amount_minor)} · {payment.payment_date}</strong><span>{payment.payer_name} · programada {payment.planned_due_date} · {payment.timing_status === "late" ? "pago atrasado" : "en fecha"}</span>{payment.observations ? <small>{payment.observations}</small> : null}</div><div className="payment-actions"><span className={`payment-state state-${payment.status}`}>{payment.status}</span>{payment.document_id ? <a className="text-link" href={`/api/v1/documents/${payment.document_id}/download`}>Ver respaldo</a> : null}{payment.status === "pending_reconciliation" ? <button type="button" className="primary-button" disabled={working} onClick={() => confirmPayment(payment.id, payment.version)}>Confirmar</button> : null}</div></article>) : <p className="empty-copy">Todavía no hay comprobantes para este período.</p>}</div>
      {ledger.receipt ? <div className="receipt-banner"><div><p className="eyebrow">Arriendo completado</p><h3>{ledger.receipt.folio}</h3><p>Comprobante firmado y verificado por Control de Propiedades.</p></div><div><a className="primary-button" href={`/api/v1/documents/${ledger.receipt.document_id}/download`}>Descargar PDF</a><a className="secondary-button" href={`/verificar/comprobante/${ledger.receipt.public_token}`} target="_blank">Verificar QR</a></div></div> : null}
      {ledger.receipt_history.length ? <div className="rental-card receipt-history"><h3>Historial de comprobantes de arriendo</h3><p>Vales emitidos después de conciliar el pago completo de cada mes.</p>{ledger.receipt_history.map((item) => <article key={item.id}><div><strong>{item.period}</strong><span>{item.folio} · emitido {item.issued_at}</span></div><div><a className="text-link" href={`/api/v1/documents/${item.document_id}/download`}>Descargar PDF</a><a className="text-link" href={`/verificar/comprobante/${item.public_token}`} target="_blank">Verificar QR</a></div></article>)}</div> : null}
      <div className="rental-grid contract-management"><div className="rental-card"><h3>Mantenciones y renovaciones</h3><p>Programa inspecciones, mantenciones preventivas, seguros u otras renovaciones.</p><form className="compact-form" onSubmit={createMaintenance}><label>Nombre<input name="name" placeholder="Ej. Revisión calefón" required /></label><label>Categoría<select name="category" defaultValue="inspection"><option value="inspection">Inspección</option><option value="plumbing">Gasfitería</option><option value="electrical">Electricidad</option><option value="gas">Gas</option><option value="roof">Techo</option><option value="painting">Pintura</option><option value="appliance">Artefacto</option><option value="garden">Jardín</option><option value="pest_control">Control de plagas</option><option value="insurance">Renovación de seguro</option><option value="other">Otra</option></select></label><label>Cada cuántos meses<input name="frequency_months" type="number" min="1" max="120" defaultValue="12" required /></label><label>Próxima fecha<input name="next_due_on" type="date" required /></label><label>Avisar con días de anticipación<input name="reminder_days" type="number" min="0" max="365" defaultValue="15" required /></label><label>Notas<input name="notes" /></label><button className="secondary-button full-field" disabled={working}>Programar mantención</button></form>{ledger.maintenance.map((item) => <article className="maintenance-row" key={item.id}><div><strong>{item.name}</strong><small>Próxima: {item.next_due_on} · cada {item.frequency_months} meses{item.last_completed_on ? ` · última ${item.last_completed_on}` : ""}</small></div><button type="button" className="text-button" disabled={working} onClick={() => completeMaintenance(item.id, item.version)}>Marcar realizada hoy</button></article>)}</div>
      <div className="rental-card danger-card"><h3>Cerrar contrato vigente</h3><p>El arrendatario no se puede reemplazar dentro de este contrato. Para registrar otro, cierra primero esta vigencia; todo el historial permanecerá asociado al arrendatario actual.</p><form className="compact-form" onSubmit={endLease}><label>Fecha efectiva<input name="ended_on" type="date" defaultValue={today} required /></label><label>Resultado<select name="status" defaultValue="ended"><option value="ended">Término normal</option><option value="terminated">Rescisión anticipada</option></select></label><label className="full-field">Motivo<input name="reason" placeholder="Término de vigencia, entrega de la propiedad…" required /></label><button className="danger-button full-field" disabled={working}>Cerrar contrato</button></form></div></div>
      <div className="rental-card"><h3>Bitácora y observaciones</h3><form className="observation-form" onSubmit={createObservation}><select name="category"><option value="payment">Pago</option><option value="tenant_report">Reporte del arrendatario</option><option value="service">Servicio</option><option value="property">Propiedad</option><option value="general">General</option></select><input name="content" placeholder="Ej. El arrendatario informó una filtración…" required /><button className="secondary-button" disabled={working}>Guardar</button></form>{ledger.observations.map((item) => <p className="observation-row" key={item.id}><strong>{item.category}</strong> {item.content}<small>{item.observed_at} · {item.created_by}</small></p>)}</div>
    </>}
  </section>;
}
