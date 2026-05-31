import { useState } from "react";
import { Link, useParams } from "react-router-dom";
import { ArrowLeft, Send, Lock, Bot } from "lucide-react";
import { toast } from "sonner";
import {
  useTicket,
  useTicketMessages,
  useUpdateTicket,
  useAddMessage,
} from "@/features/tickets/api";
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

export function TicketDetailPage() {
  const { id } = useParams();
  const ticketId = id ? Number(id) : undefined;
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
      toast.success(`Updated ${field}`);
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
      toast.error(apiError(err, "Could not post message"));
    }
  }

  if (isLoading) {
    return (
      <div className="mx-auto max-w-5xl space-y-4">
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-48 w-full" />
      </div>
    );
  }

  if (!ticket) {
    return (
      <div className="mx-auto max-w-5xl py-20 text-center text-muted-foreground">
        Ticket not found.
        <div className="mt-4">
          <Button variant="secondary" asChild>
            <Link to="/tickets">
              <ArrowLeft /> Back to tickets
            </Link>
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div ref={ref} className="mx-auto max-w-5xl">
      <Link
        to="/tickets"
        className="mb-4 inline-flex items-center gap-1.5 text-sm text-muted-foreground transition-colors hover:text-foreground"
      >
        <ArrowLeft className="size-4" /> Tickets
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
            <Label>Description</Label>
            <p className="mt-2 whitespace-pre-wrap text-sm leading-relaxed text-foreground/90">
              {ticket.description || "No description provided."}
            </p>
          </Card>

          <div data-reveal>
            <h2 className="mb-3 text-sm font-semibold">
              Conversation{" "}
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
                          {m.is_from_ai ? "AI" : `U${m.user_id}`}
                        </AvatarFallback>
                      </Avatar>
                      <span className="font-medium text-foreground/80">
                        {m.is_from_ai ? "Assistant" : `User #${m.user_id}`}
                      </span>
                      {m.is_from_ai && <Bot className="size-3" />}
                      {m.is_internal && (
                        <span className="inline-flex items-center gap-1 font-mono text-[10px] uppercase tracking-wider text-primary">
                          <Lock className="size-3" /> internal
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
                <p className="text-sm text-muted-foreground">No messages yet.</p>
              )}
            </div>

            {/* Reply box */}
            <Card className="mt-4 p-4">
              <Textarea
                value={draft}
                onChange={(e) => setDraft(e.target.value)}
                placeholder="Write a reply…"
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
                  Internal note
                </label>
                <Button
                  size="sm"
                  onClick={send}
                  disabled={addMessage.isPending || !draft.trim()}
                >
                  <Send /> {addMessage.isPending ? "Sending…" : "Send"}
                </Button>
              </div>
            </Card>
          </div>
        </div>

        {/* Meta sidebar */}
        <aside data-reveal className="space-y-4">
          <Card className="p-5">
            <div className="space-y-3">
              <div>
                <Label>Status</Label>
                <Select
                  value={ticket.status}
                  onValueChange={(v) => patch("status", v)}
                >
                  <SelectTrigger className="mt-1.5">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {STATUS_OPTIONS.map((s) => (
                      <SelectItem key={s} value={s} className="capitalize">
                        {s.replace("_", " ")}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div>
                <Label>Priority</Label>
                <Select
                  value={ticket.priority}
                  onValueChange={(v) => patch("priority", v)}
                >
                  <SelectTrigger className="mt-1.5">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {PRIORITY_OPTIONS.map((p) => (
                      <SelectItem key={p} value={p} className="capitalize">
                        {p}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            </div>
          </Card>

          <Card className="p-5">
            <MetaRow
              label="Severity"
              value={<PriorityBadge priority={ticket.severity as never} />}
            />
            <Separator />
            <MetaRow
              label="Customer"
              value={ticket.customer_name || (ticket.customer_id ? `#${ticket.customer_id}` : "—")}
            />
            <MetaRow label="Requester" value={ticket.requester_name || "—"} />
            <MetaRow
              label="Email"
              value={
                <span className="font-mono text-xs">
                  {ticket.requester_email || "—"}
                </span>
              }
            />
            <Separator />
            <MetaRow
              label="Assignee"
              value={ticket.assigned_to ? `#${ticket.assigned_to}` : "Unassigned"}
            />
            <MetaRow label="Created" value={relativeTime(ticket.created_at)} />
            <MetaRow label="Due" value={relativeTime(ticket.due_date)} />
          </Card>
        </aside>
      </div>
    </div>
  );
}
