/**
 * Local branding, read at build time from frontend/.env.local (gitignored;
 * see .env.example). Image paths are URLs under public/, normally
 * public/brand/ (also gitignored). Anything unset falls back to a generic
 * name or no image.
 */
const env = import.meta.env;

export const BRAND = {
  /** project name, e.g. on the dashboard and report cover */
  project: (env.VITE_BRAND_PROJECT ?? "").trim(),
  /** company name, in the printed report's footer */
  company: (env.VITE_BRAND_COMPANY ?? "").trim(),
  /** project logo for the printed report and commission sheet */
  logo: (env.VITE_BRAND_LOGO ?? "").trim(),
  /** company mark for the printed report's cover and footer */
  mark: (env.VITE_BRAND_MARK ?? "").trim(),
  /** browser tab icon; the generic /favicon.svg when unset */
  favicon: (env.VITE_BRAND_FAVICON ?? "").trim(),
};

/** Joins the non-empty parts with " · ". */
export const dot = (...parts: (string | undefined)[]) => parts.filter(Boolean).join(" · ");

/** Swaps in the local favicon, if one is configured. */
export function applyFavicon() {
  if (!BRAND.favicon) return;
  const link = document.querySelector<HTMLLinkElement>('link[rel="icon"]');
  if (link) {
    link.removeAttribute("type");
    link.href = BRAND.favicon;
  }
}
