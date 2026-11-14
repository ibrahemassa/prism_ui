import { useEffect, useMemo, useState } from "react";
import { api } from "../api";
import { BASIS, type Basis, type CommissionAgent, type CommissionsPayload } from "../commissions";

/* Who is on commission and on what rule. Saved to the backend's
   commissions.json, so every machine on the LAN sees the same settings. */

type Draft = CommissionAgent & { rateText: string };

const DEFAULT_RULE = "Peşinat tamamen ödendi ve sözleşme imzalandı → komisyon ertesi ay faturalanır.";

const field = "px-2.5 py-1.5 rounded-lg border border-slate-200 bg-white text-sm outline-none focus:border-indigo-400";

function parseRate(s: string) {
  const v = Number(s.replace(/\s/g, "").replace(",", "."));
  return Number.isFinite(v) ? v : NaN;
}

export default function CommissionAgents({
  data,
  onClose,
  onSaved,
}: {
  data: CommissionsPayload;
  onClose: () => void;
  onSaved: (d: CommissionsPayload) => void;
}) {
  const [draft, setDraft] = useState<Draft[]>(() =>
    data.agents.map((a) => ({ ...a, rateText: a.rate ? String(a.rate).replace(".", ",") : "" })),
  );
  const [pick, setPick] = useState("");
  const [saving, setSaving] = useState(false);
  const [err, setErr] = useState<string | null>(null);
  const [dirty, setDirty] = useState(false);

  const close = () => {
    if (!dirty || confirm("Kaydedilmemiş değişiklikler silinsin mi?")) onClose();
  };
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && close();
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  });

  const candidates = useMemo(() => {
    const taken = new Set(draft.map((d) => d.personnel_id));
    return data.staff.filter((s) => !taken.has(s.id) && s.name).sort((a, b) => a.name.localeCompare(b.name, "tr"));
  }, [data.staff, draft]);

  const update = (i: number, patch: Partial<Draft>) => {
    setDirty(true);
    setDraft((ds) => ds.map((d, j) => (j === i ? { ...d, ...patch } : d)));
  };

  const add = () => {
    const s = data.staff.find((x) => x.id === pick);
    if (!s) return;
    setDirty(true);
    setDraft((ds) => [
      ...ds,
      {
        personnel_id: s.id, name: s.name, rate: 0, rateText: "", basis: "sale_price", need_down_payment: true, need_contract: true,
        delay_months: 1, agency_deals: true, since: "", rule: DEFAULT_RULE,
      },
    ]);
    setPick("");
  };

  const remove = (i: number) => {
    if (!confirm(`${draft[i].name} komisyon takibinden çıkarılsın mı? (Ödeme kayıtları silinmez.)`)) return;
    setDirty(true);
    setDraft((ds) => ds.filter((_, j) => j !== i));
  };

  const save = async () => {
    setErr(null);
    const agents: CommissionAgent[] = [];
    for (const { rateText, ...a } of draft) {
      const rate = rateText.trim() === "" ? 0 : parseRate(rateText);
      if (Number.isNaN(rate) || rate < 0 || rate > 100) {
        setErr(`${a.name}: oran 0 ile 100 arasında bir sayı olmalı`);
        return;
      }
      agents.push({ ...a, rate, delay_months: Math.max(0, Math.round(Number(a.delay_months) || 0)) });
    }
    setSaving(true);
    try {
      onSaved(await api.commissionAgents(agents));
      onClose();
    } catch (e) {
      setErr((e as Error).message);
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 bg-black/40 grid place-items-center p-4" role="dialog" aria-modal="true" aria-label="Danışmanlar ve kurallar">
      <div className="card w-full max-w-4xl max-h-[90vh] flex flex-col">
        <div className="flex items-center justify-between px-6 py-4 border-b border-slate-100">
          <div>
            <h2 className="text-lg font-extrabold">Danışmanlar ve kurallar</h2>
            <p className="text-xs text-slate-500 mt-0.5">
              Komisyon, satışın <b>Satış Personeli 1</b> alanındaki danışmana yazılır. Ayarlar tüm ofiste ortaktır.
            </p>
          </div>
          <button onClick={close} className="w-8 h-8 grid place-items-center rounded-lg text-slate-500 hover:bg-slate-100" aria-label="Kapat">✕</button>
        </div>

        <div className="flex-1 overflow-y-auto scrolly px-6 py-4 space-y-4">
          {draft.length === 0 && <div className="text-center text-slate-400 py-6">Henüz danışman yok — aşağıdan ekleyin.</div>}
          {draft.map((a, i) => (
            <div key={a.personnel_id} className="rounded-xl border border-slate-200 p-4">
              <div className="flex items-center justify-between gap-3 mb-3">
                <div className="font-semibold">{a.name}</div>
                <button onClick={() => remove(i)} className="text-xs font-semibold text-rose-600 hover:underline">Kaldır</button>
              </div>
              <div className="grid grid-cols-2 md:grid-cols-4 gap-3 text-sm">
                <label className="flex flex-col gap-1">
                  <span className="text-xs text-slate-500">Komisyon oranı (%)</span>
                  <input
                    className={field}
                    inputMode="decimal"
                    placeholder="örn. 1,5"
                    value={a.rateText}
                    onChange={(e) => update(i, { rateText: e.target.value })}
                  />
                </label>
                <label className="flex flex-col gap-1">
                  <span className="text-xs text-slate-500">Hesaplanacak tutar</span>
                  <select className={field} value={a.basis} onChange={(e) => update(i, { basis: e.target.value as Basis })}>
                    {(Object.keys(BASIS) as Basis[]).map((b) => (
                      <option key={b} value={b}>{BASIS[b]}</option>
                    ))}
                  </select>
                </label>
                <label className="flex flex-col gap-1">
                  <span className="text-xs text-slate-500">Hak edişten kaç ay sonra faturalanır</span>
                  <input
                    className={field}
                    type="number"
                    min={0}
                    max={24}
                    value={a.delay_months}
                    onChange={(e) => update(i, { delay_months: Number(e.target.value) })}
                  />
                </label>
                <label className="flex flex-col gap-1">
                  <span className="text-xs text-slate-500">Şu tarihten itibaren satışlar</span>
                  <input className={field} type="date" value={a.since} onChange={(e) => update(i, { since: e.target.value })} />
                </label>
              </div>
              <div className="flex flex-wrap gap-x-6 gap-y-2 mt-3 text-sm">
                <label className="flex items-center gap-2">
                  <input type="checkbox" checked={a.need_down_payment} onChange={(e) => update(i, { need_down_payment: e.target.checked })} />
                  Peşinat tamamen ödenmeli
                </label>
                <label className="flex items-center gap-2">
                  <input type="checkbox" checked={a.need_contract} onChange={(e) => update(i, { need_contract: e.target.checked })} />
                  Sözleşme imzalanmalı
                </label>
                <label className="flex items-center gap-2">
                  <input type="checkbox" checked={a.agency_deals} onChange={(e) => update(i, { agency_deals: e.target.checked })} />
                  Acenta getirdiği satışlar dahil
                </label>
              </div>
              <label className="flex flex-col gap-1 mt-3 text-sm">
                <span className="text-xs text-slate-500">Kural / not (raporlarda görünür)</span>
                <textarea className={`${field} resize-y`} rows={2} value={a.rule} onChange={(e) => update(i, { rule: e.target.value })} />
              </label>
            </div>
          ))}

          <div className="flex items-center gap-2">
            <select className={`${field} flex-1`} value={pick} onChange={(e) => setPick(e.target.value)} aria-label="Eklenecek personel">
              <option value="">Personel seç…</option>
              {candidates.map((s) => (
                <option key={s.id} value={s.id}>{s.name}</option>
              ))}
            </select>
            <button
              onClick={add}
              disabled={!pick}
              className="px-3 py-1.5 rounded-lg text-sm font-semibold border border-indigo-300 text-indigo-700 hover:bg-indigo-50 disabled:opacity-50"
            >
              + Danışman ekle
            </button>
          </div>
        </div>

        <div className="flex items-center justify-end gap-2 px-6 py-4 border-t border-slate-100">
          {err && <span className="text-sm text-rose-600 mr-auto">{err}</span>}
          <button onClick={close} className="px-3 py-2 rounded-xl text-sm font-semibold text-slate-600 hover:bg-slate-100">Vazgeç</button>
          <button
            onClick={save}
            disabled={saving}
            className="px-4 py-2 rounded-xl text-sm font-semibold bg-indigo-600 text-white hover:bg-indigo-700 disabled:opacity-60"
          >
            {saving ? "Kaydediliyor…" : "Kaydet"}
          </button>
        </div>
      </div>
    </div>
  );
}
