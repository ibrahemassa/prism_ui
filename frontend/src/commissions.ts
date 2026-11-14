import { money } from "./api";

/* Agent commissions. The backend works every figure out (see
   backend/internal/commissions); this file only names things and adds up. */

export type Basis = "sale_price" | "list_price";

export type CommissionAgent = {
  personnel_id: string;
  name: string;
  rate: number; // percent
  basis: Basis;
  need_down_payment: boolean;
  need_contract: boolean;
  delay_months: number;
  agency_deals: boolean;
  since: string; // YYYY-MM-DD or ""
  rule: string;
};

export type Payout = { agent_id: string; paid_at: string; amount: number; currency: string; month: string };

export type CommissionStatus = "waiting" | "scheduled" | "due" | "overdue" | "paid" | "cancelled" | "clawback";

export type CommissionLine = {
  deal_id: string;
  agent_id: string;
  agent: string;
  customer: string;
  unit: string;
  unit_type: string;
  agency: string;
  stage: string;
  currency: string;
  deal_date: string;
  sale_price: number;
  list_price: number;
  base: number;
  rate: number;
  amount: number;
  contracted: boolean;
  contract_at: string;
  down_required: number;
  down_paid: number;
  down_paid_at: string;
  eligible_at: string;
  bill_month: string;
  status: CommissionStatus;
  missing: string[];
  notes: string[];
  payout?: Payout;
};

export type CommissionsPayload = {
  month: string; // this month, YYYY-MM
  agents: CommissionAgent[];
  lines: CommissionLine[];
  staff: { id: string; name: string }[];
};

export const STATUS: Record<CommissionStatus, { label: string; bg: string; fg: string }> = {
  waiting: { label: "Bekliyor", bg: "#fffbeb", fg: "#b45309" },
  scheduled: { label: "Planlandı", bg: "#eff6ff", fg: "#1d4ed8" },
  due: { label: "Faturalanacak", bg: "#eef2ff", fg: "#4338ca" },
  overdue: { label: "Gecikmiş", bg: "#fff1f2", fg: "#be123c" },
  paid: { label: "Ödendi", bg: "#ecfdf5", fg: "#047857" },
  cancelled: { label: "İptal", bg: "#f1f5f9", fg: "#475569" },
  clawback: { label: "Geri alınacak", bg: "#fff1f2", fg: "#be123c" },
};

export const MISSING: Record<string, string> = {
  contract: "Sözleşme imzalanmadı",
  down_payment: "Peşinat tamamlanmadı",
  date: "Hak ediş tarihi yok",
};

export const NOTES: Record<string, string> = {
  no_down_plan: "Peşinat planı yok — fiyatın tamamı peşinat sayıldı",
  contract_no_date: "Prism'de sözleşme işaretli ama tarihi girilmemiş",
  no_list_price: "Liste fiyatı yok — satış fiyatı kullanıldı",
  no_receipt_date: "Peşinat Prism'de tamam görünüyor, tahsilat tarihleri tutmuyor",
  payout_differs: "Ödenen tutar bugünkü hesaptan farklı",
};

export const BASIS: Record<Basis, string> = { sale_price: "Satış fiyatı", list_price: "Liste fiyatı" };

export const STAGE: Record<string, string> = { sale: "Satıldı", reserved: "Rezerve", cancelled: "İptal", negotiation: "Görüşme" };

/** What the line is worth: what was paid if it was, else today's figure. */
export const worth = (l: CommissionLine) =>
  l.payout ? { amount: l.payout.amount, currency: l.payout.currency } : { amount: l.amount, currency: l.currency };

/** Counts toward its billing month: earned and not called off. */
export const billed = (l: CommissionLine) => l.status !== "waiting" && l.status !== "cancelled" && l.status !== "clawback";

/** In place of a YYYY-MM month: every billing month at once. */
export const ALL = "all";

/** Billed in the period — the month, or any month for ALL. */
export const inPeriod = (l: CommissionLine, month: string) => billed(l) && (month === ALL || l.bill_month === month);

export type ByCur = Record<string, number>;

export function sumBy(lines: CommissionLine[], pick: (l: CommissionLine) => { amount: number; currency: string } | null): ByCur {
  const out: ByCur = {};
  for (const l of lines) {
    const v = pick(l);
    if (v && v.amount) out[v.currency || "TRY"] = (out[v.currency || "TRY"] || 0) + v.amount;
  }
  return out;
}

/** "₺150.000" — or "₺150.000 + $3.000" when the lines span currencies. */
export function moneyCur(m: ByCur) {
  const keys = Object.keys(m).filter((k) => m[k]).sort((a, b) => (a === "TRY" ? -1 : b === "TRY" ? 1 : a.localeCompare(b)));
  if (keys.length === 0) return "—";
  return keys.map((k) => money(m[k], k)).join(" + ");
}

export function rateLabel(r: number) {
  return `%${(r || 0).toLocaleString("tr-TR", { maximumFractionDigits: 2 })}`;
}

export function monthLabel(m: string) {
  if (!m) return "—";
  if (m === ALL) return "Tüm zamanlar";
  const [y, mm] = m.split("-").map(Number);
  return new Date(y, (mm || 1) - 1, 1).toLocaleDateString("tr-TR", { month: "long", year: "numeric" });
}

export function addMonth(m: string, n: number) {
  const [y, mm] = m.split("-").map(Number);
  const d = new Date(y, mm - 1 + n, 1);
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}`;
}

export function today() {
  const d = new Date();
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
}

export function exportUrl(month: string, agent?: string) {
  const q = new URLSearchParams({ month });
  if (agent) q.set("agent", agent);
  return `/api/commissions/export.xlsx?${q}`;
}

export function printUrl(month: string, agent?: string) {
  const q = new URLSearchParams({ ay: month });
  if (agent) q.set("danisman", agent);
  return `/komisyonlar/pdf?${q}`;
}
