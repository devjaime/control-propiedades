export type User = {
  user_id: string;
  organization_id: string;
  name: string;
  email: string;
  organization: string;
  role: string;
};

export type AgentToken = {
  id: string;
  name: string;
  prefix: string;
  scopes: string[];
  status: "active" | "revoked";
  expires_at: string | null;
  last_used_at: string | null;
  created_at: string;
  token?: string;
};

export type Property = {
  id: string;
  name: string;
  property_type: string;
  usage: string;
  status: string;
  address_line: string;
  commune: string;
  region: string;
  created_at: string;
};

export type IncidentTask = {
  id: string;
  title: string;
  priority: "low" | "medium" | "high" | "critical";
  status: "pending" | "completed" | "cancelled";
  due_on: string | null;
  completed_at: string | null;
  version: number;
  visible_to_tenant: boolean;
};

export type IncidentUpdate = {
  id: string;
  update_type: "note" | "tenant_contact" | "builder_claim" | "insurance_claim" | "inspection" | "status_change";
  content: string;
  occurred_at: string;
  created_by: string;
  visible_to_tenant: boolean;
};

export type Incident = {
  id: string;
  property_id: string;
  reference: string;
  title: string;
  summary: string;
  status: "open" | "in_progress" | "resolved" | "closed";
  priority: "low" | "medium" | "high" | "critical";
  event_date: string;
  categories: string[];
  tags: string[];
  habitability: "unaffected" | "partially_affected" | "uninhabitable";
  tasks: IncidentTask[];
  updates: IncidentUpdate[];
  documents: Array<{ id: string; display_name: string; mime_type: string }>;
  created_at: string;
  updated_at: string;
  version: number;
};

export type Document = {
  id: string;
  property_id: string;
  property_name: string;
  document_type: string;
  display_name: string;
  original_name: string;
  mime_type: string;
  size_bytes: number;
  sha256: string;
  tags: string[];
  status: string;
  document_date: string | null;
  period: string | null;
  issuer: string | null;
  amount_minor: number | null;
  currency_code: string | null;
  confidentiality: "internal" | "confidential" | "restricted";
  rejection_reason: string | null;
  verified_at: string | null;
  verified_by: string | null;
  created_at: string;
  updated_at: string;
  version: number;
};

export type TenantApplicationDocument = {
  document_id: string;
  document_kind: "commercial_report" | "identity" | "income_proof" | "employment" | "guarantor" | "insurance" | "other";
  display_name: string;
  mime_type: string;
  linked_at: string;
};

export type TenantApplication = {
  id: string;
  full_name: string;
  identifier: string;
  nationality: string;
  marital_status: string;
  profession: string;
  current_address: string;
  email: string;
  phone: string;
  employer: string;
  monthly_income_minor: number | null;
  authorized_occupants: string;
  notes: string;
  status: "draft" | "under_review" | "approved" | "rejected" | "contracted" | "archived";
  commercial_report_provenance_confirmed: boolean;
  created_at: string;
  documents: TenantApplicationDocument[];
};

export type LeaseContractDraft = {
  id: string;
  application_id: string;
  draft_version: number;
  template_version: string;
  legal_basis_version: string;
  status: "draft" | "reviewed" | "approved_for_signature" | "signed_notarized" | "superseded" | "void";
  pdf_document_id: string;
  docx_document_id: string;
  signed_document_id: string;
  pdf_sha256: string;
  docx_sha256: string;
  legal_reviewed_by: string;
  created_at: string;
};

export type LeaseContractWorkflow = {
  applications: TenantApplication[];
  drafts: LeaseContractDraft[];
};

export type Lease = {
  id: string;
  tenant_id: string;
  tenant_name: string;
  tenant_email: string;
  starts_on: string;
  ends_on: string;
  status: string;
  rent_amount_minor: number;
  currency_code: string;
  payment_day: number;
  payment_installments: number;
  schedule_kind: "contractual" | "special_agreement";
  agreement_notes: string;
  installments: LeaseInstallment[];
  renewal_notice_days: number;
  renewal_review_days: number;
  vacate_notice_months: number;
  vacate_notice_days: number;
  adjustment_method: "none" | "ipc";
  adjustment_frequency_months: number;
  next_adjustment_on: string;
  adjustment_effective_on: string;
  version: number;
};

export type LeaseInstallment = {
  installment_number: number;
  due_day: number;
  expected_amount_minor: number;
};

export type Charge = {
  id: string;
  period: string;
  charge_type: string;
  concept: string;
  due_date: string;
  original_amount_minor: number;
  balance_minor: number;
  currency_code: string;
  status: string;
  notes: string;
  version: number;
};

export type Payment = {
  id: string;
  charge_id: string;
  payer_name: string;
  payment_date: string;
  amount_minor: number;
  currency_code: string;
  payment_method: string;
  bank_reference: string;
  status: string;
  timing_status: string;
  observations: string;
  document_id: string;
  created_at: string;
  version: number;
  installment_number: number;
  planned_due_date: string;
  expected_amount_minor: number;
};

export type InternalNotification = {
  id: string;
  type: string;
  title: string;
  message: string;
  due_at: string;
  status: string;
};

export type RentalObservation = {
  id: string;
  category: string;
  content: string;
  observed_at: string;
  created_by: string;
};

export type MaintenanceSchedule = {
  id: string;
  name: string;
  category: string;
  frequency_months: number;
  next_due_on: string;
  reminder_days: number;
  last_completed_on: string;
  notes: string;
  status: string;
  version: number;
};

export type ReviewCalendarItem = {
  id: string;
  period: string;
  kind: "alert" | "suggestion";
  type: string;
  title: string;
  message: string;
  scheduled_on: string;
  status: "unread" | "read" | "suggested";
};

export type RentReceipt = {
  id: string;
  document_id: string;
  folio: string;
  public_token: string;
  verification_code: string;
  status: string;
  issued_at: string;
};

export type RentReceiptHistoryItem = {
  id: string;
  document_id: string;
  folio: string;
  public_token: string;
  period: string;
  status: string;
  issued_at: string;
};

export type RentalLedger = {
  period: string;
  lease: Lease | null;
  charges: Charge[];
  payments: Payment[];
  notifications: InternalNotification[];
  observations: RentalObservation[];
  maintenance: MaintenanceSchedule[];
  review_calendar: ReviewCalendarItem[];
  receipt: RentReceipt | null;
  receipt_history: RentReceiptHistoryItem[];
};

export type TenantPortalLink = {
  id: string;
  property_id: string;
  public_token: string;
  status: string;
  url: string;
  expires_at: string | null;
  last_accessed_at: string | null;
  created_at: string;
  access_key?: string;
  access_events: Array<{
    id: string;
    credential_kind: "owner" | "tenant" | "legacy_key";
    accessed_at: string;
  }>;
};

export type TenantPortalIncident = {
  id: string;
  reference: string;
  title: string;
  summary: string;
  status: "open" | "in_progress" | "resolved";
  priority: "low" | "medium" | "high" | "critical";
  habitability: "unaffected" | "partially_affected" | "uninhabitable";
  event_date: string;
  tasks: Array<{
    id: string;
    title: string;
    priority: string;
    status: string;
    due_on: string | null;
  }>;
  updates: Array<{
    id: string;
    update_type: string;
    content: string;
    occurred_at: string;
  }>;
  documents: Array<{
    id: string;
    display_name: string;
    original_name: string;
    mime_type: string;
    document_date: string;
    url: string;
  }>;
};
