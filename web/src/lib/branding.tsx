import { createContext, useContext, useEffect, type ReactNode } from "react";
import { useQuery } from "@tanstack/react-query";
import { api, unwrap } from "@/lib/api";
import type { Branding } from "@/lib/types";

// The defaults mirror the backend's branding defaults so the UI renders
// sensibly before the (public) branding request resolves and if it fails.
export const DEFAULT_BRANDING: Branding = {
  app_name: "SmartTicket",
  app_subtitle: "console",
  workspace_name: "LINBIT workspace",
  primary_color: "#f59e0b",
  login_tagline: "Every ticket, SLA and customer — under one calm, fast surface.",
  login_subtext: "Self-hosted. Single-tenant. Your data, your rules.",
  has_logo: false,
  logo_url: "",
  updated_at: 0,
};

const BrandingContext = createContext<Branding>(DEFAULT_BRANDING);

async function fetchBranding(): Promise<Branding> {
  const res = await api.get("/settings/branding");
  return unwrap<Branding>(res.data);
}

/** Relative luminance per WCAG; used to pick a readable foreground. */
function readableForeground(hex: string): string {
  const m = /^#?([0-9a-f]{3}|[0-9a-f]{6})$/i.exec(hex.trim());
  if (!m) return "#19130a";
  let h = m[1];
  if (h.length === 3) h = h.split("").map((c) => c + c).join("");
  const r = parseInt(h.slice(0, 2), 16) / 255;
  const g = parseInt(h.slice(2, 4), 16) / 255;
  const b = parseInt(h.slice(4, 6), 16) / 255;
  const lin = (c: number) =>
    c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4;
  const L = 0.2126 * lin(r) + 0.7152 * lin(g) + 0.0722 * lin(b);
  // Dark ink on bright accents, near-white on dark accents.
  return L > 0.5 ? "#19130a" : "#ffffff";
}

/** Apply the accent color as runtime CSS variables (overrides the :root + .dark
 * palette values for --primary / --ring, and a matching readable foreground).
 * Inline styles on <html> win over both light and dark stylesheet rules. */
function applyAccent(color: string) {
  const root = document.documentElement;
  const c = color?.trim();
  if (c) {
    root.style.setProperty("--primary", c);
    root.style.setProperty("--ring", c);
    root.style.setProperty("--primary-fg", readableForeground(c));
  } else {
    root.style.removeProperty("--primary");
    root.style.removeProperty("--ring");
    root.style.removeProperty("--primary-fg");
  }
}

export function BrandingProvider({ children }: { children: ReactNode }) {
  const { data } = useQuery({
    queryKey: ["branding"],
    queryFn: fetchBranding,
    staleTime: 60_000,
    // Branding rarely changes; keep the last value through refetches.
    placeholderData: (prev) => prev,
  });

  const branding = data ?? DEFAULT_BRANDING;

  useEffect(() => {
    applyAccent(branding.primary_color);
  }, [branding.primary_color]);

  useEffect(() => {
    const name = branding.app_name || "SmartTicket";
    document.title = `${name} · ${branding.app_subtitle || "console"}`;
  }, [branding.app_name, branding.app_subtitle]);

  return (
    <BrandingContext.Provider value={branding}>
      {children}
    </BrandingContext.Provider>
  );
}

export function useBranding(): Branding {
  return useContext(BrandingContext);
}
