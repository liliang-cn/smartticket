import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { Ticket, ArrowRight, ArrowLeft } from "lucide-react";
import { useAuth } from "@/lib/auth";
import { useBranding } from "@/lib/branding";
import { apiError } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { LanguageToggle } from "@/components/language-toggle";

const LIVE_DEMO_HOST = "smartticket.superleo.app";
const DEMO_EMAIL = "visitor@smartticket.local";
const DEMO_PASSWORD = "DemoPass123!";

export function LoginPage() {
  const { t } = useTranslation("auth");
  const { login } = useAuth();
  const branding = useBranding();
  const navigate = useNavigate();
  const isLiveDemo = window.location.hostname === LIVE_DEMO_HOST;
  const [email, setEmail] = useState(isLiveDemo ? DEMO_EMAIL : "");
  const [password, setPassword] = useState(isLiveDemo ? DEMO_PASSWORD : "");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setBusy(true);
    try {
      await login(email.trim(), password);
      navigate("/dashboard");
    } catch (err) {
      setError(apiError(err, t("invalid_credentials")));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="grid min-h-screen lg:grid-cols-2">
      {/* Language switcher, available before sign-in. */}
      <div className="absolute right-4 top-4 z-10">
        <LanguageToggle />
      </div>
      {/* Form side */}
      <div className="flex items-center justify-center px-6 py-12">
        <div className="w-full max-w-sm">
          {import.meta.env.BASE_URL !== "/" && (
            <a
              href="/"
              className="mb-6 inline-flex items-center gap-1.5 text-sm text-muted-foreground transition-colors hover:text-foreground"
            >
              <ArrowLeft className="size-4" /> {t("back_to_home")}
            </a>
          )}
          <div className="mb-10 flex items-center gap-3">
            <div className="grid size-10 place-items-center overflow-hidden rounded-lg bg-primary text-primary-foreground shadow-[0_0_30px_-6px_color-mix(in_srgb,var(--primary)_75%,transparent)]">
              {branding.has_logo ? (
                <img
                  src={branding.logo_url}
                  alt={branding.app_name}
                  className="size-full object-contain"
                />
              ) : (
                <Ticket className="size-5" strokeWidth={2.5} />
              )}
            </div>
            <div>
              <div className="font-display text-xl font-bold tracking-tight">
                {branding.app_name}
              </div>
              <div className="font-mono text-[10px] uppercase tracking-[0.25em] text-muted-foreground">
                {branding.app_subtitle}
              </div>
            </div>
          </div>

          <h1 className="text-2xl font-bold">{t("title")}</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            {t("subtitle")}
          </p>
          {isLiveDemo && (
            <p className="mt-3 rounded-md border border-primary/25 bg-primary/10 px-3 py-2 text-sm text-muted-foreground">
              Live demo credentials are pre-filled with a low-permission customer account.
            </p>
          )}

          <form onSubmit={onSubmit} className="mt-8 space-y-5">
            <div className="space-y-1.5">
              <Label htmlFor="email">{t("email")}</Label>
              <Input
                id="email"
                type="email"
                autoComplete="username"
                placeholder={t("email_placeholder")}
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                required
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="password">{t("password")}</Label>
              <Input
                id="password"
                type="password"
                autoComplete="current-password"
                placeholder="••••••••"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
              />
            </div>
            {error && (
              <p className="rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive-foreground">
                {error}
              </p>
            )}
            <Button type="submit" size="lg" className="w-full" disabled={busy}>
              {busy ? t("signing_in") : t("sign_in")}
              {!busy && <ArrowRight />}
            </Button>
          </form>
        </div>
      </div>

      {/* Aesthetic side */}
      <div className="relative hidden overflow-hidden border-l border-border lg:block">
        <div className="grid-texture absolute inset-0 opacity-60" />
        <div className="absolute inset-0 bg-[radial-gradient(40rem_30rem_at_70%_30%,rgba(255,176,31,0.16),transparent_70%)]" />
        <div className="relative flex h-full flex-col justify-end p-12">
          <div className="font-mono text-xs uppercase tracking-[0.3em] text-primary/80">
            {t("mission_control")}
          </div>
          <p className="mt-4 max-w-md font-display text-3xl font-bold leading-tight tracking-tight">
            {branding.login_tagline}
          </p>
          <p className="mt-4 max-w-sm text-sm text-muted-foreground">
            {branding.login_subtext}
          </p>
        </div>
      </div>
    </div>
  );
}
