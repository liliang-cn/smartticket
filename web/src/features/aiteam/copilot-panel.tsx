import { useState } from "react";
import { useTranslation } from "react-i18next";
import {
  Bot,
  ChevronDown,
  ChevronRight,
  Loader2,
  AlertTriangle,
  BookOpen,
  X,
} from "lucide-react";
import { toast } from "sonner";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { apiError } from "@/lib/api";
import {
  useSuggestions,
  useRunResearcher,
  useRunReviewer,
  useRunDraft,
  useAdoptSuggestion,
  useDismissSuggestion,
  type AISuggestion,
  type AgentName,
  type TriagePayload,
  type SentinelPayload,
  type ResearcherPayload,
  type ReviewerPayload,
  type DrafterPayload,
} from "./api";

// ─── helpers ─────────────────────────────────────────────────────────────────

function parsePayload<T>(raw: string): T | null {
  try {
    return JSON.parse(raw) as T;
  } catch {
    return null;
  }
}

function confidenceLabel(c: number): string {
  if (c >= 0.7) return "high";
  if (c >= 0.4) return "med";
  return "low";
}

const CONFIDENCE_STYLE: Record<string, string> = {
  high: "border-emerald-500/30 bg-emerald-500/10 text-emerald-300",
  med: "border-amber-500/30 bg-amber-500/10 text-amber-300",
  low: "border-border/60 bg-muted text-muted-foreground",
};

const AGENT_ORDER: AgentName[] = ["Triage", "Sentinel", "Researcher", "Reviewer", "Drafter"];

// ─── ConfidenceBadge ──────────────────────────────────────────────────────────

function ConfidenceBadge({ confidence }: { confidence: number }) {
  const { t } = useTranslation("aiteam");
  const label = confidenceLabel(confidence);
  return (
    <span
      className={`rounded-full border px-2 py-0.5 font-mono text-[10px] uppercase tracking-wider ${CONFIDENCE_STYLE[label]}`}
    >
      {t(`confidence.${label}`)} {Math.round(confidence * 100)}%
    </span>
  );
}

// ─── ReasoningSection ─────────────────────────────────────────────────────────

function ReasoningSection({ reasoning }: { reasoning: string }) {
  const { t } = useTranslation("aiteam");
  const [open, setOpen] = useState(false);
  return (
    <div className="mt-2">
      <button
        type="button"
        className="flex items-center gap-1 font-mono text-[11px] text-muted-foreground hover:text-foreground"
        onClick={() => setOpen((v) => !v)}
      >
        {open ? <ChevronDown className="size-3" /> : <ChevronRight className="size-3" />}
        {t("reasoning_label")}
      </button>
      {open && (
        <p className="mt-1.5 rounded-md bg-muted/40 px-3 py-2 text-xs leading-relaxed text-foreground/80">
          {reasoning}
        </p>
      )}
    </div>
  );
}

// ─── TriageCard ───────────────────────────────────────────────────────────────

interface TriageCardProps {
  suggestion: AISuggestion;
  onApplyFields: (fields: { priority?: string; severity?: string; category?: string }) => void;
  onAdopt: () => void;
  onDismiss: () => void;
  adopting: boolean;
  dismissing: boolean;
  dimmed: boolean;
}

function TriageCard({
  suggestion,
  onApplyFields,
  onAdopt,
  onDismiss,
  adopting,
  dismissing,
  dimmed,
}: TriageCardProps) {
  const { t } = useTranslation("aiteam");
  const p = parsePayload<TriagePayload>(suggestion.payload);
  if (!p) return null;

  function handleApply() {
    onApplyFields({ priority: p!.priority, severity: p!.severity, category: p!.category });
    onAdopt();
  }

  return (
    <div className={dimmed ? "opacity-50" : ""}>
      <div className="space-y-1 text-sm">
        <div className="flex flex-wrap gap-2">
          <MetaChip label={t("triage.priority")} value={p.priority} />
          <MetaChip label={t("triage.severity")} value={p.severity} />
          {p.category && <MetaChip label={t("triage.category")} value={p.category} />}
        </div>
      </div>
      <ReasoningSection reasoning={p.reasoning} />
      <div className="mt-3 flex items-center gap-2">
        <Button
          size="sm"
          onClick={handleApply}
          disabled={adopting || dismissing || suggestion.status === "adopted"}
        >
          {adopting ? <Loader2 className="animate-spin" /> : null}
          {suggestion.status === "adopted" ? t("adopted_label") : t("triage.apply_btn")}
        </Button>
        {suggestion.status !== "dismissed" && (
          <Button
            size="sm"
            variant="ghost"
            onClick={onDismiss}
            disabled={dismissing || adopting}
          >
            {dismissing ? <Loader2 className="animate-spin" /> : <X className="size-3" />}
            {t("dismiss_btn")}
          </Button>
        )}
      </div>
    </div>
  );
}

// ─── SentinelCard ─────────────────────────────────────────────────────────────

interface SentinelCardProps {
  suggestion: AISuggestion;
  onAdopt: () => void;
  onDismiss: () => void;
  adopting: boolean;
  dismissing: boolean;
  dimmed: boolean;
}

function SentinelCard({
  suggestion,
  onAdopt,
  onDismiss,
  adopting,
  dismissing,
  dimmed,
}: SentinelCardProps) {
  const { t } = useTranslation("aiteam");
  const p = parsePayload<SentinelPayload>(suggestion.payload);
  if (!p) return null;

  return (
    <div className={dimmed ? "opacity-50" : ""}>
      <div className="flex flex-wrap gap-2 text-sm">
        <MetaChip label={t("sentinel.sentiment")} value={p.sentiment} />
        <MetaChip label={t("sentinel.churn_risk")} value={p.churn_risk} />
        <MetaChip label={t("sentinel.sla_risk")} value={p.sla_breach_risk} />
      </div>
      {p.escalate && (
        <div className="mt-2 flex items-center gap-1.5 rounded-md border border-red-500/30 bg-red-500/10 px-2 py-1.5 text-xs text-red-300">
          <AlertTriangle className="size-3.5 shrink-0" />
          {t("sentinel.escalate_badge")}
        </div>
      )}
      <ReasoningSection reasoning={p.reasoning} />
      <div className="mt-3 flex items-center gap-2">
        <Button
          size="sm"
          onClick={onAdopt}
          disabled={adopting || dismissing || suggestion.status === "adopted"}
        >
          {adopting ? <Loader2 className="animate-spin" /> : null}
          {suggestion.status === "adopted" ? t("adopted_label") : t("adopt_btn")}
        </Button>
        {suggestion.status !== "dismissed" && (
          <Button
            size="sm"
            variant="ghost"
            onClick={onDismiss}
            disabled={dismissing || adopting}
          >
            {dismissing ? <Loader2 className="animate-spin" /> : <X className="size-3" />}
            {t("dismiss_btn")}
          </Button>
        )}
      </div>
    </div>
  );
}

// ─── ResearcherCard ───────────────────────────────────────────────────────────

interface ResearcherCardProps {
  suggestion: AISuggestion;
  onInsertReply: (text: string) => void;
  onAdopt: () => void;
  onDismiss: () => void;
  onRun: () => void;
  running: boolean;
  adopting: boolean;
  dismissing: boolean;
  dimmed: boolean;
}

function ResearcherCard({
  suggestion,
  onInsertReply,
  onAdopt,
  onDismiss,
  onRun,
  running,
  adopting,
  dismissing,
  dimmed,
}: ResearcherCardProps) {
  const { t } = useTranslation("aiteam");
  const p = parsePayload<ResearcherPayload>(suggestion.payload);
  if (!p) return null;

  function handleInsert() {
    onInsertReply(p!.suggested_resolution);
    onAdopt();
  }

  return (
    <div className={dimmed ? "opacity-50" : ""}>
      {p.kb_citations.length > 0 && (
        <div className="mb-2">
          <div className="mb-1 flex items-center gap-1 font-mono text-[10px] uppercase tracking-wider text-muted-foreground">
            <BookOpen className="size-3" /> {t("researcher.kb_label")}
          </div>
          <ul className="space-y-1">
            {p.kb_citations.map((c, i) => (
              <li key={i} className="rounded-md border border-border/60 bg-muted/30 px-2 py-1.5 text-xs">
                <div className="font-medium">{c.title}</div>
                <div className="mt-0.5 text-muted-foreground line-clamp-2">{c.snippet}</div>
              </li>
            ))}
          </ul>
        </div>
      )}
      {p.similar_tickets.length > 0 && (
        <div className="mb-2">
          <div className="mb-1 font-mono text-[10px] uppercase tracking-wider text-muted-foreground">
            {t("researcher.similar_label")}
          </div>
          <ul className="space-y-1">
            {p.similar_tickets.map((st) => (
              <li key={st.id} className="rounded-md border border-border/60 bg-muted/30 px-2 py-1.5 text-xs">
                <div className="flex items-center gap-1.5">
                  <span className="font-mono text-primary/80">#{st.id}</span>
                  <span className="truncate font-medium">{st.title}</span>
                  {st.merge_candidate && (
                    <span className="ml-auto shrink-0 rounded-full border border-primary/30 bg-primary/10 px-1.5 py-0.5 font-mono text-[9px] uppercase text-primary">
                      {t("researcher.merge_tag")}
                    </span>
                  )}
                </div>
                {st.resolution && (
                  <div className="mt-0.5 text-muted-foreground line-clamp-2">{st.resolution}</div>
                )}
              </li>
            ))}
          </ul>
        </div>
      )}
      {p.suggested_resolution && (
        <div className="rounded-md border border-border/60 bg-muted/30 px-2 py-1.5 text-xs">
          <div className="mb-0.5 font-mono text-[10px] uppercase tracking-wider text-muted-foreground">
            {t("researcher.resolution_label")}
          </div>
          <p className="whitespace-pre-wrap leading-relaxed">{p.suggested_resolution}</p>
        </div>
      )}
      <div className="mt-3 flex flex-wrap items-center gap-2">
        {p.suggested_resolution && (
          <Button
            size="sm"
            onClick={handleInsert}
            disabled={adopting || dismissing || suggestion.status === "adopted"}
          >
            {adopting ? <Loader2 className="animate-spin" /> : null}
            {suggestion.status === "adopted" ? t("adopted_label") : t("researcher.insert_btn")}
          </Button>
        )}
        <Button size="sm" variant="secondary" onClick={onRun} disabled={running}>
          {running ? <Loader2 className="animate-spin" /> : null}
          {t("run_btn")}
        </Button>
        {suggestion.status !== "dismissed" && (
          <Button
            size="sm"
            variant="ghost"
            onClick={onDismiss}
            disabled={dismissing || adopting}
          >
            {dismissing ? <Loader2 className="animate-spin" /> : <X className="size-3" />}
            {t("dismiss_btn")}
          </Button>
        )}
      </div>
    </div>
  );
}

// ─── ReviewerCard ─────────────────────────────────────────────────────────────

interface ReviewerCardProps {
  suggestion: AISuggestion;
  currentDraft: string;
  onInsertReply: (text: string) => void;
  onAdopt: () => void;
  onDismiss: () => void;
  onRun: () => void;
  running: boolean;
  adopting: boolean;
  dismissing: boolean;
  dimmed: boolean;
}

function ReviewerCard({
  suggestion,
  currentDraft,
  onInsertReply,
  onAdopt,
  onDismiss,
  onRun,
  running,
  adopting,
  dismissing,
  dimmed,
}: ReviewerCardProps) {
  const { t } = useTranslation("aiteam");
  const p = parsePayload<ReviewerPayload>(suggestion.payload);
  if (!p) return null;

  function handleUseDraft() {
    if (p?.revised_draft) {
      onInsertReply(p.revised_draft);
      onAdopt();
    }
  }

  return (
    <div className={dimmed ? "opacity-50" : ""}>
      {p.issues.length > 0 && (
        <ul className="mb-2 space-y-1">
          {p.issues.map((issue, i) => (
            <li
              key={i}
              className="flex gap-2 rounded-md border border-border/60 bg-muted/30 px-2 py-1.5 text-xs"
            >
              <span
                className={`shrink-0 rounded-full border px-1.5 py-0.5 font-mono text-[9px] uppercase ${
                  issue.severity === "high"
                    ? "border-red-500/30 bg-red-500/10 text-red-300"
                    : issue.severity === "medium"
                    ? "border-amber-500/30 bg-amber-500/10 text-amber-300"
                    : "border-border/60 bg-muted text-muted-foreground"
                }`}
              >
                {issue.severity}
              </span>
              <div>
                <div className="font-medium">{issue.type}</div>
                <div className="text-muted-foreground">{issue.note}</div>
              </div>
            </li>
          ))}
        </ul>
      )}
      {p.approve && (
        <div className="mb-2 flex items-center gap-1.5 rounded-md border border-emerald-500/30 bg-emerald-500/10 px-2 py-1.5 text-xs text-emerald-300">
          {t("reviewer.approved_badge")}
        </div>
      )}
      {p.revised_draft && (
        <div className="mb-2 rounded-md border border-border/60 bg-muted/30 px-2 py-1.5 text-xs">
          <div className="mb-0.5 font-mono text-[10px] uppercase tracking-wider text-muted-foreground">
            {t("reviewer.revised_draft_label")}
          </div>
          <p className="whitespace-pre-wrap leading-relaxed line-clamp-4">{p.revised_draft}</p>
        </div>
      )}
      <div className="mt-3 flex flex-wrap items-center gap-2">
        {p.revised_draft && (
          <Button
            size="sm"
            onClick={handleUseDraft}
            disabled={adopting || dismissing || suggestion.status === "adopted"}
          >
            {adopting ? <Loader2 className="animate-spin" /> : null}
            {suggestion.status === "adopted" ? t("adopted_label") : t("reviewer.use_draft_btn")}
          </Button>
        )}
        <Button
          size="sm"
          variant="secondary"
          onClick={onRun}
          disabled={running || !currentDraft.trim()}
          title={!currentDraft.trim() ? t("reviewer.run_needs_draft") : undefined}
        >
          {running ? <Loader2 className="animate-spin" /> : null}
          {t("run_btn")}
        </Button>
        {suggestion.status !== "dismissed" && (
          <Button
            size="sm"
            variant="ghost"
            onClick={onDismiss}
            disabled={dismissing || adopting}
          >
            {dismissing ? <Loader2 className="animate-spin" /> : <X className="size-3" />}
            {t("dismiss_btn")}
          </Button>
        )}
      </div>
    </div>
  );
}

// ─── DrafterCard ──────────────────────────────────────────────────────────────

interface DrafterCardProps {
  suggestion: AISuggestion;
  onInsertReply: (text: string) => void;
  onAdopt: () => void;
  onDismiss: () => void;
  onRun: () => void;
  running: boolean;
  adopting: boolean;
  dismissing: boolean;
  dimmed: boolean;
}

function DrafterCard({
  suggestion,
  onInsertReply,
  onAdopt,
  onDismiss,
  onRun,
  running,
  adopting,
  dismissing,
  dimmed,
}: DrafterCardProps) {
  const { t } = useTranslation("aiteam");
  const p = parsePayload<DrafterPayload>(suggestion.payload);
  if (!p) return null;

  function handleUseDraft() {
    onInsertReply(p!.reply);
    onAdopt();
  }

  return (
    <div className={dimmed ? "opacity-50" : ""}>
      <div className="rounded-md border border-border/60 bg-muted/30 px-2 py-1.5 text-xs">
        <p className="whitespace-pre-wrap leading-relaxed">{p.reply}</p>
      </div>
      <div className="mt-3 flex flex-wrap items-center gap-2">
        <Button
          size="sm"
          onClick={handleUseDraft}
          disabled={adopting || dismissing || suggestion.status === "adopted"}
        >
          {adopting ? <Loader2 className="animate-spin" /> : null}
          {suggestion.status === "adopted" ? t("adopted_label") : t("drafter.use_draft_btn")}
        </Button>
        <Button size="sm" variant="secondary" onClick={onRun} disabled={running}>
          {running ? <Loader2 className="animate-spin" /> : null}
          {t("run_btn")}
        </Button>
        {suggestion.status !== "dismissed" && (
          <Button
            size="sm"
            variant="ghost"
            onClick={onDismiss}
            disabled={dismissing || adopting}
          >
            {dismissing ? <Loader2 className="animate-spin" /> : <X className="size-3" />}
            {t("dismiss_btn")}
          </Button>
        )}
      </div>
    </div>
  );
}

// ─── MetaChip ─────────────────────────────────────────────────────────────────

function MetaChip({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center gap-1 rounded-md border border-border/60 bg-muted/50 px-2 py-1 text-xs">
      <span className="font-mono text-[10px] uppercase tracking-wider text-muted-foreground">
        {label}
      </span>
      <span className="font-medium capitalize">{value}</span>
    </div>
  );
}

// ─── AgentCard wrapper ────────────────────────────────────────────────────────

function AgentCardWrapper({
  agentName,
  suggestion,
  children,
}: {
  agentName: AgentName;
  suggestion: AISuggestion;
  children: React.ReactNode;
}) {
  const { t } = useTranslation("aiteam");
  return (
    <Card className="p-4">
      <div className="mb-3 flex items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <Bot className="size-4 text-primary/70" />
          <span className="font-semibold text-sm">{t(`agent.${agentName}`)}</span>
        </div>
        <div className="flex items-center gap-2">
          <ConfidenceBadge confidence={suggestion.confidence} />
          {suggestion.status === "dismissed" && (
            <span className="rounded-full border border-border/60 bg-muted px-2 py-0.5 font-mono text-[10px] uppercase text-muted-foreground">
              {t("dismissed_label")}
            </span>
          )}
          {suggestion.status === "adopted" && (
            <span className="rounded-full border border-emerald-500/30 bg-emerald-500/10 px-2 py-0.5 font-mono text-[10px] uppercase text-emerald-300">
              {t("adopted_label")}
            </span>
          )}
        </div>
      </div>
      {children}
    </Card>
  );
}

// ─── CopilotPanel ─────────────────────────────────────────────────────────────

export interface CopilotPanelProps {
  ticketId: number;
  /** Current text in the reply draft textarea. */
  currentDraft: string;
  /** Called when an agent result should populate the reply box. */
  onInsertReply: (text: string) => void;
  /** Called when Triage Apply should patch ticket fields. */
  onApplyFields: (fields: { priority?: string; severity?: string; category?: string }) => void;
}

export function CopilotPanel({
  ticketId,
  currentDraft,
  onInsertReply,
  onApplyFields,
}: CopilotPanelProps) {
  const { t } = useTranslation("aiteam");
  const { data: suggestions, isLoading } = useSuggestions(ticketId);

  const runResearcher = useRunResearcher(ticketId);
  const runReviewer = useRunReviewer(ticketId);
  const runDraft = useRunDraft(ticketId);
  const adopt = useAdoptSuggestion(ticketId);
  const dismiss = useDismissSuggestion(ticketId);

  // Index suggestions by agent_name (keep latest per agent)
  const byAgent = (suggestions ?? []).reduce<Record<string, AISuggestion>>((acc, s) => {
    const prev = acc[s.agent_name];
    if (!prev || s.id > prev.id) acc[s.agent_name] = s;
    return acc;
  }, {});

  const agentsWithSuggestions = AGENT_ORDER.filter((name) => byAgent[name]);

  async function handleAdopt(sid: number) {
    try {
      await adopt.mutateAsync(sid);
    } catch (err) {
      toast.error(apiError(err, t("toast.adopt_error")));
    }
  }

  async function handleDismiss(sid: number) {
    try {
      await dismiss.mutateAsync(sid);
    } catch (err) {
      toast.error(apiError(err, t("toast.dismiss_error")));
    }
  }

  async function handleRunResearcher() {
    try {
      await runResearcher.mutateAsync();
      toast.success(t("toast.research_done"));
    } catch (err) {
      toast.error(apiError(err, t("toast.run_error")));
    }
  }

  async function handleRunReviewer() {
    try {
      await runReviewer.mutateAsync(currentDraft);
      toast.success(t("toast.review_done"));
    } catch (err) {
      toast.error(apiError(err, t("toast.run_error")));
    }
  }

  async function handleRunDraft() {
    try {
      await runDraft.mutateAsync();
      toast.success(t("toast.draft_done"));
    } catch (err) {
      toast.error(apiError(err, t("toast.run_error")));
    }
  }

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Bot className="size-4 text-primary" />
          <Label>{t("panel_title")}</Label>
        </div>
        {/* Quick-run toolbar for agents without existing suggestions */}
        <div className="flex items-center gap-1">
          {!byAgent["Researcher"] && (
            <Button
              size="sm"
              variant="ghost"
              onClick={handleRunResearcher}
              disabled={runResearcher.isPending}
              title={t("researcher.run_hint")}
            >
              {runResearcher.isPending ? (
                <Loader2 className="size-3 animate-spin" />
              ) : null}
              {t("agent.Researcher")}
            </Button>
          )}
          {!byAgent["Drafter"] && (
            <Button
              size="sm"
              variant="ghost"
              onClick={handleRunDraft}
              disabled={runDraft.isPending}
              title={t("drafter.run_hint")}
            >
              {runDraft.isPending ? (
                <Loader2 className="size-3 animate-spin" />
              ) : null}
              {t("agent.Drafter")}
            </Button>
          )}
        </div>
      </div>

      {isLoading ? (
        <div className="flex items-center gap-2 text-sm text-muted-foreground py-3">
          <Loader2 className="size-4 animate-spin" />
          {t("loading")}
        </div>
      ) : agentsWithSuggestions.length === 0 ? (
        <p className="text-sm text-muted-foreground py-2">{t("empty")}</p>
      ) : (
        agentsWithSuggestions.map((agentName) => {
          const s = byAgent[agentName];
          const dimmed = s.confidence < 0.4 && s.status !== "adopted";
          const isAdopting = adopt.isPending && adopt.variables === s.id;
          const isDismissing = dismiss.isPending && dismiss.variables === s.id;

          return (
            <AgentCardWrapper key={agentName} agentName={agentName} suggestion={s}>
              {agentName === "Triage" && (
                <TriageCard
                  suggestion={s}
                  onApplyFields={onApplyFields}
                  onAdopt={() => handleAdopt(s.id)}
                  onDismiss={() => handleDismiss(s.id)}
                  adopting={isAdopting}
                  dismissing={isDismissing}
                  dimmed={dimmed}
                />
              )}
              {agentName === "Sentinel" && (
                <SentinelCard
                  suggestion={s}
                  onAdopt={() => handleAdopt(s.id)}
                  onDismiss={() => handleDismiss(s.id)}
                  adopting={isAdopting}
                  dismissing={isDismissing}
                  dimmed={dimmed}
                />
              )}
              {agentName === "Researcher" && (
                <ResearcherCard
                  suggestion={s}
                  onInsertReply={onInsertReply}
                  onAdopt={() => handleAdopt(s.id)}
                  onDismiss={() => handleDismiss(s.id)}
                  onRun={handleRunResearcher}
                  running={runResearcher.isPending}
                  adopting={isAdopting}
                  dismissing={isDismissing}
                  dimmed={dimmed}
                />
              )}
              {agentName === "Reviewer" && (
                <ReviewerCard
                  suggestion={s}
                  currentDraft={currentDraft}
                  onInsertReply={onInsertReply}
                  onAdopt={() => handleAdopt(s.id)}
                  onDismiss={() => handleDismiss(s.id)}
                  onRun={handleRunReviewer}
                  running={runReviewer.isPending}
                  adopting={isAdopting}
                  dismissing={isDismissing}
                  dimmed={dimmed}
                />
              )}
              {agentName === "Drafter" && (
                <DrafterCard
                  suggestion={s}
                  onInsertReply={onInsertReply}
                  onAdopt={() => handleAdopt(s.id)}
                  onDismiss={() => handleDismiss(s.id)}
                  onRun={handleRunDraft}
                  running={runDraft.isPending}
                  adopting={isAdopting}
                  dismissing={isDismissing}
                  dimmed={dimmed}
                />
              )}
            </AgentCardWrapper>
          );
        })
      )}
    </div>
  );
}
