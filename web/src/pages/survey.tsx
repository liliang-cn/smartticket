import { useState } from "react";
import { useParams } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { Ticket, Star, CheckCircle2, AlertTriangle } from "lucide-react";
import { useSurveyPublic, useSubmitSurvey } from "@/features/survey/api";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/input";
import { useBranding } from "@/lib/branding";
import { LanguageToggle } from "@/components/language-toggle";
import { cn } from "@/lib/utils";

/** A simple loading skeleton. */
function SurveyLoader() {
  return (
    <div className="flex flex-col items-center gap-4 py-12">
      <div className="size-10 animate-pulse rounded-full bg-muted" />
      <div className="h-4 w-48 animate-pulse rounded bg-muted" />
      <div className="h-3 w-32 animate-pulse rounded bg-muted" />
    </div>
  );
}

/** Star rating picker (1–5). */
function StarPicker({
  value,
  onChange,
}: {
  value: number;
  onChange: (n: number) => void;
}) {
  const [hovered, setHovered] = useState(0);
  return (
    <div className="flex gap-1" role="group" aria-label="Star rating">
      {[1, 2, 3, 4, 5].map((n) => {
        const filled = n <= (hovered || value);
        return (
          <button
            key={n}
            type="button"
            aria-label={`${n} star`}
            onClick={() => onChange(n)}
            onMouseEnter={() => setHovered(n)}
            onMouseLeave={() => setHovered(0)}
            className={cn(
              "size-10 rounded-md transition-colors sm:size-12",
              filled ? "text-amber-400" : "text-muted-foreground/30 hover:text-amber-300"
            )}
          >
            <Star
              className="size-full"
              fill={filled ? "currentColor" : "none"}
              strokeWidth={1.5}
            />
          </button>
        );
      })}
    </div>
  );
}

export function SurveyPage() {
  const { token } = useParams<{ token: string }>();
  const { t } = useTranslation("survey");
  const branding = useBranding();

  const { data, isLoading, isError } = useSurveyPublic(token ?? "");
  const submit = useSubmitSurvey(token ?? "");

  const [rating, setRating] = useState(0);
  const [comment, setComment] = useState("");
  const [submitted, setSubmitted] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (rating < 1) return;
    setSubmitError(null);
    try {
      await submit.mutateAsync({ rating, comment });
      setSubmitted(true);
    } catch {
      setSubmitError(t("error_generic"));
    }
  }

  return (
    <div className="relative min-h-screen bg-background text-foreground">
      {/* Language toggle — accessible on the public page */}
      <div className="absolute right-4 top-4 z-10">
        <LanguageToggle />
      </div>

      <div className="mx-auto flex min-h-screen max-w-lg flex-col items-center justify-center px-6 py-16">
        {/* Brand header */}
        <div className="mb-10 flex flex-col items-center gap-3 text-center">
          <div className="grid size-12 place-items-center overflow-hidden rounded-xl bg-primary text-primary-foreground shadow-[0_0_30px_-6px_color-mix(in_srgb,var(--primary)_75%,transparent)]">
            {branding.has_logo ? (
              <img
                src={branding.logo_url}
                alt={branding.app_name}
                className="size-full object-contain"
              />
            ) : (
              <Ticket className="size-6" strokeWidth={2.5} />
            )}
          </div>
          <div className="leading-none">
            <div className="font-display text-xl font-bold tracking-tight">
              {branding.app_name}
            </div>
            {branding.app_subtitle && (
              <div className="mt-0.5 font-mono text-xs uppercase tracking-[0.2em] text-muted-foreground">
                {branding.app_subtitle}
              </div>
            )}
          </div>
        </div>

        {/* Main card */}
        <div className="w-full rounded-2xl border border-border bg-card p-8 shadow-sm">
          {isLoading && <SurveyLoader />}

          {isError && (
            <div className="flex flex-col items-center gap-3 py-8 text-center">
              <AlertTriangle className="size-10 text-destructive" />
              <h1 className="text-lg font-semibold">{t("not_found_title")}</h1>
              <p className="text-sm text-muted-foreground">{t("not_found_desc")}</p>
            </div>
          )}

          {!isLoading && !isError && data?.responded && !submitted && (
            <div className="flex flex-col items-center gap-3 py-8 text-center">
              <CheckCircle2 className="size-10 text-emerald-400" />
              <h1 className="text-lg font-semibold">{t("already_responded_title")}</h1>
              <p className="text-sm text-muted-foreground">{t("already_responded_desc")}</p>
            </div>
          )}

          {submitted && (
            <div className="flex flex-col items-center gap-3 py-8 text-center">
              <CheckCircle2 className="size-10 text-emerald-400" />
              <h1 className="text-lg font-semibold">{t("thank_you_title")}</h1>
              <p className="text-sm text-muted-foreground">{t("thank_you_desc")}</p>
            </div>
          )}

          {!isLoading && !isError && data && !data.responded && !submitted && (
            <form onSubmit={onSubmit} className="space-y-6">
              <div className="text-center">
                <h1 className="text-xl font-semibold">{t("title")}</h1>
                <p className="mt-1 text-sm text-muted-foreground">{t("subtitle")}</p>
              </div>

              <div className="space-y-2">
                <div className="text-sm font-medium">{t("rate_heading")}</div>
                <StarPicker value={rating} onChange={setRating} />
              </div>

              <div className="space-y-1.5">
                <label className="text-sm font-medium" htmlFor="survey-comment">
                  {t("comment_label")}
                </label>
                <Textarea
                  id="survey-comment"
                  value={comment}
                  onChange={(e) => setComment(e.target.value)}
                  rows={3}
                  maxLength={2000}
                  placeholder={t("comment_placeholder")}
                />
              </div>

              {submitError && (
                <p className="text-sm text-destructive">{submitError}</p>
              )}

              <Button
                type="submit"
                className="w-full"
                disabled={rating < 1 || submit.isPending}
              >
                {submit.isPending ? t("submitting") : t("submit")}
              </Button>
            </form>
          )}
        </div>

        {/* Powered-by footer */}
        <p className="mt-6 text-xs text-muted-foreground">{t("powered_by")}</p>
      </div>
    </div>
  );
}
