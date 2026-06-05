import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { Sparkles } from "lucide-react";
import { toast } from "sonner";
import { Card } from "@/components/ui/card";
import { Input, Textarea } from "@/components/ui/input";
import { Separator } from "@/components/ui/misc";
import { cn } from "@/lib/utils";
import { apiError } from "@/lib/api";
import { useAISettings, useUpdateAISettings, type AISettings } from "@/features/ai/api";

function Switch({
  checked,
  onChange,
  disabled,
}: {
  checked: boolean;
  onChange: (v: boolean) => void;
  disabled?: boolean;
}) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      disabled={disabled}
      onClick={() => onChange(!checked)}
      className={cn(
        "relative h-6 w-11 shrink-0 rounded-full transition-colors",
        checked ? "bg-[var(--color-primary)]" : "bg-muted-foreground/25",
        disabled && "cursor-not-allowed opacity-40"
      )}
    >
      <span
        className={cn(
          "absolute left-0.5 top-0.5 size-5 rounded-full bg-white shadow transition-transform",
          checked && "translate-x-5"
        )}
      />
    </button>
  );
}

function ToggleRow({
  label,
  desc,
  checked,
  onChange,
  disabled,
}: {
  label: string;
  desc: string;
  checked: boolean;
  onChange: (v: boolean) => void;
  disabled?: boolean;
}) {
  return (
    <div className="flex items-center justify-between gap-4 py-2.5">
      <div className="min-w-0">
        <div className="text-sm font-medium">{label}</div>
        <div className="text-xs text-muted-foreground">{desc}</div>
      </div>
      <Switch checked={checked} onChange={onChange} disabled={disabled} />
    </div>
  );
}

/** Admin AI controls: master switch, per-feature toggles and reply guidance. */
export function AISettingsCard() {
  const { t } = useTranslation("settings");
  const { data, isLoading } = useAISettings();
  const update = useUpdateAISettings();
  const [instructions, setInstructions] = useState("");

  useEffect(() => {
    if (data) setInstructions(data.reply_instructions ?? "");
  }, [data]);

  const [confidence, setConfidence] = useState(
    data ? Math.round((data.auto_reply_confidence ?? 0.75) * 100) : 75
  );
  const [maxReplies, setMaxReplies] = useState(
    data ? (data.max_auto_replies_per_ticket ?? 2) : 2
  );

  useEffect(() => {
    if (data) {
      setConfidence(Math.round((data.auto_reply_confidence ?? 0.75) * 100));
      setMaxReplies(data.max_auto_replies_per_ticket ?? 2);
    }
  }, [data]);

  if (isLoading || !data) {
    return (
      <Card data-reveal className="p-5">
        <div className="h-24 animate-pulse rounded bg-muted" />
      </Card>
    );
  }

  async function set(patch: Partial<AISettings>) {
    try {
      await update.mutateAsync(patch);
    } catch (err) {
      toast.error(apiError(err, t("ai.toast_error")));
    }
  }

  async function saveInstructions() {
    if (instructions === (data?.reply_instructions ?? "")) return;
    try {
      await update.mutateAsync({ reply_instructions: instructions });
      toast.success(t("ai.toast_saved"));
    } catch (err) {
      toast.error(apiError(err, t("ai.toast_error")));
    }
  }

  async function saveConfidence() {
    const clamped = Math.min(100, Math.max(0, confidence));
    if (clamped === Math.round((data?.auto_reply_confidence ?? 0.75) * 100)) return;
    try {
      await update.mutateAsync({ auto_reply_confidence: clamped / 100 });
      toast.success(t("ai.toast_saved"));
    } catch (err) {
      toast.error(apiError(err, t("ai.toast_error")));
    }
  }

  async function saveMaxReplies() {
    const v = Math.max(1, maxReplies);
    if (v === (data?.max_auto_replies_per_ticket ?? 2)) return;
    try {
      await update.mutateAsync({ max_auto_replies_per_ticket: v });
      toast.success(t("ai.toast_saved"));
    } catch (err) {
      toast.error(apiError(err, t("ai.toast_error")));
    }
  }

  const off = !data.enabled;

  return (
    <Card data-reveal className="space-y-1 p-5">
      <div className="mb-1 flex items-center gap-2">
        <Sparkles className="size-4 text-[var(--color-primary)]" />
        <h2 className="text-sm font-semibold">{t("ai.heading")}</h2>
      </div>
      <p className="text-xs text-muted-foreground">{t("ai.desc")}</p>

      <div className="mt-2">
        <ToggleRow
          label={t("ai.enabled")}
          desc={t("ai.enabled_desc")}
          checked={data.enabled}
          onChange={(v) => set({ enabled: v })}
        />
        <Separator />
        <ToggleRow
          label={t("ai.suggest_replies")}
          desc={t("ai.suggest_replies_desc")}
          checked={data.suggest_replies}
          onChange={(v) => set({ suggest_replies: v })}
          disabled={off}
        />
        <ToggleRow
          label={t("ai.knowledge_ai")}
          desc={t("ai.knowledge_ai_desc")}
          checked={data.knowledge_ai}
          onChange={(v) => set({ knowledge_ai: v })}
          disabled={off}
        />
        <ToggleRow
          label={t("ai.auto_classify")}
          desc={t("ai.auto_classify_desc")}
          checked={data.auto_classify}
          onChange={(v) => set({ auto_classify: v })}
          disabled={off}
        />
        <Separator />
        <ToggleRow
          label={t("ai.auto_reply_enabled")}
          desc={t("ai.auto_reply_enabled_desc")}
          checked={data.auto_reply_enabled}
          onChange={(v) => set({ auto_reply_enabled: v })}
          disabled={off}
        />
        {data.auto_reply_enabled && !off && (
          <div className="ml-0 pb-1 pl-0 pt-1">
            <div className="text-xs font-medium text-muted-foreground mb-1">
              {t("ai.auto_reply_confidence")}
            </div>
            <div className="flex items-center gap-3">
              <input
                type="range"
                min={0}
                max={100}
                step={5}
                value={confidence}
                onChange={(e) => setConfidence(Number(e.target.value))}
                onMouseUp={saveConfidence}
                onTouchEnd={saveConfidence}
                className="flex-1 accent-[var(--color-primary)]"
                aria-label={t("ai.auto_reply_confidence")}
              />
              <span className="w-10 text-right text-sm tabular-nums">
                {confidence}%
              </span>
            </div>
            <p className="mt-0.5 text-xs text-muted-foreground">
              {t("ai.auto_reply_confidence_desc")}
            </p>
          </div>
        )}
        <ToggleRow
          label={t("ai.auto_resolve_enabled")}
          desc={t("ai.auto_resolve_enabled_desc")}
          checked={data.auto_resolve_enabled}
          onChange={(v) => set({ auto_resolve_enabled: v })}
          disabled={off}
        />
        <div className="flex items-center justify-between gap-4 py-2.5">
          <div className="min-w-0">
            <div className="text-sm font-medium">{t("ai.max_auto_replies")}</div>
            <div className="text-xs text-muted-foreground">{t("ai.max_auto_replies_desc")}</div>
          </div>
          <Input
            type="number"
            min={1}
            value={maxReplies}
            onChange={(e) => setMaxReplies(Number(e.target.value))}
            onBlur={saveMaxReplies}
            disabled={off}
            className="w-20 text-right"
            aria-label={t("ai.max_auto_replies")}
          />
        </div>
        <ToggleRow
          label={t("ai.auto_summarize_on_resolve")}
          desc={t("ai.auto_summarize_on_resolve_desc")}
          checked={data.auto_summarize_on_resolve}
          onChange={(v) => set({ auto_summarize_on_resolve: v })}
          disabled={off}
        />
      </div>

      <Separator className="my-2" />
      <div className="space-y-1.5 pt-1">
        <Label className="text-sm font-medium">{t("ai.instructions")}</Label>
        <Textarea
          value={instructions}
          onChange={(e) => setInstructions(e.target.value)}
          onBlur={saveInstructions}
          rows={2}
          maxLength={1000}
          placeholder={t("ai.instructions_placeholder")}
          disabled={off}
        />
        <p className="text-xs text-muted-foreground">{t("ai.instructions_hint")}</p>
      </div>
    </Card>
  );
}

// Label kept local-light to avoid an extra import churn.
function Label({ children, className }: { children: React.ReactNode; className?: string }) {
  return <div className={className}>{children}</div>;
}
