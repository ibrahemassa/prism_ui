import { useEffect, useState } from "react";
import { NavLink, Outlet, useNavigate, useSearchParams } from "react-router-dom";
import { api } from "../api";
import type { SyncStatus } from "../types";
import { SearchBox } from "./kit";
import { useSettings } from "./settings";
import { BRAND, dot } from "../brand";

const nav = [
  { to: "/panel", label: "Panel", icon: "M3 12l9-9 9 9M5 10v10h14V10" },
  { to: "/musteriler", label: "Müşteriler", icon: "M16 14a4 4 0 10-8 0M12 7a3 3 0 100 6 3 3 0 000-6zM4 20a8 8 0 0116 0" },
  { to: "/daireler", label: "Daireler", icon: "M3 21h18M5 21V7l7-4 7 4v14M9 21v-6h6v6" },
  { to: "/satislar", label: "Satışlar", icon: "M3 3v18h18M7 14l3-3 3 3 5-6" },
  { to: "/raporlar", label: "Raporlar", icon: "M4 4v16h16M8 16V9m4 7V5m4 11v-6" },
  { to: "/komisyonlar", label: "Komisyonlar", icon: "M19 5L5 19M6.5 9a2.5 2.5 0 100-5 2.5 2.5 0 000 5zM17.5 20a2.5 2.5 0 100-5 2.5 2.5 0 000 5z" },
  { to: "/personel", label: "Personel", icon: "M17 20v-2a4 4 0 00-4-4H7a4 4 0 00-4 4v2M10 11a3 3 0 100-6 3 3 0 000 6zM21 20v-2a4 4 0 00-3-3.87" },
  { to: "/acentalar", label: "Acentalar", icon: "M3 21h18M4 21V8h16v13M9 12h2m3 0h2M9 16h2m3 0h2" },
];

function GlobalSearch() {
  const nav = useNavigate();
  const [params] = useSearchParams();
  const [q, setQ] = useState(params.get("q") ?? "");
  // keep the box in step with the URL when the user lands on /ara?q=… by link
  useEffect(() => setQ(params.get("q") ?? ""), [params]);
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "k") {
        e.preventDefault();
        (document.getElementById("global-search")?.querySelector("input") as HTMLInputElement | null)?.focus();
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);
  return (
    <div id="global-search" className="mr-auto flex items-center gap-2">
      <SearchBox value={q} onChange={setQ} onSubmit={() => q.trim() && nav(`/ara?q=${encodeURIComponent(q.trim())}`)} placeholder="Her yerde ara…  (Ctrl+K)" className="w-80 [&>input]:w-full" />
    </div>
  );
}

/** The CRM push: a status pill (when the last one landed) and a "Push to CRM"
 *  button that sends a full snapshot right now. Hidden entirely when the
 *  backend has no CRM configured. */
function SyncControls() {
  const [st, setSt] = useState<SyncStatus | null>(null);
  const [busy, setBusy] = useState(false);
  const [flash, setFlash] = useState<string | null>(null);
  const load = () => api.syncStatus().then(setSt).catch(() => setSt(null));
  useEffect(() => {
    load();
    const t = setInterval(load, 30_000);
    return () => clearInterval(t);
  }, []);
  if (!st?.enabled) return null;
  const ok = st.last_ok ? new Date(st.last_ok) : null;
  const stale = !ok || Date.now() - +ok > 45 * 60_000;
  const tone = st.last_error ? "bg-rose-500" : stale ? "bg-amber-500" : "bg-emerald-500";
  const running = busy || st.running;
  const label = running
    ? "Gönderiliyor…"
    : st.last_error
      ? "CRM hatası"
      : ok
        ? `Son gönderim ${ok.toLocaleTimeString("tr-TR", { hour: "2-digit", minute: "2-digit" })}`
        : "Henüz gönderilmedi";
  const summary = st.last_result ? (st.last_result as { summary?: Record<string, unknown> }).summary : undefined;
  const title = [
    st.dry_run ? "DENEME MODU — CRM yazmıyor" : "",
    st.last_error || "",
    summary ? Object.entries(summary).map(([k, v]) => `${k}: ${v}`).join("\n") : "",
  ].filter(Boolean).join("\n");
  const push = async () => {
    setBusy(true);
    setFlash(null);
    try {
      const { result } = await api.syncRun();
      const sm = (result as { summary?: Record<string, unknown> }).summary;
      setFlash(sm ? Object.entries(sm).filter(([, v]) => v).map(([k, v]) => `${k} ${v}`).join(" · ") || "Değişiklik yok" : "Gönderildi");
    } catch (e) {
      setFlash(`Hata: ${(e as Error).message}`);
    } finally {
      setBusy(false);
      load();
    }
  };
  return (
    <div className="flex items-center gap-2">
      <span title={title || undefined} className="flex items-center gap-2 px-2.5 py-1.5 rounded-lg text-xs text-slate-500 border border-slate-200">
        <span className={`w-2 h-2 rounded-full ${tone}`} />
        {label}
        {st.dry_run && <span className="text-[10px] font-bold text-amber-600">DENEME</span>}
      </span>
      {flash && <span className="text-xs text-slate-500 max-w-72 truncate" title={flash}>{flash}</span>}
      <button
        onClick={push}
        disabled={running}
        title="Prism'deki her şeyi (daireler, satışlar, iptaller, müşteriler, personel, acentalar) şimdi CRM'e gönder"
        className="px-3 py-1.5 rounded-lg text-sm font-semibold bg-indigo-600 text-white hover:bg-indigo-700 disabled:opacity-60"
      >
        {running ? "Gönderiliyor…" : "CRM'e Gönder"}
      </button>
    </div>
  );
}

function TopBar() {
  const { theme, toggleTheme, prismOnly, togglePrismOnly } = useSettings();
  return (
    <div className="sticky top-0 z-10 flex items-center justify-end gap-2 px-8 h-14 topbar">
      <GlobalSearch />
      <SyncControls />
      <button
        onClick={togglePrismOnly}
        title="Sadece Prism verisini göster (CRM verisini gizle)"
        className={`px-3 py-1.5 rounded-lg text-sm font-semibold border transition ${
          prismOnly
            ? "bg-amber-500 text-white border-amber-500"
            : "border-slate-300 text-slate-600 hover:bg-slate-50"
        }`}
      >
        {prismOnly ? "● Yalnızca Prism" : "Yalnızca Prism"}
      </button>
      <button
        onClick={toggleTheme}
        title="Koyu / açık tema"
        className="w-9 h-9 grid place-items-center rounded-lg border border-slate-300 text-slate-600 hover:bg-slate-50"
      >
        {theme === "dark" ? "☀" : "☾"}
      </button>
    </div>
  );
}

export function Layout() {
  return (
    <div className="flex h-full">
      <aside className="w-60 shrink-0 bg-[#111827] text-slate-300 flex flex-col">
        <div className="px-5 py-5 flex items-center gap-2.5 border-b border-white/10">
          <div className="w-9 h-9 rounded-xl bg-indigo-500 grid place-items-center font-extrabold text-white">P</div>
          <div>
            <div className="text-white font-bold leading-tight">Prism UI v2</div>
            <div className="text-[11px] text-slate-400">{dot(BRAND.project, "salt-okunur")}</div>
          </div>
        </div>
        <nav className="flex-1 px-3 py-4 space-y-1">
          {nav.map((n) => (
            <NavLink
              key={n.to}
              to={n.to}
              className={({ isActive }) =>
                `flex items-center gap-3 px-3 py-2.5 rounded-xl text-sm font-medium transition ${
                  isActive ? "bg-indigo-500 text-white shadow" : "hover:bg-white/5 text-slate-300"
                }`
              }
            >
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <path d={n.icon} />
              </svg>
              {n.label}
            </NavLink>
          ))}
        </nav>
        <div className="px-5 py-4 border-t border-white/10 text-[11px] text-slate-500">
          Prism verisi · yalnızca görüntüleme
        </div>
      </aside>

      <main className="flex-1 overflow-y-auto scrolly">
        <TopBar />
        <div className="max-w-[1200px] mx-auto px-8 py-8">
          <Outlet />
        </div>
      </main>
    </div>
  );
}
