const base = ""; // dev proxy forwards /api -> backend

/** The server's error text (plain or {error}) becomes the thrown message. */
async function failure(res: Response) {
  const text = (await res.text().catch(() => "")).trim();
  let msg = text;
  try {
    msg = JSON.parse(text).error || text;
  } catch {}
  return new Error(msg || `${res.status} ${res.statusText}`);
}

async function get<T>(path: string): Promise<T> {
  const res = await fetch(base + path);
  if (!res.ok) throw await failure(res);
  return res.json();
}

async function post<T>(path: string, body: unknown): Promise<T> {
  const res = await fetch(base + path, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) });
  if (!res.ok) throw await failure(res);
  return res.json();
}

type CommissionsPayload = import("./commissions").CommissionsPayload;

export const api = {
  stats: () => get<import("./types").Stats>("/api/stats"),
  leads: (q = "", limit = 50, offset = 0) =>
    get<{ rows: import("./types").Lead[]; total: number }>(
      `/api/leads?q=${encodeURIComponent(q)}&limit=${limit}&offset=${offset}`,
    ),
  lead: (id: string) => get<import("./types").LeadDetail>(`/api/leads/${id}`),
  units: () => get<{ rows: import("./types").Unit[]; total: number }>("/api/units"),
  deals: () => get<{ rows: import("./types").Deal[]; total: number }>("/api/deals"),
  deal: (id: string) => get<import("./types").DealDetail>(`/api/deals/${id}`),
  agents: () => get<{ rows: import("./types").Agent[]; total: number }>("/api/agents"),
  agencies: () => get<{ rows: import("./types").Agency[]; total: number }>("/api/agencies"),
  reports: () => get<import("./types").Reports>("/api/reports"),
  reportFull: () => get<import("./types").FullReport>("/api/reports/full"),
  syncStatus: () => get<import("./types").SyncStatus>("/api/sync/status"),
  syncRun: async () => {
    const res = await fetch(base + "/api/sync/run", { method: "POST" });
    const body = await res.json().catch(() => ({}));
    if (!res.ok) throw new Error(body?.error || `${res.status} ${res.statusText}`);
    return body as { result: Record<string, unknown>; status: import("./types").SyncStatus };
  },
  // commissions: the two POSTs save the local commission settings, never Prism
  commissions: () => get<CommissionsPayload>("/api/commissions"),
  commissionAgents: (agents: import("./commissions").CommissionAgent[]) =>
    post<CommissionsPayload>("/api/commissions/agents", { agents }),
  commissionPaid: (deal_ids: string[], paid: boolean, paid_at?: string) =>
    post<CommissionsPayload>("/api/commissions/payouts", { deal_ids, paid, paid_at }),
};

export function money(n: number, cur = "TRY") {
  if (!n) return "—";
  return new Intl.NumberFormat("tr-TR", {
    style: "currency",
    currency: cur || "TRY",
    maximumFractionDigits: 0,
  }).format(n);
}

export function date(s: string) {
  if (!s || s.startsWith("1900") || s.startsWith("0001")) return "—";
  const d = new Date(s);
  if (isNaN(+d)) return "—";
  return d.toLocaleDateString("tr-TR", { year: "numeric", month: "short", day: "numeric" });
}

/** "B2-13" — block and unit number as one label, the way the sales team says it. */
export function unitLabel(block?: string | null, number?: string | null) {
  const b = (block || "").trim();
  const n = (number || "").trim();
  if (!b && !n) return "—";
  return b && n ? `${b}-${n}` : b || n;
}

/** Turkish-aware fold for search: "Şahin" and "sahin" meet in the middle. */
export function fold(s: string | number | null | undefined) {
  return String(s ?? "")
    .toLocaleLowerCase("tr")
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/g, "")
    .replace(/ı/g, "i");
}

/** Every word of the query must appear somewhere in the joined fields. */
export function hit(q: string, ...fields: (string | number | null | undefined)[]) {
  const words = fold(q).trim().split(/\s+/).filter(Boolean);
  if (words.length === 0) return true;
  const hay = fold(fields.map((f) => f ?? "").join(" "));
  return words.every((w) => hay.includes(w));
}

/** "₺2,15 Mlr" / "₺315 M" / "₺27,3 B" — for chart labels and headlines. */
export function moneyShort(n: number) {
  if (!n) return "—";
  const abs = Math.abs(n);
  const f = (v: number, d: number) => v.toLocaleString("tr-TR", { maximumFractionDigits: d, minimumFractionDigits: 0 });
  if (abs >= 1e9) return `₺${f(n / 1e9, 2)} Mlr`;
  if (abs >= 1e6) return `₺${f(n / 1e6, abs >= 1e8 ? 0 : 1)} M`;
  if (abs >= 1e3) return `₺${f(n / 1e3, 0)} B`;
  return `₺${f(n, 0)}`;
}

export function num(n: number, d = 0) {
  return (n || 0).toLocaleString("tr-TR", { maximumFractionDigits: d, minimumFractionDigits: 0 });
}
