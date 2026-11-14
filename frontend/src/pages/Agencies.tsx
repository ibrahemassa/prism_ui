import { useMemo, useState } from "react";
import { api, hit } from "../api";
import { Card, PageHead, Loading, ErrorBox, SearchBox, useApi } from "../ui/kit";

export default function Agencies() {
  const { data, error, loading } = useApi(() => api.agencies(), []);
  const [q, setQ] = useState("");
  const rows = useMemo(() => (data ? data.rows.filter((a) => hit(q, a.name, a.phone, a.email, a.city)) : []), [data, q]);
  if (loading) return <Loading />;
  if (error) return <ErrorBox error={error} />;
  return (
    <>
      <PageHead
        title="Acentalar"
        subtitle={q ? `${rows.length} / ${data!.total} acenta` : `${data!.total} acenta`}
        right={<SearchBox value={q} onChange={setQ} placeholder="Acenta, telefon, şehir…" />}
      />
      <Card className="overflow-hidden">
        <div className="overflow-x-auto scrolly">
          <table className="w-full text-sm">
            <thead>
              <tr className="text-left text-slate-500 border-b border-slate-100">
                <th className="px-5 py-3 font-semibold">Acenta</th>
                <th className="px-5 py-3 font-semibold">Telefon</th>
                <th className="px-5 py-3 font-semibold">E-posta</th>
                <th className="px-5 py-3 font-semibold">Şehir</th>
                <th className="px-5 py-3 font-semibold text-right">Komisyon %</th>
              </tr>
            </thead>
            <tbody>
              {rows.length === 0 && (
                <tr><td colSpan={5} className="px-5 py-10 text-center text-slate-400">Kayıt bulunamadı</td></tr>
              )}
              {rows.map((a) => (
                <tr key={a.id} className="rowhover border-b border-slate-50 last:border-0">
                  <td className="px-5 py-3 font-medium">{a.name}</td>
                  <td className="px-5 py-3 tabular text-slate-600">{a.phone || "—"}</td>
                  <td className="px-5 py-3 text-slate-600">{a.email || "—"}</td>
                  <td className="px-5 py-3 text-slate-600">{a.city || "—"}</td>
                  <td className="px-5 py-3 text-right tabular">{a.rate ? `%${a.rate}` : "—"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Card>
    </>
  );
}
