import React from "react";

export function Card({ children, className = "", style }: { children: React.ReactNode; className?: string; style?: React.CSSProperties }) {
  return <div className={`card ${className}`} style={style}>{children}</div>;
}

export function PageHead({ title, subtitle, right }: { title: string; subtitle?: string; right?: React.ReactNode }) {
  return (
    <div className="flex items-end justify-between gap-4 mb-6">
      <div>
        <h1 className="text-2xl font-extrabold tracking-tight">{title}</h1>
        {subtitle && <p className="text-sm text-slate-500 mt-1">{subtitle}</p>}
      </div>
      {right}
    </div>
  );
}

export function Loading({ label = "Yükleniyor…" }: { label?: string }) {
  return (
    <div className="flex items-center gap-3 text-slate-400 py-16 justify-center">
      <span className="inline-block w-4 h-4 rounded-full border-2 border-slate-300 border-t-indigo-500 animate-spin" />
      {label}
    </div>
  );
}

export function ErrorBox({ error }: { error: string }) {
  return (
    <Card className="p-5 border-rose-200 bg-rose-50 text-rose-700 text-sm">
      Bağlantı hatası: {error}
      <div className="text-rose-500 mt-1 text-xs">Backend çalışıyor mu? (Go API :8080)</div>
    </Card>
  );
}

const STATUS: Record<string, { bg: string; fg: string; label: string }> = {
  available: { bg: "#ecfdf5", fg: "#047857", label: "Boşta" },
  reserved: { bg: "#fffbeb", fg: "#b45309", label: "Rezerve" },
  sold: { bg: "#eef2ff", fg: "#4338ca", label: "Satıldı" },
  unavailable: { bg: "#f1f5f9", fg: "#475569", label: "Kapalı" },
  sale: { bg: "#ecfdf5", fg: "#047857", label: "Satış" },
  negotiation: { bg: "#eff6ff", fg: "#1d4ed8", label: "Görüşme" },
  cancelled: { bg: "#fff1f2", fg: "#be123c", label: "İptal" },
};

export function StatusChip({ status }: { status: string }) {
  const s = STATUS[status] ?? { bg: "#f1f5f9", fg: "#475569", label: status || "—" };
  return (
    <span className="chip" style={{ background: s.bg, color: s.fg }}>
      <span style={{ width: 6, height: 6, borderRadius: 999, background: s.fg }} />
      {s.label}
    </span>
  );
}

export function useApi<T>(fn: () => Promise<T>, deps: React.DependencyList = []) {
  const [data, setData] = React.useState<T | null>(null);
  const [error, setError] = React.useState<string | null>(null);
  const [loading, setLoading] = React.useState(true);
  React.useEffect(() => {
    let alive = true;
    setLoading(true);
    fn()
      .then((d) => alive && (setData(d), setError(null)))
      .catch((e) => alive && setError(String(e.message ?? e)))
      .finally(() => alive && setLoading(false));
    return () => {
      alive = false;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, deps);
  return { data, error, loading };
}

/** The one search box every list page shares. Enter submits when onSubmit is given. */
export function SearchBox({
  value,
  onChange,
  placeholder = "Ara…",
  onSubmit,
  autoFocus,
  className = "",
}: {
  value: string;
  onChange: (v: string) => void;
  placeholder?: string;
  onSubmit?: () => void;
  autoFocus?: boolean;
  className?: string;
}) {
  return (
    <div className={`relative ${className}`}>
      <svg className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400 pointer-events-none" width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
        <circle cx="11" cy="11" r="7" /><path d="M20 20l-3.5-3.5" />
      </svg>
      <input
        value={value}
        autoFocus={autoFocus}
        onChange={(e) => onChange(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === "Enter" && onSubmit) onSubmit();
          if (e.key === "Escape") onChange("");
        }}
        placeholder={placeholder}
        aria-label={placeholder}
        className="pl-9 pr-8 py-2 rounded-xl border border-slate-200 bg-white text-sm w-64 outline-none focus:border-indigo-400"
      />
      {value && (
        <button
          type="button"
          onClick={() => onChange("")}
          aria-label="Temizle"
          className="absolute right-2 top-1/2 -translate-y-1/2 w-5 h-5 grid place-items-center rounded-full text-slate-400 hover:bg-slate-100 hover:text-slate-600 text-xs"
        >
          ✕
        </button>
      )}
    </div>
  );
}
