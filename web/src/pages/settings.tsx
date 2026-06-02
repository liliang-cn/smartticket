import { useEffect, useRef, useState, type CSSProperties } from "react";
import { useTranslation, Trans } from "react-i18next";
import { Ticket, Upload, Trash2, RotateCcw, Save } from "lucide-react";
import { toast } from "sonner";
import { useBranding, DEFAULT_BRANDING } from "@/lib/branding";
import {
  useUpdateBranding,
  useUploadLogo,
  useDeleteLogo,
} from "@/features/settings/api";
import { apiError } from "@/lib/api";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input, Textarea } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/misc";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { LanguageToggle } from "@/components/language-toggle";
import { useReveal } from "@/lib/use-reveal";

// A small palette of tasteful accent presets to pick from.
const PRESETS = [
  "#f59e0b", // amber (default)
  "#3b82f6", // blue
  "#10b981", // emerald
  "#8b5cf6", // violet
  "#ec4899", // pink
  "#ef4444", // red
  "#14b8a6", // teal
  "#f97316", // orange
];

const HEX_RE = /^#([0-9a-f]{3}|[0-9a-f]{6})$/i;

// readableForeground mirrors lib/branding so the preview button text stays legible.
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
  return L > 0.5 ? "#19130a" : "#ffffff";
}

function Field({
  label,
  hint,
  children,
}: {
  label: string;
  hint?: string;
  children: React.ReactNode;
}) {
  return (
    <div className="space-y-1.5">
      <Label>{label}</Label>
      {children}
      {hint && <p className="text-xs text-muted-foreground">{hint}</p>}
    </div>
  );
}

export function SettingsPage() {
  const { t } = useTranslation("settings");
  const branding = useBranding();
  const update = useUpdateBranding();
  const uploadLogo = useUploadLogo();
  const deleteLogo = useDeleteLogo();
  const ref = useReveal<HTMLDivElement>();
  const fileInput = useRef<HTMLInputElement>(null);
  const [confirmRemoveLogo, setConfirmRemoveLogo] = useState(false);

  // Local editable copy of the text + color fields, seeded from the store.
  const [form, setForm] = useState({
    app_name: branding.app_name,
    app_subtitle: branding.app_subtitle,
    workspace_name: branding.workspace_name,
    primary_color: branding.primary_color,
    login_tagline: branding.login_tagline,
    login_subtext: branding.login_subtext,
  });

  // Re-seed when the store changes (e.g. after a save resolves).
  useEffect(() => {
    setForm({
      app_name: branding.app_name,
      app_subtitle: branding.app_subtitle,
      workspace_name: branding.workspace_name,
      primary_color: branding.primary_color,
      login_tagline: branding.login_tagline,
      login_subtext: branding.login_subtext,
    });
  }, [branding]);

  const set = (k: keyof typeof form) => (v: string) =>
    setForm((f) => ({ ...f, [k]: v }));

  const validColor = HEX_RE.test(form.primary_color.trim());
  const previewColor = validColor ? form.primary_color.trim() : "#f59e0b";

  async function onSave() {
    if (!validColor) {
      toast.error(t("toasts.invalid_hex"));
      return;
    }
    try {
      await update.mutateAsync(form);
      toast.success(t("toasts.saved"));
    } catch (err) {
      toast.error(apiError(err, t("toasts.save_error")));
    }
  }

  function onResetDefaults() {
    setForm({
      app_name: DEFAULT_BRANDING.app_name,
      app_subtitle: DEFAULT_BRANDING.app_subtitle,
      workspace_name: DEFAULT_BRANDING.workspace_name,
      primary_color: DEFAULT_BRANDING.primary_color,
      login_tagline: DEFAULT_BRANDING.login_tagline,
      login_subtext: DEFAULT_BRANDING.login_subtext,
    });
  }

  async function onPickLogo(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    e.target.value = ""; // allow re-selecting the same file
    if (!file) return;
    try {
      await uploadLogo.mutateAsync(file);
      toast.success(t("toasts.logo_updated"));
    } catch (err) {
      toast.error(apiError(err, t("toasts.logo_upload_error")));
    }
  }

  async function onRemoveLogo() {
    try {
      await deleteLogo.mutateAsync();
      setConfirmRemoveLogo(false);
      toast.success(t("toasts.logo_removed"));
    } catch (err) {
      toast.error(apiError(err, t("toasts.logo_remove_error")));
    }
  }

  // Scope the chosen accent to the preview pane via CSS-variable overrides.
  const previewVars = {
    "--primary": previewColor,
    "--ring": previewColor,
    "--primary-fg": readableForeground(previewColor),
  } as CSSProperties;

  return (
    <div ref={ref} className="w-full">
      <div data-reveal className="mb-6">
        <h1 className="text-2xl">{t("title")}</h1>
        <p className="mt-1 text-sm text-muted-foreground">{t("subtitle")}</p>
      </div>

      <div className="grid gap-6 lg:grid-cols-[1fr_22rem]">
        {/* Controls */}
        <div className="space-y-5">
          {/* Language */}
          <Card data-reveal className="flex items-center justify-between gap-4 p-5">
            <div>
              <h2 className="text-sm font-semibold">{t("language.heading")}</h2>
              <p className="text-xs text-muted-foreground">{t("language.desc")}</p>
            </div>
            <LanguageToggle />
          </Card>

          {/* Identity */}
          <Card data-reveal className="space-y-4 p-5">
            <div>
              <h2 className="text-sm font-semibold">{t("identity.heading")}</h2>
              <p className="text-xs text-muted-foreground">
                {t("identity.desc")}
              </p>
            </div>
            <div className="grid gap-4 sm:grid-cols-2">
              <Field label={t("identity.app_name")} hint={t("identity.app_name_hint")}>
                <Input
                  value={form.app_name}
                  maxLength={100}
                  onChange={(e) => set("app_name")(e.target.value)}
                  placeholder="SmartTicket"
                />
              </Field>
              <Field label={t("identity.app_subtitle")} hint={t("identity.app_subtitle_hint")}>
                <Input
                  value={form.app_subtitle}
                  maxLength={100}
                  onChange={(e) => set("app_subtitle")(e.target.value)}
                  placeholder="console"
                />
              </Field>
            </div>
            <Field
              label={t("identity.workspace_name")}
              hint={t("identity.workspace_name_hint")}
            >
              <Input
                value={form.workspace_name}
                maxLength={120}
                onChange={(e) => set("workspace_name")(e.target.value)}
                placeholder="LINBIT workspace"
              />
            </Field>
          </Card>

          {/* Accent color */}
          <Card data-reveal className="space-y-4 p-5">
            <div>
              <h2 className="text-sm font-semibold">{t("accent.heading")}</h2>
              <p className="text-xs text-muted-foreground">
                {t("accent.desc")}
              </p>
            </div>
            <div className="flex flex-wrap items-center gap-2">
              {PRESETS.map((c) => (
                <button
                  key={c}
                  type="button"
                  onClick={() => set("primary_color")(c)}
                  aria-label={t("accent.use_color", { color: c })}
                  className="size-8 rounded-full border-2 transition-transform hover:scale-110"
                  style={{
                    backgroundColor: c,
                    borderColor:
                      form.primary_color.toLowerCase() === c.toLowerCase()
                        ? "var(--foreground)"
                        : "transparent",
                  }}
                />
              ))}
            </div>
            <div className="flex items-center gap-3">
              <input
                type="color"
                value={validColor ? form.primary_color : "#f59e0b"}
                onChange={(e) => set("primary_color")(e.target.value)}
                className="size-10 cursor-pointer rounded-md border border-border bg-transparent p-1"
                aria-label={t("accent.custom_color")}
              />
              <div className="w-40">
                <Input
                  value={form.primary_color}
                  onChange={(e) => set("primary_color")(e.target.value)}
                  placeholder="#f59e0b"
                  className="font-mono"
                  aria-invalid={!validColor}
                />
              </div>
              {!validColor && (
                <span className="text-xs text-destructive">
                  {t("accent.invalid_hex")}
                </span>
              )}
            </div>
          </Card>

          {/* Logo */}
          <Card data-reveal className="space-y-4 p-5">
            <div>
              <h2 className="text-sm font-semibold">{t("logo.heading")}</h2>
              <p className="text-xs text-muted-foreground">
                {t("logo.desc")}
              </p>
            </div>
            <div className="flex items-center gap-4">
              <div className="grid size-14 place-items-center overflow-hidden rounded-lg border border-border bg-card">
                {branding.has_logo ? (
                  <img
                    src={branding.logo_url}
                    alt={t("logo.current_alt")}
                    className="size-full object-contain"
                  />
                ) : (
                  <div className="grid size-9 place-items-center rounded-md bg-primary text-primary-foreground">
                    <Ticket className="size-5" strokeWidth={2.5} />
                  </div>
                )}
              </div>
              <div className="flex gap-2">
                <input
                  ref={fileInput}
                  type="file"
                  accept="image/png,image/jpeg,image/svg+xml,image/webp,image/gif"
                  className="hidden"
                  onChange={onPickLogo}
                />
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={() => fileInput.current?.click()}
                  disabled={uploadLogo.isPending}
                >
                  <Upload /> {uploadLogo.isPending ? t("logo.uploading") : t("logo.upload")}
                </Button>
                {branding.has_logo && (
                  <Button
                    variant="ghost"
                    size="sm"
                    className="text-destructive hover:text-destructive"
                    onClick={() => setConfirmRemoveLogo(true)}
                    disabled={deleteLogo.isPending}
                  >
                    <Trash2 /> {t("logo.remove")}
                  </Button>
                )}
              </div>
            </div>
          </Card>

          {/* Sign-in page */}
          <Card data-reveal className="space-y-4 p-5">
            <div>
              <h2 className="text-sm font-semibold">{t("signin.heading")}</h2>
              <p className="text-xs text-muted-foreground">
                {t("signin.desc")}
              </p>
            </div>
            <Field label={t("signin.tagline")}>
              <Textarea
                value={form.login_tagline}
                maxLength={200}
                rows={2}
                onChange={(e) => set("login_tagline")(e.target.value)}
                placeholder="Every ticket, SLA and customer — under one calm, fast surface."
              />
            </Field>
            <Field label={t("signin.subtext")}>
              <Textarea
                value={form.login_subtext}
                maxLength={300}
                rows={2}
                onChange={(e) => set("login_subtext")(e.target.value)}
                placeholder="Self-hosted. Single-tenant. Your data, your rules."
              />
            </Field>
          </Card>

          <div data-reveal className="flex items-center gap-2">
            <Button onClick={onSave} disabled={update.isPending || !validColor}>
              <Save /> {update.isPending ? t("saving") : t("save_changes")}
            </Button>
            <Button variant="ghost" onClick={onResetDefaults} type="button">
              <RotateCcw /> {t("reset_defaults")}
            </Button>
          </div>
        </div>

        {/* Live preview */}
        <aside data-reveal className="space-y-4">
          <div className="sticky top-20 space-y-3">
            <div className="font-mono text-[11px] uppercase tracking-wider text-muted-foreground">
              {t("preview.label")}
            </div>
            <Card className="overflow-hidden p-0" style={previewVars}>
              {/* Brand block */}
              <div className="flex items-center gap-2.5 border-b border-border px-4 py-3.5">
                <div className="grid size-8 place-items-center overflow-hidden rounded-md bg-primary text-primary-foreground">
                  {branding.has_logo ? (
                    <img
                      src={branding.logo_url}
                      alt=""
                      className="size-full object-contain"
                    />
                  ) : (
                    <Ticket className="size-4.5" strokeWidth={2.5} />
                  )}
                </div>
                <div className="leading-none">
                  <div className="font-display text-[15px] font-bold tracking-tight">
                    {form.app_name || "SmartTicket"}
                  </div>
                  <div className="font-mono text-[10px] uppercase tracking-[0.2em] text-muted-foreground">
                    {form.app_subtitle || "console"}
                  </div>
                </div>
              </div>

              {/* Workspace label */}
              <div className="border-b border-border px-4 py-2.5 font-mono text-[11px] uppercase tracking-[0.25em] text-muted-foreground">
                {form.workspace_name || "workspace"}
              </div>

              {/* Sample nav + controls */}
              <div className="space-y-2 p-4">
                <div className="relative flex items-center gap-2.5 rounded-md bg-primary/10 px-3 py-2 text-sm font-medium">
                  <span className="absolute left-0 top-1/2 h-5 w-0.5 -translate-y-1/2 rounded-full bg-primary" />
                  <Ticket className="size-4" /> {t("preview.active_item")}
                </div>
                <div className="flex items-center gap-2.5 rounded-md px-3 py-2 text-sm text-muted-foreground">
                  <Ticket className="size-4" /> {t("preview.inactive_item")}
                </div>
                <Separator />
                <div className="flex flex-wrap items-center gap-2 pt-1">
                  <Button size="sm">{t("preview.primary")}</Button>
                  <Badge tone="amber">{t("preview.accent")}</Badge>
                  <a className="text-sm text-primary underline" href="#">
                    {t("preview.link")}
                  </a>
                </div>
              </div>
            </Card>
            <p className="text-xs text-muted-foreground">
              <Trans ns="settings" i18nKey="preview.note" components={[<strong key="0" />]} />
            </p>
          </div>
        </aside>
      </div>

      <ConfirmDialog
        open={confirmRemoveLogo}
        onOpenChange={setConfirmRemoveLogo}
        title={t("confirm_remove_logo.title")}
        description={t("confirm_remove_logo.description")}
        confirmLabel={t("confirm_remove_logo.confirm")}
        pending={deleteLogo.isPending}
        onConfirm={onRemoveLogo}
      />
    </div>
  );
}
