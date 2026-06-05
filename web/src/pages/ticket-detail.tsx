import { useCallback, useEffect, useRef, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { useTranslation } from "react-i18next";
import {
  ArrowLeft,
  Send,
  Sparkles,
  Lock,
  Bot,
  Download,
  Paperclip,
  Loader2,
  Timer,
  History,
  CircleDot,
  GitMerge,
  Link2,
  X,
  AlertTriangle,
  ChevronDown,
} from "lucide-react";
import { toast } from "sonner";
import {
  useTicket,
  useTicketMessages,
  useTicketSLA,
  useTicketEvents,
  useUpdateTicket,
  useAddMessage,
  useAssignTicket,
  useTicketAttachments,
  useUploadAttachment,
  downloadAttachment,
  useMergeTicket,
  useTicketLinks,
  useCreateTicketLink,
  useDeleteTicketLink,
} from "@/features/tickets/api";
import type { TicketLink } from "@/features/tickets/api";
import { useUsers } from "@/features/users/api";
import { useAISettings, useSuggestReply } from "@/features/ai/api";
import { useMacros, useApplyMacro } from "@/features/macros/api";
import { useAuth } from "@/lib/auth";
import { tokenStore } from "@/lib/api";
import type { Attachment, Ticket, UserInfo } from "@/lib/types";
import { apiError } from "@/lib/api";
import { relativeTime } from "@/lib/utils";
import {
  STATUS_OPTIONS,
  PRIORITY_OPTIONS,
  PriorityBadge,
} from "@/components/ticket-meta";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton, Separator, Avatar, AvatarFallback } from "@/components/ui/misc";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useReveal } from "@/lib/use-reveal";
import { useQueryClient } from "@tanstack/react-query";

// ─── helpers ────────────────────────────────────────────────────────────────

function formatBytes(n: number): string {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  return `${(n / (1024 * 1024)).toFixed(1)} MB`;
}

function userDisplayName(u: Pick<UserInfo, "first_name" | "last_name" | "username">) {
  return `${u.first_name} ${u.last_name}`.trim() || u.username;
}

/**
 * Render message body with @mention tokens highlighted.
 * Splits on word-boundary @handle and wraps each mention in a span.
 */
function renderWithMentions(content: string): React.ReactNode {
  const parts = content.split(/(@\w+)/g);
  return parts.map((part, i) =>
    /^@\w+$/.test(part) ? (
      <span
        key={i}
        className="rounded bg-primary/15 px-1 font-medium text-primary"
      >
        {part}
      </span>
    ) : (
      part
    )
  );
}

// ─── AttachmentsCard ────────────────────────────────────────────────────────

function AttachmentsCard({ ticketId }: { ticketId: number }) {
  const { t } = useTranslation("tickets");
  const { data: attachments, isLoading } = useTicketAttachments(ticketId);
  const upload = useUploadAttachment(ticketId);
  const fileInput = useRef<HTMLInputElement>(null);
  const [downloading, setDownloading] = useState<number | null>(null);

  async function onPick(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    e.target.value = "";
    if (!file) return;
    try {
      await upload.mutateAsync(file);
      toast.success(t("detail.attachments.toast_uploaded"));
    } catch (err) {
      toast.error(apiError(err, t("detail.attachments.toast_upload_error")));
    }
  }

  async function onDownload(att: Attachment) {
    setDownloading(att.id);
    try {
      await downloadAttachment(att);
    } catch (err) {
      toast.error(apiError(err, t("detail.attachments.toast_download_error")));
    } finally {
      setDownloading(null);
    }
  }

  return (
    <Card className="p-5">
      <div className="mb-3 flex items-center justify-between">
        <Label>{t("detail.attachments.label")}</Label>
        <input ref={fileInput} type="file" className="hidden" onChange={onPick} />
        <Button
          size="sm"
          variant="secondary"
          onClick={() => fileInput.current?.click()}
          disabled={upload.isPending}
        >
          {upload.isPending ? <Loader2 className="animate-spin" /> : <Paperclip />}
          {upload.isPending
            ? t("detail.attachments.btn_uploading")
            : t("detail.attachments.btn_attach")}
        </Button>
      </div>
      {isLoading ? (
        <Skeleton className="h-10 w-full" />
      ) : attachments && attachments.length > 0 ? (
        <ul className="space-y-1.5">
          {attachments.map((att) => (
            <li
              key={att.id}
              className="flex items-center justify-between gap-3 rounded-md border border-border/60 px-3 py-2 text-sm"
            >
              <div className="min-w-0">
                <div className="truncate font-medium">{att.original_name}</div>
                <div className="font-mono text-[11px] text-muted-foreground">
                  {formatBytes(att.file_size)} · {relativeTime(att.created_at)}
                </div>
              </div>
              <Button
                size="icon"
                variant="ghost"
                onClick={() => onDownload(att)}
                disabled={downloading === att.id}
                aria-label={t("detail.attachments.btn_download_aria", { name: att.original_name })}
              >
                {downloading === att.id ? (
                  <Loader2 className="animate-spin" />
                ) : (
                  <Download />
                )}
              </Button>
            </li>
          ))}
        </ul>
      ) : (
        <p className="text-sm text-muted-foreground">{t("detail.attachments.empty")}</p>
      )}
    </Card>
  );
}

// ─── AssigneeControl ────────────────────────────────────────────────────────

function AssigneeControl({ ticket }: { ticket: Ticket }) {
  const { t } = useTranslation("tickets");
  const assign = useAssignTicket(ticket.id);
  const { data: usersPage } = useUsers({ page: 1, page_size: 100 });
  const team = (usersPage?.items ?? []).filter((u) => u.role !== "customer");

  const options = [...team];
  if (ticket.assigned_user && !options.some((u) => u.id === ticket.assigned_user!.id)) {
    options.push(ticket.assigned_user);
  }

  async function onAssign(value: string) {
    const userId = Number(value);
    const picked = options.find((u) => u.id === userId);
    try {
      await assign.mutateAsync(userId);
      toast.success(
        t("detail.toast_updated_field", {
          field: picked ? userDisplayName(picked) : `#${userId}`,
        })
      );
    } catch (err) {
      toast.error(apiError(err));
    }
  }

  return (
    <Select
      value={ticket.assigned_to ? String(ticket.assigned_to) : ""}
      onValueChange={onAssign}
      disabled={assign.isPending}
    >
      <SelectTrigger className="h-8 w-44">
        <SelectValue placeholder={t("detail.meta.unassigned")} />
      </SelectTrigger>
      <SelectContent>
        {options.map((u) => (
          <SelectItem key={u.id} value={String(u.id)}>
            {userDisplayName(u)}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}

// ─── MetaRow ────────────────────────────────────────────────────────────────

function MetaRow({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="flex items-center justify-between gap-3 py-2 text-sm">
      <span className="font-mono text-[11px] uppercase tracking-wider text-muted-foreground">
        {label}
      </span>
      <span className="text-right">{value}</span>
    </div>
  );
}

// ─── SlaCard ────────────────────────────────────────────────────────────────

function fmtMinutes(m: number): string {
  if (!m || m <= 0) return "—";
  if (m % 1440 === 0) return `${m / 1440}d`;
  if (m % 60 === 0) return `${m / 60}h`;
  if (m > 60) return `${Math.floor(m / 60)}h ${m % 60}m`;
  return `${m}m`;
}

const SLA_STATUS_STYLE: Record<string, string> = {
  within: "border-emerald-500/30 bg-emerald-500/10 text-emerald-300",
  warning: "border-amber-500/30 bg-amber-500/10 text-amber-300",
  breached: "border-red-500/30 bg-red-500/10 text-red-300",
};

function SlaCard({ ticketId }: { ticketId: number }) {
  const { t } = useTranslation("tickets");
  const { data: sla, isLoading } = useTicketSLA(ticketId);
  if (isLoading || !sla) return null;
  const statusClass =
    SLA_STATUS_STYLE[sla.sla_status] ?? "border-border bg-muted text-muted-foreground";
  return (
    <Card className="p-5">
      <div className="mb-3 flex items-center justify-between">
        <Label>{t("detail.sla.label")}</Label>
        <span
          className={`rounded-full border px-2 py-0.5 font-mono text-[10px] uppercase tracking-wider ${statusClass}`}
        >
          {sla.sla_status || "—"}
        </span>
      </div>
      <div className="flex items-center gap-2">
        <Timer className="size-4 text-primary" />
        <span className="font-medium">{sla.policy_name}</span>
      </div>
      <p className="mt-1 font-mono text-[11px] text-muted-foreground">
        {sla.source === "rule"
          ? t("detail.sla.source_rule", { priority: sla.priority, severity: sla.severity })
          : t("detail.sla.source_default")}
        {sla.business_only ? t("detail.sla.business_hours") : ""}
      </p>
      <Separator className="my-3" />
      <MetaRow label={t("detail.sla.meta_response")} value={fmtMinutes(sla.response_minutes)} />
      <MetaRow label={t("detail.sla.meta_resolution")} value={fmtMinutes(sla.resolution_minutes)} />
      <MetaRow
        label={t("detail.sla.meta_due")}
        value={sla.due_date ? relativeTime(sla.due_date) : "—"}
      />
    </Card>
  );
}

// ─── ActivityCard ────────────────────────────────────────────────────────────

function ActivityCard({ ticketId }: { ticketId: number }) {
  const { t } = useTranslation("tickets");
  const { data: events, isLoading } = useTicketEvents(ticketId);
  return (
    <Card className="p-5">
      <div className="mb-3 flex items-center gap-2">
        <History className="size-4 text-muted-foreground" />
        <Label>{t("detail.activity.label")}</Label>
        <span className="font-mono text-xs text-muted-foreground">
          ({events?.length ?? 0})
        </span>
      </div>
      {isLoading ? (
        <p className="text-sm text-muted-foreground">{t("detail.activity.loading")}</p>
      ) : events && events.length > 0 ? (
        <ol className="relative space-y-3 border-l border-border pl-5">
          {events.map((e) => (
            <li key={e.id} className="relative">
              <CircleDot className="absolute -left-[1.42rem] top-0.5 size-3.5 text-primary/70" />
              <div className="text-sm">
                <span className="font-medium">{e.actor_name || t("detail.activity.actor_system")}</span>{" "}
                <span className="text-muted-foreground">{e.summary}</span>
              </div>
              <div className="font-mono text-[11px] text-muted-foreground/70">
                {relativeTime(e.created_at)}
                {e.actor_role ? ` · ${e.actor_role}` : ""}
              </div>
            </li>
          ))}
        </ol>
      ) : (
        <p className="text-sm text-muted-foreground">{t("detail.activity.empty")}</p>
      )}
    </Card>
  );
}

// ─── LinkedTicketsCard ──────────────────────────────────────────────────────

const LINK_TYPES = ["related", "duplicate", "blocks"] as const;

function LinkedTicketsCard({ ticketId }: { ticketId: number }) {
  const { t } = useTranslation("tickets");
  const { data: links, isLoading } = useTicketLinks(ticketId);
  const createLink = useCreateTicketLink(ticketId);
  const deleteLink = useDeleteTicketLink(ticketId);

  const [showAdd, setShowAdd] = useState(false);
  const [targetId, setTargetId] = useState("");
  const [linkType, setLinkType] = useState<string>("related");

  async function handleAdd() {
    const tid = parseInt(targetId, 10);
    if (!tid || isNaN(tid)) return;
    try {
      await createLink.mutateAsync({ target_id: tid, type: linkType });
      toast.success(t("detail.links_toast_added"));
      setTargetId("");
      setShowAdd(false);
    } catch (err) {
      toast.error(apiError(err, t("detail.links_toast_error")));
    }
  }

  async function handleRemove(link: TicketLink) {
    try {
      await deleteLink.mutateAsync(link.id);
      toast.success(t("detail.links_toast_removed"));
    } catch (err) {
      toast.error(apiError(err));
    }
  }

  return (
    <Card className="p-5">
      <div className="mb-3 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Link2 className="size-4 text-muted-foreground" />
          <Label>{t("detail.links_section")}</Label>
          {links && links.length > 0 && (
            <span className="font-mono text-xs text-muted-foreground">({links.length})</span>
          )}
        </div>
        <Button size="sm" variant="ghost" onClick={() => setShowAdd((v) => !v)}>
          {showAdd ? <X className="size-3.5" /> : <Link2 className="size-3.5" />}
          {t("detail.links_add_btn")}
        </Button>
      </div>

      {showAdd && (
        <div className="mb-3 flex flex-wrap items-center gap-2 rounded-md border border-border/60 p-3">
          <input
            type="number"
            value={targetId}
            onChange={(e) => setTargetId(e.target.value)}
            placeholder={t("detail.links_target_placeholder")}
            className="h-8 w-32 rounded-md border border-input bg-transparent px-2 text-sm focus:outline-none focus:ring-1 focus:ring-primary"
          />
          <Select value={linkType} onValueChange={setLinkType}>
            <SelectTrigger className="h-8 w-32">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {LINK_TYPES.map((lt) => (
                <SelectItem key={lt} value={lt}>
                  {t(`detail.links_type_${lt}`)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Button
            size="sm"
            onClick={handleAdd}
            disabled={!targetId || createLink.isPending}
          >
            {createLink.isPending ? <Loader2 className="animate-spin" /> : null}
            {t("detail.links_add_btn")}
          </Button>
        </div>
      )}

      {isLoading ? (
        <Skeleton className="h-8 w-full" />
      ) : links && links.length > 0 ? (
        <ul className="space-y-1.5">
          {links.map((link) => (
            <li
              key={link.id}
              className="flex items-center justify-between gap-2 rounded-md border border-border/60 px-3 py-2 text-sm"
            >
              <div className="min-w-0 flex items-center gap-2">
                <span className="rounded-full border border-border/60 bg-muted px-1.5 py-0.5 font-mono text-[10px] uppercase">
                  {t(`detail.links_type_${link.type}` as never) || link.type}
                </span>
                <span className="font-mono text-xs text-primary/80">#{link.other_ticket.id}</span>
                <span className="truncate text-foreground/80">{link.other_ticket.title}</span>
                <span className="shrink-0 font-mono text-[10px] uppercase text-muted-foreground">
                  {link.other_ticket.status}
                </span>
              </div>
              <Button
                size="icon"
                variant="ghost"
                className="size-6 shrink-0"
                onClick={() => handleRemove(link)}
                disabled={deleteLink.isPending}
              >
                <X className="size-3" />
              </Button>
            </li>
          ))}
        </ul>
      ) : (
        <p className="text-sm text-muted-foreground">{t("detail.links_empty")}</p>
      )}
    </Card>
  );
}

// ─── MacroPicker ─────────────────────────────────────────────────────────────

interface MacroPickerProps {
  ticketId: number;
  onInsert: (text: string) => void;
  disabled?: boolean;
}

function MacroPicker({ ticketId, onInsert, disabled }: MacroPickerProps) {
  const { t } = useTranslation("tickets");
  const { data: macros } = useMacros();
  const applyMacro = useApplyMacro();
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState("");
  const ref = useRef<HTMLDivElement>(null);

  // Close on outside click
  useEffect(() => {
    if (!open) return;
    function handle(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    }
    document.addEventListener("mousedown", handle);
    return () => document.removeEventListener("mousedown", handle);
  }, [open]);

  const filtered = (macros ?? []).filter(
    (m) =>
      m.title.toLowerCase().includes(search.toLowerCase()) ||
      (m.category ?? "").toLowerCase().includes(search.toLowerCase())
  );

  async function pick(macroId: number) {
    setOpen(false);
    try {
      const result = await applyMacro.mutateAsync({ macroId, ticketId });
      onInsert(result.rendered);
      toast.success(t("detail.macro_toast_inserted"));
      // TODO: apply result.actions (status/tag side-effects) if needed
    } catch (err) {
      toast.error(apiError(err, t("detail.macro_toast_error")));
    }
  }

  return (
    <div ref={ref} className="relative">
      <Button
        variant="ghost"
        size="sm"
        disabled={disabled || applyMacro.isPending}
        onClick={() => setOpen((v) => !v)}
      >
        {applyMacro.isPending ? (
          <Loader2 className="animate-spin" />
        ) : (
          <ChevronDown className="size-3.5" />
        )}
        {t("detail.macro_btn")}
      </Button>

      {open && (
        <div className="absolute bottom-full right-0 z-20 mb-1 w-64 rounded-md border border-border bg-popover shadow-lg">
          <div className="p-2">
            <input
              autoFocus
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder={t("detail.macro_search_placeholder")}
              className="w-full rounded-md border border-input bg-transparent px-2 py-1.5 text-sm focus:outline-none focus:ring-1 focus:ring-primary"
            />
          </div>
          <ul className="max-h-48 overflow-y-auto">
            {filtered.length === 0 ? (
              <li className="px-3 py-2 text-sm text-muted-foreground">{t("detail.macro_empty")}</li>
            ) : (
              filtered.map((m) => (
                <li key={m.id}>
                  <button
                    type="button"
                    className="w-full px-3 py-2 text-left text-sm hover:bg-accent"
                    onClick={() => pick(m.id)}
                  >
                    <div className="font-medium">{m.title}</div>
                    {m.category && (
                      <div className="font-mono text-[10px] uppercase text-muted-foreground">
                        {m.category}
                      </div>
                    )}
                  </button>
                </li>
              ))
            )}
          </ul>
        </div>
      )}
    </div>
  );
}

// ─── MergeDialog ──────────────────────────────────────────────────────────────

interface MergeDialogProps {
  ticketId: number;
  onSuccess: () => void;
  onClose: () => void;
}

function MergeDialog({ ticketId, onSuccess, onClose }: MergeDialogProps) {
  const { t } = useTranslation("tickets");
  const merge = useMergeTicket(ticketId);
  const [targetId, setTargetId] = useState("");

  async function handleMerge() {
    const tid = parseInt(targetId, 10);
    if (!tid || isNaN(tid)) return;
    try {
      await merge.mutateAsync(tid);
      toast.success(t("detail.merge_toast_success"));
      onSuccess();
    } catch (err) {
      toast.error(apiError(err, t("detail.merge_toast_error")));
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-background/80 backdrop-blur-sm">
      <div className="w-full max-w-sm rounded-lg border border-border bg-card p-6 shadow-xl">
        <div className="mb-4 flex items-center gap-2">
          <GitMerge className="size-5 text-primary" />
          <h2 className="text-lg font-semibold">{t("detail.merge_dialog_title")}</h2>
        </div>
        <p className="mb-4 text-sm text-muted-foreground">{t("detail.merge_dialog_desc")}</p>
        <input
          type="number"
          value={targetId}
          onChange={(e) => setTargetId(e.target.value)}
          placeholder={t("detail.merge_target_placeholder")}
          className="mb-4 h-9 w-full rounded-md border border-input bg-transparent px-3 text-sm focus:outline-none focus:ring-1 focus:ring-primary"
        />
        <div className="flex justify-end gap-2">
          <Button variant="secondary" size="sm" onClick={onClose} disabled={merge.isPending}>
            {t("detail.merge_btn_cancel")}
          </Button>
          <Button
            size="sm"
            variant="destructive"
            onClick={handleMerge}
            disabled={!targetId || merge.isPending}
          >
            {merge.isPending ? <Loader2 className="animate-spin" /> : <GitMerge className="size-4" />}
            {t("detail.merge_btn_confirm")}
          </Button>
        </div>
      </div>
    </div>
  );
}

// ─── PresenceBanner ───────────────────────────────────────────────────────────

function PresenceBanner({ presenceName }: { presenceName: string | null }) {
  const { t } = useTranslation("tickets");
  if (!presenceName) return null;
  return (
    <div className="flex items-center gap-2 rounded-md border border-amber-500/30 bg-amber-500/10 px-3 py-2 text-sm text-amber-300">
      <span className="size-2 rounded-full bg-amber-400 animate-pulse" />
      {t("detail.presence_viewing", { name: presenceName }) ||
        t("detail.presence_another")}
    </div>
  );
}

// ─── TicketDetailPage ──────────────────────────────────────────────────────────

export function TicketDetailPage() {
  const { id } = useParams();
  const ticketId = id ? Number(id) : undefined;
  const { t } = useTranslation("tickets");
  const { t: tCommon } = useTranslation("common");
  const { user } = useAuth();
  const qc = useQueryClient();
  const { data: ticket, isLoading } = useTicket(ticketId);
  const { data: messages } = useTicketMessages(ticketId);
  const update = useUpdateTicket(ticketId ?? 0);
  const addMessage = useAddMessage(ticketId ?? 0);
  const { data: aiSettings } = useAISettings();
  const suggest = useSuggestReply(ticketId ?? 0);
  const ref = useReveal(ticket?.id);

  const [draft, setDraft] = useState("");
  const [internal, setInternal] = useState(false);
  const [showMerge, setShowMerge] = useState(false);

  // AI suggest-reply result state
  const [aiResult, setAIResult] = useState<{
    confidence: number;
    needs_clarification: boolean;
    used_kb: boolean;
    sources: string[];
  } | null>(null);

  // Presence state: name of another agent currently viewing/typing
  const [presenceName, setPresenceName] = useState<string | null>(null);
  const presenceTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // WebSocket for real-time updates
  const wsRef = useRef<WebSocket | null>(null);

  const isTeam = user?.role !== "customer";
  const canSuggest =
    isTeam && !!aiSettings?.enabled && !!aiSettings?.suggest_replies;

  // ── WebSocket lifecycle ──────────────────────────────────────────────────

  const setupWebSocket = useCallback(() => {
    if (!ticketId || !isTeam) return;
    const token = tokenStore.access;
    if (!token) return;

    const scheme = window.location.protocol === "https:" ? "wss" : "ws";
    const wsUrl = `${scheme}://${window.location.host}/api/v1/ws/tickets/${ticketId}?token=${encodeURIComponent(token)}`;

    const ws = new WebSocket(wsUrl);
    wsRef.current = ws;

    ws.onmessage = (event) => {
      try {
        const frame = JSON.parse(event.data as string) as Record<string, unknown>;

        // Server broadcasts new messages as MessageResponse objects (have "ticket_id" + "content").
        if (frame.ticket_id && frame.content !== undefined) {
          // Append message if not already present — de-dupe by id.
          qc.setQueryData<typeof messages>(["ticket-messages", ticketId], (prev) => {
            if (!prev) return prev;
            const already = prev.some((m) => m.id === (frame.id as number));
            if (already) return prev;
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            return [...prev, frame as any];
          });
          return;
        }

        // Presence/typing frames from other agents.
        if (frame.type === "presence" || frame.type === "typing") {
          const name =
            typeof frame.display_name === "string"
              ? frame.display_name
              : null;
          if (name && name !== (user?.username ?? "")) {
            setPresenceName(name);
            // Clear presence banner after 8 s silence.
            if (presenceTimerRef.current) clearTimeout(presenceTimerRef.current);
            presenceTimerRef.current = setTimeout(() => setPresenceName(null), 8000);
          }
        }
      } catch {
        // Non-JSON ping frames — ignore.
      }
    };

    ws.onerror = () => {
      // Silently fail; polling remains as fallback.
    };

    ws.onclose = () => {
      wsRef.current = null;
    };
  }, [ticketId, isTeam, qc, user]);

  useEffect(() => {
    setupWebSocket();
    return () => {
      wsRef.current?.close();
      wsRef.current = null;
      if (presenceTimerRef.current) clearTimeout(presenceTimerRef.current);
    };
  }, [setupWebSocket]);

  // ── Send presence frame while agent is typing ───────────────────────────

  const presenceSentRef = useRef(0);
  function onDraftChange(value: string) {
    setDraft(value);
    // Throttle: send at most once every 3 s.
    const now = Date.now();
    if (wsRef.current?.readyState === WebSocket.OPEN && now - presenceSentRef.current > 3000) {
      presenceSentRef.current = now;
      wsRef.current.send(
        JSON.stringify({
          type: "typing",
          display_name: user?.username ?? user?.email ?? "Agent",
        })
      );
    }
  }

  // Send a viewing presence frame on mount (once).
  useEffect(() => {
    if (!ticketId || !isTeam) return;
    const timer = setTimeout(() => {
      if (wsRef.current?.readyState === WebSocket.OPEN) {
        wsRef.current.send(
          JSON.stringify({
            type: "presence",
            display_name: user?.username ?? user?.email ?? "Agent",
          })
        );
      }
    }, 1000);
    return () => clearTimeout(timer);
  }, [ticketId, isTeam, user]);

  // ── AI suggest reply ────────────────────────────────────────────────────

  async function suggestReply() {
    try {
      const result = await suggest.mutateAsync();
      setDraft(result.reply);
      setAIResult({
        confidence: result.confidence,
        needs_clarification: result.needs_clarification,
        used_kb: result.used_kb,
        sources: result.sources,
      });
    } catch (err) {
      toast.error(apiError(err, t("detail.suggest_error")));
    }
  }

  // ── Macro insert ────────────────────────────────────────────────────────

  function onMacroInsert(text: string) {
    setDraft((prev) => (prev ? prev + "\n\n" + text : text));
  }

  // ── Status/priority patch ───────────────────────────────────────────────

  async function patch(field: "status" | "priority", value: string) {
    try {
      await update.mutateAsync({ [field]: value });
      toast.success(t("detail.toast_updated_field", { field }));
    } catch (err) {
      toast.error(apiError(err));
    }
  }

  // ── Send reply ──────────────────────────────────────────────────────────

  async function send() {
    if (!draft.trim()) return;
    try {
      await addMessage.mutateAsync({ content: draft.trim(), is_internal: internal });
      setDraft("");
      setAIResult(null);
    } catch (err) {
      toast.error(apiError(err, t("detail.toast_post_error")));
    }
  }

  // ── Merge success ───────────────────────────────────────────────────────

  function onMergeSuccess() {
    setShowMerge(false);
    qc.invalidateQueries({ queryKey: ["ticket", ticketId] });
  }

  // ── Loading / not-found states ──────────────────────────────────────────

  if (isLoading) {
    return (
      <div className="w-full space-y-4">
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-48 w-full" />
      </div>
    );
  }

  if (!ticket) {
    return (
      <div className="w-full py-20 text-center text-muted-foreground">
        {t("detail.not_found")}
        <div className="mt-4">
          <Button variant="secondary" asChild>
            <Link to="/tickets">
              <ArrowLeft /> {t("detail.btn_back_to_tickets")}
            </Link>
          </Button>
        </div>
      </div>
    );
  }

  const isMerged = ticket.status === "merged";

  return (
    <div ref={ref} className="w-full">
      {/* Merge dialog */}
      {showMerge && (
        <MergeDialog
          ticketId={ticket.id}
          onSuccess={onMergeSuccess}
          onClose={() => setShowMerge(false)}
        />
      )}

      <Link
        to="/tickets"
        className="mb-4 inline-flex items-center gap-1.5 text-sm text-muted-foreground transition-colors hover:text-foreground"
      >
        <ArrowLeft className="size-4" /> {t("detail.back_link")}
      </Link>

      <div data-reveal className="mb-6 flex items-start justify-between gap-3">
        <div>
          <div className="font-mono text-xs text-primary/80">{ticket.ticket_number}</div>
          <h1 className="mt-1 text-2xl">{ticket.title}</h1>
        </div>
        {isTeam && !isMerged && (
          <Button
            variant="secondary"
            size="sm"
            onClick={() => setShowMerge(true)}
            className="shrink-0"
          >
            <GitMerge className="size-4" />
            {t("detail.merge_btn")}
          </Button>
        )}
      </div>

      {/* Merged banner */}
      {isMerged && ticket.merged_into_id && (
        <div className="mb-4 flex items-center gap-2 rounded-md border border-primary/30 bg-primary/10 px-4 py-3 text-sm text-primary">
          <GitMerge className="size-4 shrink-0" />
          {t("detail.merged_banner", { id: ticket.merged_into_id })}
        </div>
      )}

      {/* Presence banner */}
      <div className="mb-3">
        <PresenceBanner presenceName={presenceName} />
      </div>

      <div className="grid gap-6 lg:grid-cols-[1fr_18rem]">
        {/* Conversation column */}
        <div className="space-y-5">
          <Card data-reveal className="p-5">
            <Label>{t("detail.section_description")}</Label>
            <p className="mt-2 whitespace-pre-wrap text-sm leading-relaxed text-foreground/90">
              {ticket.description || t("detail.no_description")}
            </p>
          </Card>

          <div data-reveal>
            <h2 className="mb-3 text-sm font-semibold">
              {t("detail.section_conversation")}{" "}
              <span className="font-mono text-xs text-muted-foreground">
                ({messages?.length ?? 0})
              </span>
            </h2>
            <div className="space-y-3">
              {messages && messages.length > 0 ? (
                messages.map((m) => (
                  <Card
                    key={m.id}
                    className={m.is_internal ? "border-primary/30 bg-primary/5 p-4" : "p-4"}
                  >
                    <div className="mb-1.5 flex items-center gap-2 text-xs text-muted-foreground">
                      <Avatar className="size-5">
                        <AvatarFallback className="text-[9px]">
                          {m.is_from_ai
                            ? "AI"
                            : m.author_name
                              ? m.author_name
                                  .split(" ")
                                  .map((p) => p[0])
                                  .join("")
                                  .slice(0, 2)
                                  .toUpperCase()
                              : `U${m.user_id}`}
                        </AvatarFallback>
                      </Avatar>
                      <span className="font-medium text-foreground/80">
                        {m.is_from_ai
                          ? t("detail.msg_sender_ai")
                          : m.author_name || t("detail.msg_sender_user", { userId: m.user_id })}
                      </span>
                      {!m.is_from_ai && m.author_role && (
                        <span className="font-mono text-[10px] uppercase tracking-wider text-muted-foreground/70">
                          {m.author_role === "customer"
                            ? t("detail.msg_role_customer")
                            : t("detail.msg_role_team")}
                        </span>
                      )}
                      {m.is_from_ai && <Bot className="size-3" />}
                      {m.is_internal && (
                        <span className="inline-flex items-center gap-1 font-mono text-[10px] uppercase tracking-wider text-primary">
                          <Lock className="size-3" /> {t("detail.msg_internal_badge")}
                        </span>
                      )}
                      <span className="ml-auto font-mono">{relativeTime(m.created_at)}</span>
                    </div>
                    {/* Message body with @mention highlighting */}
                    <p className="whitespace-pre-wrap text-sm leading-relaxed">
                      {renderWithMentions(m.content)}
                    </p>
                  </Card>
                ))
              ) : (
                <p className="text-sm text-muted-foreground">{t("detail.no_messages")}</p>
              )}
            </div>

            {/* Reply box — hidden when ticket is merged */}
            {!isMerged && (
              <Card className="mt-4 p-4">
                {/* AI confidence banner (shown after suggest-reply) */}
                {aiResult && (
                  <div className="mb-3 space-y-1.5">
                    <div className="flex flex-wrap items-center gap-2">
                      {aiResult.confidence > 0 && (
                        <span className="inline-flex items-center gap-1 rounded-full border border-primary/30 bg-primary/10 px-2 py-0.5 font-mono text-[11px] text-primary">
                          <Sparkles className="size-3" />
                          {t("detail.ai_confidence", {
                            pct: Math.round(aiResult.confidence * 100),
                          })}
                        </span>
                      )}
                      {aiResult.used_kb && (
                        <span className="inline-flex items-center gap-1 rounded-full border border-emerald-500/30 bg-emerald-500/10 px-2 py-0.5 font-mono text-[11px] text-emerald-300">
                          KB
                        </span>
                      )}
                    </div>
                    {aiResult.needs_clarification && (
                      <div className="flex items-start gap-1.5 rounded-md border border-amber-500/30 bg-amber-500/10 px-2 py-1.5 text-xs text-amber-300">
                        <AlertTriangle className="mt-0.5 size-3.5 shrink-0" />
                        {t("detail.ai_needs_clarification")}
                      </div>
                    )}
                    {aiResult.sources.length > 0 && (
                      <div className="text-xs text-muted-foreground">
                        <span className="font-medium">{t("detail.ai_sources_label")}</span>{" "}
                        {aiResult.sources.join(", ")}
                      </div>
                    )}
                  </div>
                )}

                <Textarea
                  value={draft}
                  onChange={(e) => onDraftChange(e.target.value)}
                  placeholder={t("detail.reply_placeholder")}
                  className="min-h-24 border-0 bg-transparent p-0 shadow-none focus-visible:ring-0"
                />
                <Separator className="my-3" />
                <div className="flex items-center justify-between">
                  <div className="flex flex-col gap-1">
                    <label className="flex cursor-pointer items-center gap-2 text-xs text-muted-foreground">
                      <input
                        type="checkbox"
                        checked={internal}
                        onChange={(e) => setInternal(e.target.checked)}
                        className="accent-[var(--color-primary)]"
                      />
                      {t("detail.internal_note_label")}
                    </label>
                    {internal && (
                      <p className="text-[11px] text-muted-foreground/70">
                        {t("detail.internal_note_mention_hint")}
                      </p>
                    )}
                  </div>
                  <div className="flex items-center gap-2">
                    {/* Macro picker — team only */}
                    {isTeam && (
                      <MacroPicker
                        ticketId={ticket.id}
                        onInsert={onMacroInsert}
                        disabled={addMessage.isPending}
                      />
                    )}
                    {canSuggest && (
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={suggestReply}
                        disabled={suggest.isPending}
                        title={t("detail.suggest_hint")}
                      >
                        <Sparkles className={suggest.isPending ? "animate-pulse" : ""} />
                        {suggest.isPending ? t("detail.suggesting") : t("detail.suggest_btn")}
                      </Button>
                    )}
                    <Button
                      size="sm"
                      onClick={send}
                      disabled={addMessage.isPending || !draft.trim()}
                    >
                      <Send />{" "}
                      {addMessage.isPending ? t("detail.btn_sending") : t("detail.btn_send")}
                    </Button>
                  </div>
                </div>
              </Card>
            )}
          </div>

          <AttachmentsCard ticketId={ticket.id} />

          {/* Linked tickets panel */}
          {isTeam && <LinkedTicketsCard ticketId={ticket.id} />}

          <ActivityCard ticketId={ticket.id} />
        </div>

        {/* Meta sidebar */}
        <aside data-reveal className="space-y-4">
          <Card className="p-5">
            <div className="space-y-3">
              <div>
                <Label>{t("detail.meta.label_status")}</Label>
                <Select
                  value={ticket.status}
                  onValueChange={(v) => patch("status", v)}
                  disabled={isMerged}
                >
                  <SelectTrigger className="mt-1.5">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {STATUS_OPTIONS.map((s) => (
                      <SelectItem key={s} value={s}>
                        {tCommon(`enums.ticket_status.${s}`)}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div>
                <Label>{t("detail.meta.label_priority")}</Label>
                <Select
                  value={ticket.priority}
                  onValueChange={(v) => patch("priority", v)}
                  disabled={isMerged}
                >
                  <SelectTrigger className="mt-1.5">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {PRIORITY_OPTIONS.map((p) => (
                      <SelectItem key={p} value={p}>
                        {tCommon(`enums.priority.${p}`)}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            </div>
          </Card>

          <Card className="p-5">
            <MetaRow
              label={t("detail.meta.label_severity")}
              value={<PriorityBadge priority={ticket.severity as never} />}
            />
            <Separator />
            <MetaRow
              label={t("detail.meta.label_customer")}
              value={
                ticket.customer_name ||
                (ticket.customer_id ? `#${ticket.customer_id}` : "—")
              }
            />
            <MetaRow
              label={t("detail.meta.label_requester")}
              value={ticket.requester_name || "—"}
            />
            <MetaRow
              label={t("detail.meta.label_email")}
              value={
                <span className="font-mono text-xs">{ticket.requester_email || "—"}</span>
              }
            />
            <Separator />
            <MetaRow
              label={t("detail.meta.label_assignee")}
              value={
                user?.role !== "customer" ? (
                  <AssigneeControl ticket={ticket} />
                ) : ticket.assigned_user ? (
                  `${ticket.assigned_user.first_name} ${ticket.assigned_user.last_name}`.trim() ||
                  ticket.assigned_user.username
                ) : ticket.assigned_to ? (
                  `#${ticket.assigned_to}`
                ) : (
                  t("detail.meta.unassigned")
                )
              }
            />
            <MetaRow
              label={t("detail.meta.label_created")}
              value={relativeTime(ticket.created_at)}
            />
            <MetaRow
              label={t("detail.meta.label_due")}
              value={relativeTime(ticket.due_date)}
            />
          </Card>

          <SlaCard ticketId={ticket.id} />
        </aside>
      </div>
    </div>
  );
}
