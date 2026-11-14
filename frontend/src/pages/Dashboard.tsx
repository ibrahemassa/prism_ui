import { api, money } from "../api";
import { Card, PageHead, Loading, ErrorBox, useApi } from "../ui/kit";
import type { LabeledCount } from "../types";
import { BRAND, dot } from "../brand";

function Stat({ label, value, sub, tone = "#4f46e5" }: { label: string; value: string; sub?: string; tone?: string }) {
  return (
    <Card className="p-5">
      <div className="text-sm text-slate-500 font-medium">{label}</div>
      <div className="text-3xl font-extrabold mt-2 tabular" style={{ color: tone }}>
        {value}
      </div>
      {sub && <div className="text-xs text-slate-400 mt-1">{sub}</div>}
    </Card>
  );
}

function BarList({ title, rows, tone }: { title: string; rows: LabeledCount[]; tone: string }) {
  const max = Math.max(1, ...rows.map((r) => r.count));
  return (
    <Card className="p-5">
      <div className="font-bold mb-4">{title}</div>
      <div className="space-y-3">
        {rows.map((r) => (
          <div key={r.label}>
            <div className="flex justify-between text-sm mb-1">
              <span className="text-slate-600 truncate pr-2">{r.label}</span>
              <span className="font-semibold tabular">{r.count}</span>
            </div>
            <div className="h-2 rounded-full bg-slate-100 overflow-hidden">
              <div className="h-full rounded-full" style={{ width: `${(r.count / max) * 100}%`, background: tone }} />
            </div>
          </div>
        ))}
        {rows.length === 0 && <div className="text-sm text-slate-400">Veri yok</div>}
      </div>
    </Card>
  );
}

export default function Dashboard() {
  const { data, error, loading } = useApi(() => api.stats(), []);
  if (loading) return <Loading />;
  if (error) return <ErrorBox error={error} />;
  const s = data!;
  const soldPct = s.units ? Math.round((s.units_sold / s.units) * 100) : 0;
  const reservedPct = s.units ? Math.round((s.units_reserved / s.units) * 100) : 0;
  return (
    <>
      <PageHead title="Panel" subtitle={dot(BRAND.project, "genel bakış")} />
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-5">
        <Stat label="Müşteriler" value={s.customers.toLocaleString("tr-TR")} sub="kayıtlı müşteri" />
        <Stat label="Daireler" value={s.units.toString()} sub={`${s.units_sold} satıldı · ${s.units_reserved} rezerve · ${s.units_free} boşta`} tone="#0ea5e9" />
        <Stat label="Açık satışlar" value={(s.deals_sale + s.deals_reserved).toString()} sub={`${s.deals_sale} satıldı · ${s.deals_reserved} rezerve · ${s.deals} aktivite`} tone="#059669" />
        <Stat label="Acentalar" value={s.agencies.toString()} sub={`${s.agents} personel`} tone="#d97706" />
      </div>

      <Card className="p-5 mb-5">
        <div className="flex justify-between items-center mb-3">
          <div className="font-bold">Daire doluluk</div>
          <div className="text-sm text-slate-500 tabular">{soldPct}% satıldı · {reservedPct}% rezerve</div>
        </div>
        <div className="h-3 rounded-full bg-slate-100 overflow-hidden flex">
          <div className="h-full bg-indigo-500" style={{ width: `${soldPct}%` }} title={`${s.units_sold} satıldı`} />
          <div className="h-full bg-amber-400" style={{ width: `${reservedPct}%` }} title={`${s.units_reserved} rezerve`} />
        </div>
        <div className="flex gap-5 mt-3 text-sm text-slate-500">
          <span><b className="text-indigo-600">{s.units_sold}</b> satıldı</span>
          <span><b className="text-amber-600">{s.units_reserved}</b> rezerve</span>
          <span><b className="text-slate-700">{s.units_free}</b> boşta</span>
        </div>
      </Card>

      <div className="grid md:grid-cols-2 gap-5">
        <BarList title="Personele göre açık satışlar" rows={s.sales_by_agent} tone="#4f46e5" />
        <BarList title="Müşteri kategorileri" rows={s.leads_by_category} tone="#0ea5e9" />
      </div>
    </>
  );
}
