import { Link, useParams } from "react-router-dom";
import { api, money, date, unitLabel } from "../api";
import { Card, Loading, ErrorBox, StatusChip, useApi } from "../ui/kit";
import { useSettings } from "../ui/settings";

function Field({ label, value }: { label: string; value?: string }) {
  return (
    <div className="flex justify-between gap-4 py-2 border-b border-slate-50 last:border-0">
      <span className="text-slate-500 text-sm">{label}</span>
      <span className="text-sm font-medium text-right">{value || "—"}</span>
    </div>
  );
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <Card className="p-5">
      <div className="font-bold mb-2">{title}</div>
      {children}
    </Card>
  );
}

export default function LeadDetail() {
  const { id } = useParams();
  const { prismOnly } = useSettings();
  const { data: d, error, loading } = useApi(() => api.lead(id!), [id]);
  if (loading) return <Loading />;
  if (error) return <ErrorBox error={error} />;
  const crm = d!.crm;
  const initials = d!.name.split(/\s+/).filter(Boolean).slice(0, 2).map((s) => s[0]).join("").toUpperCase();

  return (
    <>
      <Link to="/musteriler" className="text-sm text-indigo-600 hover:underline">← Müşteriler</Link>

      <div className="flex items-center gap-4 mt-3 mb-6">
        <div className="w-16 h-16 rounded-2xl bg-indigo-100 text-indigo-700 grid place-items-center text-xl font-extrabold">{initials}</div>
        <div>
          <h1 className="text-2xl font-extrabold">{d!.name}</h1>
          <div className="flex gap-2 mt-1 flex-wrap">
            {d!.category && <span className="chip" style={{ background: "#eef2ff", color: "#4338ca" }}>{d!.category}</span>}
            {d!.agent && <span className="chip" style={{ background: "#f1f5f9", color: "#475569" }}>{d!.agent}</span>}
            {d!.office && <span className="chip" style={{ background: "#ecfeff", color: "#0e7490" }}>{d!.office}</span>}
          </div>
        </div>
      </div>

      <div className="grid md:grid-cols-3 gap-5">
        <Section title="İletişim">
          <Field label="Telefon" value={d!.phone} />
          <Field label="Telefon 2" value={d!.phone2} />
          <Field label="E-posta" value={d!.email} />
        </Section>
        <Section title="Kimlik">
          <Field label="TC Kimlik" value={d!.national_id} />
          <Field label="Doğum" value={date(d!.birthday)} />
          <Field label="Uyruk" value={d!.nationality} />
          <Field label="Cinsiyet" value={d!.gender} />
          <Field label="Anne / Baba" value={[d!.mother, d!.father].filter(Boolean).join(" / ")} />
        </Section>
        <Section title="Adres & Profil">
          <Field label="Şehir" value={d!.city} />
          <Field label="İlçe" value={d!.district} />
          <Field label="Meslek" value={d!.occupation} />
          <Field label="Kaynak" value={d!.source} />
          <Field label="Sonuç" value={d!.result} />
          <Field label="Kayıt" value={date(d!.created_at)} />
        </Section>
      </div>

      {!prismOnly && crm && (
        <Card className="p-5 mt-5" style={{ borderColor: crm.in_crm ? "#c7d2fe" : undefined }}>
          <div className="flex items-center gap-2 mb-3">
            <span className="chip" style={{ background: "#eef2ff", color: "#4338ca" }}>● CRM (canlı)</span>
            {crm.in_crm ? (
              <span className="text-sm text-slate-500">Bu kişi CRM'de kayıtlı</span>
            ) : (
              <span className="text-sm text-slate-400">CRM'de bulunamadı</span>
            )}
          </div>
          {crm.in_crm && (
            <>
              <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
                <div><div className="text-slate-400 text-xs">Kaynak</div><div className="font-semibold">{crm.source || "—"}</div></div>
                <div><div className="text-slate-400 text-xs">Durum</div><div className="font-semibold">{crm.status || "—"}</div></div>
                <div><div className="text-slate-400 text-xs">İlk temas</div><div className="font-semibold">{date(crm.first_contact_at)}</div></div>
                <div><div className="text-slate-400 text-xs">CRM kaydı</div><div className="font-semibold">{date(crm.created_at)}</div></div>
              </div>
              {crm.stages.length > 0 && (
                <div className="mt-4">
                  <div className="text-xs text-slate-400 mb-2">Aşama geçmişi</div>
                  <div className="flex flex-wrap items-center gap-1.5">
                    {crm.stages.map((s, i) => (
                      <span key={i} className="inline-flex items-center gap-1.5">
                        {i > 0 && <span className="text-slate-300">→</span>}
                        <span className="chip" style={{ background: "#f1f5f9", color: "#475569" }}>
                          {s.to || s.from}
                          <span className="text-slate-400 font-normal"> {date(s.at)}</span>
                        </span>
                      </span>
                    ))}
                  </div>
                </div>
              )}
              {crm.summary && <div className="mt-3 text-sm text-slate-600 border-t border-slate-50 pt-3">{crm.summary}</div>}
            </>
          )}
        </Card>
      )}

      <div className="font-bold mt-8 mb-3">Satışlar ({d!.deals.length})</div>
      {d!.deals.length === 0 ? (
        <Card className="p-5 text-slate-400 text-sm">Bu müşteriye bağlı satış yok.</Card>
      ) : (
        <div className="grid md:grid-cols-2 gap-3">
          {d!.deals.map((dl) => (
            <Link key={dl.id} to={`/satislar/${dl.id}`}>
              <Card className="p-4 hover:border-indigo-300 transition">
                <div className="flex justify-between items-start">
                  <div>
                    <div className="font-semibold tabular whitespace-nowrap">{unitLabel(dl.block, dl.unit_number)}</div>
                    <div className="text-xs text-slate-500">{dl.unit_type}</div>
                  </div>
                  <StatusChip status={dl.stage} />
                </div>
                <div className="mt-2 font-bold tabular">{money(dl.sale_price, dl.currency)}</div>
                <div className="text-xs text-slate-400 mt-1">{date(dl.contract_at)}</div>
              </Card>
            </Link>
          ))}
        </div>
      )}

      <div className="font-bold mt-8 mb-3">Görüşme geçmişi ({d!.activities.length})</div>
      {d!.activities.length === 0 ? (
        <Card className="p-5 text-slate-400 text-sm">Kayıtlı görüşme yok.</Card>
      ) : (
        <Card className="p-2">
          <div className="divide-y divide-slate-50">
            {d!.activities.map((a, i) => (
              <div key={i} className="flex gap-4 p-4">
                <div className="w-24 shrink-0 text-xs text-slate-400">{date(a.date)}</div>
                <div className="flex-1">
                  <div className="flex gap-2 items-center flex-wrap">
                    {a.channel && <span className="chip" style={{ background: "#f1f5f9", color: "#475569" }}>{a.channel}</span>}
                    {a.result && <span className="chip" style={{ background: "#eff6ff", color: "#1d4ed8" }}>{a.result}</span>}
                    {a.agent && <span className="text-xs text-slate-500">{a.agent}</span>}
                    {a.has_audio && <span className="text-xs text-emerald-600">🎧 kayıt</span>}
                  </div>
                  {a.notes && <div className="text-sm text-slate-700 mt-1">{a.notes}</div>}
                </div>
              </div>
            ))}
          </div>
        </Card>
      )}
    </>
  );
}
