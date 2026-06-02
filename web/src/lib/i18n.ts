import i18n, { type Resource } from "i18next";
import { initReactI18next } from "react-i18next";
import LanguageDetector from "i18next-browser-languagedetector";

/** Languages this console ships, with their native display names. */
export const LANGUAGES = [
  { code: "en", label: "English" },
  { code: "zh", label: "中文" },
  { code: "es", label: "Español" },
  { code: "de", label: "Deutsch" },
  { code: "ja", label: "日本語" },
  { code: "ko", label: "한국어" },
  { code: "it", label: "Italiano" },
] as const;

export type LanguageCode = (typeof LANGUAGES)[number]["code"];

export const SUPPORTED_LANGS = LANGUAGES.map((l) => l.code);

/** localStorage key — sits alongside `st.theme`, `st.sidebar`. */
const STORAGE_KEY = "st.lang";

// Auto-load every `src/locales/<lang>/<namespace>.json` so adding a namespace
// is just dropping files — no edits here. Eager + bundled = synchronous init.
const modules = import.meta.glob<{ default: Record<string, unknown> }>(
  "../locales/*/*.json",
  { eager: true }
);

const resources: Resource = {};
const namespaces = new Set<string>();
for (const [path, mod] of Object.entries(modules)) {
  const match = /\/locales\/([^/]+)\/([^/]+)\.json$/.exec(path);
  if (!match) continue;
  const [, lang, ns] = match;
  namespaces.add(ns);
  (resources[lang] ??= {})[ns] = mod.default;
}

i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources,
    fallbackLng: "en",
    supportedLngs: SUPPORTED_LANGS,
    // Map regional tags (zh-CN, en-US, de-AT…) down to the base language.
    load: "languageOnly",
    ns: [...namespaces],
    defaultNS: "common",
    fallbackNS: "common",
    detection: {
      order: ["localStorage", "navigator"],
      lookupLocalStorage: STORAGE_KEY,
      caches: ["localStorage"],
    },
    interpolation: { escapeValue: false },
    returnNull: false,
    // Resources are bundled synchronously, so there is no async load to wait
    // on — skip Suspense to avoid needing a boundary around the whole app.
    react: { useSuspense: false },
  });

/** Keep `<html lang>` in sync so the browser localizes correctly. */
function syncHtmlLang(lng: string) {
  document.documentElement.lang = lng.split("-")[0];
}
syncHtmlLang(i18n.resolvedLanguage ?? "en");
i18n.on("languageChanged", syncHtmlLang);

/** Change language and persist the choice (detector also caches it). */
export function setLanguage(lng: LanguageCode) {
  i18n.changeLanguage(lng);
}

export default i18n;
