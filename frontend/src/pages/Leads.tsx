import { useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { api, date } from "../api";
import { Card, PageHead, Loading, ErrorBox, SearchBox, useApi } from "../ui/kit";

export default function Leads() {
  const nav = useNavigate();
  const [params, setParams] = useSearchParams();
  const term = params.get("q") ?? "";
  const [q, setQ] = useState(term);
  const [page, setPage] = useState(0);
  const submit = () => {
    setPage(0);
    setParams(q ? { q } : {}, { replace: true });
  };
  const limit = 50;
  const { data, error, loading } = useApi(() => api.leads(term, limit, page * limit), [term, page]);

  return (
    <>
      <PageHead
        title="Müşteriler"
        subtitle={data ? `${data.total.toLocaleString("tr-TR")} kayıt` : "—"}
        right={
          <div className="flex gap-2">
            <SearchBox value={q} onChange={setQ} onSubmit={submit} placeholder="İsim veya telefon ara…" />
            <button onClick={submit} className="px-4 py-2 rounded-xl bg-indigo-500 text-white text-sm font-semibold hover:bg-indigo-600">
              Ara
            </button>
          </div>
        }
      />

      {error ? (
        <ErrorBox error={error} />
      ) : (
        <Card className="overflow-hidden">
          <div className="overflow-x-auto scrolly">
            <table className="w-full text-sm">
              <thead>
                <tr className="text-left text-slate-500 border-b border-slate-100">
                  <th className="px-5 py-3 font-semibold">İsim</th>
                  <th className="px-5 py-3 font-semibold">Telefon</th>
                  <th className="px-5 py-3 font-semibold">Kategori</th>
                  <th className="px-5 py-3 font-semibold">Danışman</th>
                  <th className="px-5 py-3 font-semibold">Ofis</th>
                  <th className="px-5 py-3 font-semibold">Şehir</th>
                  <th className="px-5 py-3 font-semibold">Kayıt</th>
                </tr>
              </thead>
              <tbody>
                {loading ? (
                  <tr><td colSpan={7}><Loading /></td></tr>
                ) : (
                  data!.rows.map((l) => (
                    <tr key={l.id} onClick={() => nav(`/musteriler/${l.id}`)} className="rowhover border-b border-slate-50 last:border-0 cursor-pointer">
                      <td className="px-5 py-3 font-medium text-indigo-700">{l.name || "—"}</td>
                      <td className="px-5 py-3 tabular text-slate-600">{l.phone || "—"}</td>
                      <td className="px-5 py-3 text-slate-600">{l.category || "—"}</td>
                      <td className="px-5 py-3 text-slate-600">{l.agent || "—"}</td>
                      <td className="px-5 py-3 text-slate-600">{l.office || "—"}</td>
                      <td className="px-5 py-3 text-slate-600">{l.city || "—"}</td>
                      <td className="px-5 py-3 text-slate-400">{date(l.created_at)}</td>
                    </tr>
                  ))
                )}
                {!loading && data!.rows.length === 0 && (
                  <tr><td colSpan={7} className="px-5 py-10 text-center text-slate-400">Kayıt bulunamadı</td></tr>
                )}
              </tbody>
            </table>
          </div>
          {data && data.total > limit && (
            <div className="flex items-center justify-between px-5 py-3 border-t border-slate-100 text-sm">
              <span className="text-slate-500">
                {page * limit + 1}–{Math.min((page + 1) * limit, data.total)} / {data.total}
              </span>
              <div className="flex gap-2">
                <button
                  disabled={page === 0}
                  onClick={() => setPage((p) => p - 1)}
                  className="px-3 py-1.5 rounded-lg border border-slate-200 disabled:opacity-40 hover:bg-slate-50"
                >
                  Önceki
                </button>
                <button
                  disabled={(page + 1) * limit >= data.total}
                  onClick={() => setPage((p) => p + 1)}
                  className="px-3 py-1.5 rounded-lg border border-slate-200 disabled:opacity-40 hover:bg-slate-50"
                >
                  Sonraki
                </button>
              </div>
            </div>
          )}
        </Card>
      )}
    </>
  );
}
