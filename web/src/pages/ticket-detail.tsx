import { useRef, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { useTranslation } from "react-i18next";
import {
  ArrowLeft,
  Send,
  Lock,
  Bot,
  Download,
  Paperclip,
  Loader2,
  Timer,
  History,
  CircleDot,
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
} from "@/features/tickets/api";
import { useUsers } from "@/features/users/api";
import { useAuth } from "@/lib/auth";
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

function formatBytes(n: number): string {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  return `${(n / (1024 * 1024)).toFixed(1)} MB`;
}

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
        <input
          ref={fileInput}
          type="file"
          className="hidden"
          onChange={onPick}
        />
        <Button
          size="sm"
          variant="secondary"
          onClick={() => fileInput.current?.click()}
          disabled={upload.isPending}
        >
          {upload.isPending ? (
            <Loader2 className="animate-spin" />
          ) : (
            <Paperclip />
          )}
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

function userDisplayName(u: Pick<UserInfo, "first_name" | "last_name" | "username">) {
  return `${u.first_name} ${u.last_name}`.trim() || u.username;
}

function AssigneeControl({ ticket }: { ticket: Ticket }) {
  const { t } = useTranslation("tickets");
  const assign = useAssignTicket(ticket.id);
  // Assignable users are non-customer roles (the support team).
  const { data: usersPage } = useUsers({ page: 1, page_size: 100 });
  const team = (usersPage?.items ?? []).filter((u) => u.role !== "customer");

  // Ensure the current assignee is always selectable even if outside the page.
  const options = [...team];
  if (
    ticket.assigned_user &&
    !options.some((u) => u.id === ticket.assigned_user!.id)
  ) {
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

// fmtMinutes renders an SLA target in minutes as a compact human string.
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

/** SlaCard shows which SLA policy governs the ticket — the matched rule/template,
 *  its response & resolution targets, and the current SLA status. */
function SlaCard({ ticketId }: { ticketId: number }) {
  const { t } = useTranslation("tickets");
  const { data: sla, isLoading } = useTicketSLA(ticketId);
  if (isLoading || !sla) return null;
  const statusClass =
    SLA_STATUS_STYLE[sla.sla_status] ??
    "border-border bg-muted text-muted-foreground";
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

// ActivityCard renders the ticket's operation history as a timeline.
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

export function TicketDetailPage() {
  const { id } = useParams();
  const ticketId = id ? Number(id) : undefined;
  const { t } = useTranslation("tickets");
  const { t: tCommon } = useTranslation("common");
  const { user } = useAuth();
  const { data: ticket, isLoading } = useTicket(ticketId);
  const { data: messages } = useTicketMessages(ticketId);
  const update = useUpdateTicket(ticketId ?? 0);
  const addMessage = useAddMessage(ticketId ?? 0);
  const [draft, setDraft] = useState("");
  const [internal, setInternal] = useState(false);
  const ref = useReveal(ticket?.id);

  async function patch(field: "status" | "priority", value: string) {
    try {
      await update.mutateAsync({ [field]: value });
      toast.success(t("detail.toast_updated_field", { field }));
    } catch (err) {
      toast.error(apiError(err));
    }
  }

  async function send() {
    if (!draft.trim()) return;
    try {
      await addMessage.mutateAsync({ content: draft.trim(), is_internal: internal });
      setDraft("");
    } catch (err) {
      toast.error(apiError(err, t("detail.toast_post_error")));
    }
  }

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

  return (
    <div ref={ref} className="w-full">
      <Link
        to="/tickets"
        className="mb-4 inline-flex items-center gap-1.5 text-sm text-muted-foreground transition-colors hover:text-foreground"
      >
        <ArrowLeft className="size-4" /> {t("detail.back_link")}
      </Link>

      <div data-reveal className="mb-6 flex items-start gap-3">
        <div>
          <div className="font-mono text-xs text-primary/80">
            {ticket.ticket_number}
          </div>
          <h1 className="mt-1 text-2xl">{ticket.title}</h1>
        </div>
      </div>

      <div className="grid gap-6 lg:grid-cols-[1fr_18rem]">
        {/* Conversation */}
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
                    className={
                      m.is_internal
                        ? "border-primary/30 bg-primary/5 p-4"
                        : "p-4"
                    }
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
                      <span className="ml-auto font-mono">
                        {relativeTime(m.created_at)}
                      </span>
                    </div>
                    <p className="whitespace-pre-wrap text-sm leading-relaxed">
                      {m.content}
                    </p>
                  </Card>
                ))
              ) : (
                <p className="text-sm text-muted-foreground">{t("detail.no_messages")}</p>
              )}
            </div>

            {/* Reply box */}
            <Card className="mt-4 p-4">
              <Textarea
                value={draft}
                onChange={(e) => setDraft(e.target.value)}
                placeholder={t("detail.reply_placeholder")}
                className="min-h-24 border-0 bg-transparent p-0 shadow-none focus-visible:ring-0"
              />
              <Separator className="my-3" />
              <div className="flex items-center justify-between">
                <label className="flex cursor-pointer items-center gap-2 text-xs text-muted-foreground">
                  <input
                    type="checkbox"
                    checked={internal}
                    onChange={(e) => setInternal(e.target.checked)}
                    className="accent-[var(--color-primary)]"
                  />
                  {t("detail.internal_note_label")}
                </label>
                <Button
                  size="sm"
                  onClick={send}
                  disabled={addMessage.isPending || !draft.trim()}
                >
                  <Send /> {addMessage.isPending ? t("detail.btn_sending") : t("detail.btn_send")}
                </Button>
              </div>
            </Card>
          </div>

          <AttachmentsCard ticketId={ticket.id} />

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
              value={ticket.customer_name || (ticket.customer_id ? `#${ticket.customer_id}` : "—")}
            />
            <MetaRow label={t("detail.meta.label_requester")} value={ticket.requester_name || "—"} />
            <MetaRow
              label={t("detail.meta.label_email")}
              value={
                <span className="font-mono text-xs">
                  {ticket.requester_email || "—"}
                </span>
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
            <MetaRow label={t("detail.meta.label_created")} value={relativeTime(ticket.created_at)} />
            <MetaRow label={t("detail.meta.label_due")} value={relativeTime(ticket.due_date)} />
          </Card>

          <SlaCard ticketId={ticket.id} />
        </aside>
      </div>
    </div>
  );
}
