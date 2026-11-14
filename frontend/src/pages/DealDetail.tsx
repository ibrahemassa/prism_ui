import { Link, useParams } from "react-router-dom";
import { api, money, date, unitLabel } from "../api";
import { Card, Loading, ErrorBox, StatusChip, useApi } from "../ui/kit";

function Field({ label, value }: { label: string; value?: string }) {
  return (
    <div className="flex justify-between gap-4 py-2 border-b border-slate-50 last:border-0">
      <span className="text-slate-500 text-sm">{label}</span>
      <span className="text-sm font-medium text-right">{value || "—"}</span>
    </div>
  );
}

export default function DealDetail() {
  const { id } = useParams();
  const { data: d, error, loading } = useApi(() => api.deal(id!), [id]);
  if (loading) return <Loading />;
  if (error) return <ErrorBox error={error} />;

  return (
    <>
      <Link to="/satislar" className="text-sm text-indigo-600 hover:underline">← Satışlar</Link>

      <div className="flex items-start justify-between mt-3 mb-6 gap-4">
        <div>
          <div className="flex items-center gap-3">
            <h1 className="text-2xl font-extrabold">
              Blok {d!.block || "—"} · No {d!.unit_number || "—"}
            </h1>
            <StatusChip status={d!.stage} />
          </div>
          <div className="text-slate-500 mt-1">
            {d!.unit_type} · {d!.project}
          </div>
          <Link to={`/musteriler/${d!.customer_id}`} className="text-indigo-600 hover:underline text-sm mt-1 inline-block">
            {d!.customer} →
          </Link>
        </div>
        <div className="text-right">
          <div className="text-xs text-slate-400">Satış Fiyatı</div>
          <div className="text-2xl font-extrabold tabular text-indigo-600">{money(d!.sale_price, d!.currency)}</div>
        </div>
      </div>

      <div className="grid md:grid-cols-3 gap-5">
        <Card className="p-5">
          <div className="font-bold mb-2">Finansal</div>
          <Field label="Liste Fiyatı" value={money(d!.list_price, d!.currency)} />
          <Field label="Satış Fiyatı" value={money(d!.sale_price, d!.currency)} />
          <Field label="Peşinat" value={money(d!.down_payment, d!.currency)} />
          <Field label="Vadeli Tutar" value={money(d!.financed_amount, d!.currency)} />
          <div className="flex justify-between gap-4 py-2 border-b border-slate-50">
            <span className="text-slate-500 text-sm">Kapora</span>
            <span className="text-sm font-medium text-right">
              {d!.kapora_paid > 0 ? (
                <span className="chip" style={{ background: "#ecfdf5", color: "#047857" }}>✓ {money(d!.kapora_paid, d!.currency)}</span>
              ) : (
                <span className="chip" style={{ background: "#fff1f2", color: "#be123c" }}>ödenmedi</span>
              )}
            </span>
          </div>
          <Field label="Tahsil edilen" value={money(d!.paid_total, d!.currency)} />
          <Field label="Kalan borç" value={money(d!.remaining, d!.currency)} />
          {d!.commission_amount > 0 && <Field label="Komisyon" value={`${money(d!.commission_amount, d!.currency)} (%${d!.commission_pct})`} />}
        </Card>
        <Card className="p-5">
          <div className="font-bold mb-2">Tarihler</div>
          <Field label="Rezerve" value={date(d!.reserved_at)} />
          <Field label="Sözleşme" value={date(d!.contract_at)} />
          <Field label="Aktivite" value={date(d!.activity_at)} />
          {d!.cancelled_at && <Field label="İptal" value={date(d!.cancelled_at)} />}
        </Card>
        <Card className="p-5">
          <div className="font-bold mb-2">Daire</div>
          <Field label="Proje" value={d!.project} />
          <Field label="Daire" value={unitLabel(d!.block, d!.unit_number)} />
          <Field label="Tip" value={d!.unit_type} />
          <Field label="Brüt / Net m²" value={`${d!.unit_gross_m2 || "—"} / ${d!.unit_net_m2 || "—"}`} />
          <Field label="Manzara" value={d!.unit_view} />
          <Field label="Danışman" value={d!.agent} />
          <Field label="Acenta" value={d!.agency} />
        </Card>
      </div>

      {d!.cancel_reason && (
        <Card className="p-5 mt-5 border-rose-200 bg-rose-50">
          <div className="font-bold text-rose-700 mb-1">İptal nedeni</div>
          <div className="text-sm text-rose-600">{d!.cancel_reason}</div>
        </Card>
      )}

      <div className="flex items-center justify-between mt-8 mb-3">
        <div className="font-bold">Ödeme planı ({d!.payment_plan.length} kalem)</div>
        {d!.plan_total > 0 && <div className="text-sm text-slate-500 tabular">Toplam: {money(d!.plan_total, d!.currency)}</div>}
      </div>
      {d!.payment_plan.length === 0 ? (
        <Card className="p-5 text-slate-400 text-sm">Bu satış için ödeme planı bulunmuyor.</Card>
      ) : (
        <Card className="overflow-hidden">
          <div className="max-h-[520px] overflow-y-auto scrolly">
            <table className="w-full text-sm">
              <thead className="sticky top-0 bg-white">
                <tr className="text-left text-slate-500 border-b border-slate-100">
                  <th className="px-5 py-3 font-semibold">#</th>
                  <th className="px-5 py-3 font-semibold">Tarih</th>
                  <th className="px-5 py-3 font-semibold">Kalem</th>
                  <th className="px-5 py-3 font-semibold">Ödeme</th>
                  <th className="px-5 py-3 font-semibold text-right">Tutar</th>
                  <th className="px-5 py-3 font-semibold text-right">Ödenen</th>
                  <th className="px-5 py-3 font-semibold">Not</th>
                </tr>
              </thead>
              <tbody>
                {d!.payment_plan.map((p, i) => {
                  const full = p.paid >= p.amount && p.amount > 0;
                  const part = p.paid > 0 && !full;
                  return (
                    <tr key={p.id} className="rowhover border-b border-slate-50 last:border-0">
                      <td className="px-5 py-2.5 text-slate-400 tabular">{i + 1}</td>
                      <td className="px-5 py-2.5">{date(p.date)}</td>
                      <td className="px-5 py-2.5">
                        <span className="chip" style={{ background: p.kind === "Peşinat" ? "#eef2ff" : "#f1f5f9", color: p.kind === "Peşinat" ? "#4338ca" : "#475569" }}>
                          {p.kind || "—"}{p.interim ? " · ara" : ""}
                        </span>
                      </td>
                      <td className="px-5 py-2.5 text-slate-600">{p.type || "—"}</td>
                      <td className="px-5 py-2.5 text-right tabular font-medium">{money(p.amount, p.currency)}</td>
                      <td className="px-5 py-2.5 text-right tabular">
                        {p.paid > 0 ? (
                          <span style={{ color: full ? "#047857" : "#b45309" }}>
                            {full ? "✓ " : ""}{money(p.paid, p.currency)}
                          </span>
                        ) : (
                          <span className="text-slate-300">—</span>
                        )}
                      </td>
                      <td className="px-5 py-2.5 text-slate-500">{p.note}</td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </Card>
      )}

      <div className="flex items-center justify-between mt-8 mb-3">
        <div className="font-bold">Tahsilatlar ({d!.receipts.length})</div>
        {d!.paid_total > 0 && <div className="text-sm text-emerald-600 font-semibold tabular">Toplam tahsil: {money(d!.paid_total, d!.currency)}</div>}
      </div>
      {d!.receipts.length === 0 ? (
        <Card className="p-5 text-slate-400 text-sm">Henüz tahsilat yok.</Card>
      ) : (
        <Card className="overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="text-left text-slate-500 border-b border-slate-100">
                <th className="px-5 py-3 font-semibold">Tarih</th>
                <th className="px-5 py-3 font-semibold">Ödeme</th>
                <th className="px-5 py-3 font-semibold">Hesap</th>
                <th className="px-5 py-3 font-semibold text-right">Tutar</th>
                <th className="px-5 py-3 font-semibold">Not</th>
              </tr>
            </thead>
            <tbody>
              {d!.receipts.map((r) => (
                <tr key={r.id} className="rowhover border-b border-slate-50 last:border-0">
                  <td className="px-5 py-2.5">{date(r.date)}</td>
                  <td className="px-5 py-2.5">
                    {r.type || "—"}
                    {r.deposit && <span className="chip ml-2" style={{ background: "#ecfdf5", color: "#047857" }}>kapora</span>}
                  </td>
                  <td className="px-5 py-2.5 text-slate-600">{r.account || "—"}</td>
                  <td className="px-5 py-2.5 text-right tabular font-medium text-emerald-600">{money(r.amount, r.currency)}</td>
                  <td className="px-5 py-2.5 text-slate-500">{r.note}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </Card>
      )}
    </>
  );
}
