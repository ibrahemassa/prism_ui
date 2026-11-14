import { createContext, useContext, useEffect, useState } from "react";

type Settings = {
  theme: "light" | "dark";
  toggleTheme: () => void;
  prismOnly: boolean;
  togglePrismOnly: () => void;
};

const Ctx = createContext<Settings>(null!);
export const useSettings = () => useContext(Ctx);

function read(key: string, fallback: string) {
  try {
    return localStorage.getItem(key) ?? fallback;
  } catch {
    return fallback;
  }
}

export function SettingsProvider({ children }: { children: React.ReactNode }) {
  const [theme, setTheme] = useState<"light" | "dark">(() => (read("theme", "light") as "light" | "dark"));
  const [prismOnly, setPrismOnly] = useState(() => read("prismOnly", "0") === "1");

  useEffect(() => {
    document.documentElement.dataset.theme = theme;
    try {
      localStorage.setItem("theme", theme);
    } catch {}
  }, [theme]);
  useEffect(() => {
    try {
      localStorage.setItem("prismOnly", prismOnly ? "1" : "0");
    } catch {}
  }, [prismOnly]);

  return (
    <Ctx.Provider
      value={{
        theme,
        toggleTheme: () => setTheme((t) => (t === "light" ? "dark" : "light")),
        prismOnly,
        togglePrismOnly: () => setPrismOnly((p) => !p),
      }}
    >
      {children}
    </Ctx.Provider>
  );
}
