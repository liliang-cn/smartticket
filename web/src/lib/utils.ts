import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";
import i18n from "@/lib/i18n";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

/** The active i18next locale, used to drive all `Intl` formatting. */
function currentLocale(): string {
  return i18n.resolvedLanguage || i18n.language || "en";
}

/**
 * Coerce the timestamp shapes the API returns — unix seconds (number) or
 * RFC3339 / ISO string — into a Date. Returns null for empty / unparseable.
 * Timezone is intentionally left to the browser (renders in the user's zone).
 */
function toDate(value?: number | string | null): Date | null {
  if (value == null || value === "" || value === 0) return null;
  const d = typeof value === "number" ? new Date(value * 1000) : new Date(value);
  return Number.isNaN(d.getTime()) ? null : d;
}

/** Localized date + time, e.g. "Jun 2, 2026, 3:04 PM" / "2026年6月2日 15:04". */
export function formatDateTime(value?: number | string | null): string {
  const d = toDate(value);
  if (!d) return "";
  return new Intl.DateTimeFormat(currentLocale(), {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(d);
}

/** Localized date only, e.g. "Jun 2, 2026" / "2026年6月2日". */
export function formatDate(value?: number | string | null): string {
  const d = toDate(value);
  if (!d) return "";
  return new Intl.DateTimeFormat(currentLocale(), { dateStyle: "medium" }).format(d);
}

/** Compact month/day + time, used in dense lists (e.g. notifications). */
export function formatShortDateTime(value?: number | string | null): string {
  const d = toDate(value);
  if (!d) return "";
  return new Intl.DateTimeFormat(currentLocale(), {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  }).format(d);
}

/** Localized relative time ("2 hours ago" / "2小时前" / "vor 2 Stunden"). */
export function relativeTime(value?: number | string | null): string {
  const d = toDate(value);
  if (!d) return "—";
  const diff = Date.now() - d.getTime();
  const abs = Math.abs(diff);
  const units: [number, Intl.RelativeTimeFormatUnit][] = [
    [60_000, "minute"],
    [3_600_000, "hour"],
    [86_400_000, "day"],
    [604_800_000, "week"],
    [2_592_000_000, "month"],
  ];
  const rtf = new Intl.RelativeTimeFormat(currentLocale(), { numeric: "auto" });
  if (abs < 60_000) return rtf.format(0, "second");
  for (let i = units.length - 1; i >= 0; i--) {
    const [ms, unit] = units[i];
    if (abs >= ms) return rtf.format(Math.round(-diff / ms), unit);
  }
  return rtf.format(0, "second");
}
