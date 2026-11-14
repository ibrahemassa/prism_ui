export interface Stats {
  customers: number;
  units: number;
  units_sold: number; // Prism's last movement on the unit: Satıldı
  units_reserved: number; // Rezerve
  units_free: number; // Satılık
  deals: number;
  deals_sale: number; // live deals, hareket Satıldı
  deals_reserved: number; // live deals, hareket Rezerve
  agencies: number;
  agents: number;
  sales_by_agent: LabeledCount[];
  leads_by_category: LabeledCount[];
}
export interface LabeledCount {
  label: string;
  count: number;
}
export interface Lead {
  id: string;
  name: string;
  phone: string;
  email: string;
  category: string;
  agent: string;
  office: string;
  agency: string;
  result: string;
  city: string;
  created_at: string;
}
export interface Unit {
  id: string;
  project: string;
  block: string;
  number: string;
  type: string;
  gross_m2: number;
  net_m2: number;
  price: number;
  currency: string;
  view: string;
  status: "available" | "reserved" | "sold" | "unavailable";
}
export interface Deal {
  id: string;
  customer_id: string;
  customer: string;
  unit_id: string;
  block: string;
  unit_number: string;
  unit_type: string;
  stage: "negotiation" | "reserved" | "sale" | "cancelled";
  sale_price: number;
  down_payment: number;
  currency: string;
  agent: string;
  agency: string;
  reserved_at: string;
  contract_at: string;
  activity_at: string;
  cancelled_at: string; // only set when the deal is in T_AktiviteIptal
  kapora_paid: number;
  paid_total: number;
  remaining: number;
  paid_pct: number;
}
export interface Agent {
  id: string;
  name: string;
  email: string;
  active: boolean;
}
export interface Agency {
  id: string;
  name: string;
  phone: string;
  email: string;
  city: string;
  rate: number;
}
export interface Activity {
  date: string;
  agent: string;
  channel: string;
  notes: string;
  result: string;
  reminder: string;
  has_audio: boolean;
}
export interface LeadDetail {
  id: string;
  name: string;
  phone: string;
  phone2: string;
  email: string;
  category: string;
  agent: string;
  office: string;
  agency: string;
  result: string;
  national_id: string;
  birthday: string;
  nationality: string;
  mother: string;
  father: string;
  gender: string;
  city: string;
  district: string;
  address: string;
  occupation: string;
  source: string;
  purpose: string;
  income: string;
  created_at: string;
  deals: Deal[];
  activities: Activity[];
  crm?: CRMInfo;
}
export interface PaymentRow {
  id: number;
  date: string;
  amount: number;
  paid: number;
  currency: string;
  kind: string;
  type: string;
  interim: boolean;
  note: string;
}
export interface ReceiptRow {
  id: number;
  date: string;
  amount: number;
  currency: string;
  type: string;
  deposit: boolean;
  account: string;
  note: string;
}
export interface CRMStage {
  from: string;
  to: string;
  at: string;
}
export interface CRMInfo {
  in_crm: boolean;
  phone: string;
  name: string;
  source: string;
  status: string;
  created_at: string;
  first_contact_at: string;
  summary: string;
  stages: CRMStage[];
}
// Everything here reads Prism's hareket tipi (Rezerve / Satıldı) on live deals;
// the contract flag is never used, and cancelled deals are out of all money.
export interface Reports {
  funnel: { total: number; reserved: number; sale: number; cancelled: number; negotiation: number; cancel_rate: number };
  collection: {
    pipeline_value: number; // Σ sale price of live deals
    collected: number; // receipts on live deals — cancelled ones are refunded, so out
    to_collect: number; // pipeline_value − collected (Prism's "Kalan")
    remaining: number; // = to_collect, older name
    refunds_owed: number; // receipts sitting on cancelled deals
    kapora_paid: number;
    collect_pct: number;
    deals_with_kapora: number;
  };
  sales_by_month: { month: string; count: number; reserved: number; sold: number; revenue: number }[]; // by RezerveTarihi (AktiviteTarihi where unset)
  by_agent: { agent: string; reserved: number; sales: number; cancelled: number; revenue: number; collected: number }[]; // agents holding a live deal
  by_block: { label: string; count: number; reserved: number; sold: number; revenue: number }[];
  by_type: { label: string; total: number; available: number; reserved: number; sold: number; closed: number }[]; // stock by unit type
}
export interface DealDetail extends Deal {
  project: string;
  unit_gross_m2: number;
  unit_net_m2: number;
  unit_view: string;
  unit_bedrooms: string;
  list_price: number;
  financed_amount: number;
  commission_pct: number;
  commission_amount: number;
  cancelled_at: string;
  cancel_reason: string;
  payment_plan: PaymentRow[];
  plan_total: number;
  receipts: ReceiptRow[];
  paid_total: number;
}

// ---------- long-form printed report (/api/reports/full) ----------
export interface LabeledMoney { label: string; count: number; revenue: number }
export interface M2Group {
  label: string; m2_sold: number; m2_reserved: number; m2_available: number; m2_total: number;
  net_sold: number; net_reserved: number; net_available: number;
  revenue: number; revenue_sold: number; revenue_reserved: number; list_value: number; stock_list: number; collected: number; deals: number;
}
export interface FullReport {
  generated_at: string;
  summary: {
    deals: number; live: number; sold: number; reserved: number; cancelled: number;
    customers: number; buyers: number;
    units: number; units_sold: number; units_reserved: number; units_available: number; units_closed: number;
    m2_total: number; m2_sold: number; m2_reserved: number; m2_available: number;
    pipeline_value: number; list_value: number; discount: number;
    collected: number; to_collect: number; kapora_paid: number; refunds_owed: number;
    avg_price: number; price_per_m2: number; stock_value: number;
    agencies_active: number; via_agency: number;
  };
  daily: { date: string; reserved: number; sold: number; cancelled: number; revenue: number; receipts: number }[];
  receipts: { by_month: LabeledMoney[]; by_method: LabeledMoney[]; by_account: LabeledMoney[]; count: number; total: number; largest: number; average: number };
  schedule: { month: string; due: number; paid: number; remaining: number; count: number }[];
  down_payment: { planned: number; paid: number; installments: number; bank_loan: number; avg_down_pct: number };
  agents: { agent: string; reserved: number; sales: number; cancelled: number; revenue: number; collected: number; m2: number; list_value: number; customers: number }[];
  agencies: { agency: string; reserved: number; sold: number; cancelled: number; revenue: number; collected: number; m2: number; commission_pct: number; commission: number; types: LabeledCount[] }[];
  sources: { source: string; reserved: number; sold: number; cancelled: number; revenue: number; collected: number }[];
  ad_sales: {
    since: string; count: number; reserved: number; sold: number; cancelled: number; revenue: number; collected: number; all_since: number;
    deals: { id: string; date: string; customer: string; unit: string; unit_type: string; stage: string; price: number; collected: number; source: string; agent: string; agency: string }[];
    by_source: LabeledMoney[]; by_agent: LabeledMoney[]; by_type: LabeledCount[];
  };
  blocks: { label: string; units: number; sold: number; reserved: number; available: number; closed: number; m2_total: number; m2_live: number; m2_sold: number; m2_reserved: number; m2_available: number; revenue: number; collected: number; stock_list: number }[];
  types: { label: string; total: number; sold: number; reserved: number; available: number; closed: number; avg_m2: number; m2_total: number; m2_live: number; m2_sold: number; m2_reserved: number; m2_available: number; revenue: number; avg_price: number; price_per_m2: number; stock_list: number; collected: number }[];
  m2: {
    net_total: number; net_sold: number; net_reserved: number; net_available: number; net_closed: number; brut_closed: number;
    cancelled_m2: number; ask_per_m2: number; sold_per_m2: number; res_per_m2: number; list_per_m2: number;
    by_month: { month: string; m2_sold: number; m2_reserved: number; count: number; revenue: number }[];
    by_type: M2Group[]; by_block: M2Group[]; by_agent: M2Group[]; by_agency: M2Group[];
    largest: { unit: string; unit_type: string; customer: string; stage: string; brut_m2: number; net_m2: number; price: number }[];
  };
  customers: { total: number; companies: number; buyers: number; by_category: LabeledCount[]; by_source: LabeledCount[]; by_city: LabeledCount[]; by_week: LabeledCount[]; by_agent: LabeledCount[] };
  cancellations: { total: number; value: number; refunds: number; avg_days: number; by_month: LabeledMoney[]; by_reason: LabeledCount[]; by_kind: LabeledCount[]; by_agent: LabeledCount[]; by_type: LabeledCount[]; with_deposit: number; deposit_amount: number };
  plans: { by_type: LabeledMoney[]; npv_total: number; discount: number; avg_discount_pct: number; avg_down_pct: number; avg_term_months: number; installments: number };
}

/** GET /api/sync/status — the CRM push, as the backend last saw it. */
export type SyncStatus = {
  enabled: boolean;
  url: string;
  interval: string;
  dry_run: boolean;
  running: boolean;
  last_started?: string;
  last_ok?: string;
  last_error?: string;
  last_error_at?: string;
  last_result?: Record<string, unknown>;
  counts?: Record<string, number>;
  duration_ms?: number;
  runs: number;
};
