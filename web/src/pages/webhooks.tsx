import { useState } from "react";
import { Webhook, Plus, Trash2, Copy, Check, X, FlaskConical, BookOpen } from "lucide-react";
import { toast } from "sonner";
import { useTranslation } from "react-i18next";
import {
  useWebhooks,
  useCreateWebhook,
  useDeleteWebhook,
  useWebhookDeliveries,
  useTestWebhook,
  ALL_WEBHOOK_EVENTS,
  type WebhookDelivery,
} from "@/features/webhooks/api";
import { apiError } from "@/lib/api";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/misc";
import { useReveal } from "@/lib/use-reveal";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

/** Format a unix timestamp (seconds) to a locale string, or "—" if null. */
function fmtTime(ts: number | null | undefined): string {
  if (!ts) return "—";
  return new Date(ts * 1000).toLocaleString();
}

// --- One-time secret reveal banner ----------------------------------------

function SecretRevealBanner({
  secret,
  onClose,
}: {
  secret: string;
  onClose: () => void;
}) {
  const { t } = useTranslation("webhooks");
  const [copied, setCopied] = useState(false);

  async function handleCopy() {
    try {
      await navigator.clipboard.writeText(secret);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      toast.error(t("reveal.copyFailed"));
    }
  }

  return (
    <div className="rounded-md border border-amber-400/60 bg-amber-50/60 p-4 dark:border-amber-500/40 dark:bg-amber-950/30">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0 flex-1">
          <p className="text-sm font-semibold text-amber-800 dark:text-amber-300">
            {t("reveal.title")}
          </p>
          <p className="mt-0.5 text-xs text-amber-700/80 dark:text-amber-400/80">
            {t("reveal.subtitle")}
          </p>
          <div className="mt-3 flex items-center gap-2">
            <code className="flex-1 truncate rounded-md bg-amber-100/70 px-3 py-2 font-mono text-xs text-amber-900 dark:bg-amber-900/30 dark:text-amber-200">
              {secret}
            </code>
            <Button
              size="sm"
              variant="outline"
              className="shrink-0 border-amber-400/70 dark:border-amber-500/50"
              onClick={handleCopy}
            >
              {copied ? (
                <Check className="size-3.5 text-green-600" />
              ) : (
                <Copy className="size-3.5" />
              )}
              {copied ? t("reveal.copied") : t("reveal.copy")}
            </Button>
          </div>
        </div>
        <button
          type="button"
          onClick={onClose}
          aria-label={t("reveal.close")}
          className="mt-0.5 shrink-0 rounded-md p-1 text-amber-600 transition-colors hover:bg-amber-100 dark:text-amber-400 dark:hover:bg-amber-900/40"
        >
          <X className="size-4" />
        </button>
      </div>
    </div>
  );
}

// --- Create webhook dialog ------------------------------------------------

function CreateWebhookDialog({ onCreated }: { onCreated: (secret: string) => void }) {
  const { t } = useTranslation("webhooks");
  const [open, setOpen] = useState(false);
  const [name, setName] = useState("");
  const [url, setUrl] = useState("");
  const [selectedEvents, setSelectedEvents] = useState<Set<string>>(new Set());
  const create = useCreateWebhook();

  function handleOpen(v: boolean) {
    if (v) {
      setName("");
      setUrl("");
      setSelectedEvents(new Set());
    }
    setOpen(v);
  }

  function toggleEvent(evt: string) {
    setSelectedEvents((prev) => {
      const next = new Set(prev);
      if (next.has(evt)) {
        next.delete(evt);
      } else {
        next.add(evt);
      }
      return next;
    });
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!name.trim()) {
      toast.error(t("validation.nameRequired"));
      return;
    }
    if (!url.trim()) {
      toast.error(t("validation.urlRequired"));
      return;
    }
    if (selectedEvents.size === 0) {
      toast.error(t("validation.eventsRequired"));
      return;
    }
    try {
      const result = await create.mutateAsync({
        name: name.trim(),
        url: url.trim(),
        events: Array.from(selectedEvents),
      });
      toast.success(t("toast.created"));
      setOpen(false);
      onCreated(result.secret);
    } catch (err) {
      toast.error(apiError(err, t("toast.createFailed")));
    }
  }

  return (
    <Dialog open={open} onOpenChange={handleOpen}>
      <Button onClick={() => handleOpen(true)}>
        <Plus /> {t("newWebhook")}
      </Button>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>{t("form.title")}</DialogTitle>
          <DialogDescription>{t("form.description")}</DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-1.5">
            <Label htmlFor="wh-name">{t("form.name")}</Label>
            <Input
              id="wh-name"
              placeholder={t("form.namePlaceholder")}
              value={name}
              onChange={(e) => setName(e.target.value)}
              maxLength={100}
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="wh-url">{t("form.url")}</Label>
            <Input
              id="wh-url"
              type="url"
              placeholder="https://example.com/webhook"
              value={url}
              onChange={(e) => setUrl(e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label>{t("form.events")}</Label>
            <div className="space-y-2 rounded-md border border-border p-3">
              {ALL_WEBHOOK_EVENTS.map((evt) => (
                <label key={evt} className="flex cursor-pointer items-center gap-2.5">
                  <input
                    type="checkbox"
                    checked={selectedEvents.has(evt)}
                    onChange={() => toggleEvent(evt)}
                    className="size-4 accent-primary"
                  />
                  <span className="font-mono text-xs text-foreground">{evt}</span>
                </label>
              ))}
            </div>
          </div>
          <DialogFooter>
            <Button
              type="button"
              variant="ghost"
              onClick={() => handleOpen(false)}
              disabled={create.isPending}
            >
              {t("actions.cancel", { ns: "common" })}
            </Button>
            <Button type="submit" disabled={create.isPending}>
              {create.isPending ? t("form.creating") : t("form.submit")}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

// --- Deliveries dialog -----------------------------------------------------

function DeliveriesDialog({
  webhookId,
  webhookName,
  open,
  onClose,
}: {
  webhookId: number | null;
  webhookName: string;
  open: boolean;
  onClose: () => void;
}) {
  const { t } = useTranslation("webhooks");
  const { data: deliveries, isLoading } = useWebhookDeliveries(open ? webhookId : null);

  return (
    <Dialog open={open} onOpenChange={(v) => !v && onClose()}>
      <DialogContent className="max-w-3xl">
        <DialogHeader>
          <DialogTitle>{t("deliveries.title", { name: webhookName })}</DialogTitle>
          <DialogDescription>{t("deliveries.description")}</DialogDescription>
        </DialogHeader>
        <div className="max-h-[60vh] overflow-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border text-left font-mono text-[11px] uppercase tracking-wider text-muted-foreground">
                <th className="px-3 py-2 font-medium">{t("deliveries.table.event")}</th>
                <th className="px-3 py-2 font-medium">{t("deliveries.table.status")}</th>
                <th className="px-3 py-2 font-medium">{t("deliveries.table.code")}</th>
                <th className="px-3 py-2 font-medium">{t("deliveries.table.attempts")}</th>
                <th className="px-3 py-2 font-medium">{t("deliveries.table.lastAttempt")}</th>
                <th className="px-3 py-2 font-medium">{t("deliveries.table.error")}</th>
              </tr>
            </thead>
            <tbody>
              {isLoading ? (
                Array.from({ length: 4 }).map((_, i) => (
                  <tr key={i} className="border-b border-border/60">
                    {Array.from({ length: 6 }).map((__, j) => (
                      <td key={j} className="px-3 py-2.5">
                        <Skeleton className="h-4 w-full" />
                      </td>
                    ))}
                  </tr>
                ))
              ) : deliveries && deliveries.length > 0 ? (
                deliveries.map((d: WebhookDelivery) => (
                  <tr
                    key={d.id}
                    className="border-b border-border/60 last:border-0 hover:bg-accent/50"
                  >
                    <td className="px-3 py-2.5 font-mono text-xs text-muted-foreground">
                      {d.event_type}
                    </td>
                    <td className="px-3 py-2.5">
                      <Badge
                        tone={
                          d.status === "success"
                            ? "green"
                            : d.status === "failed"
                            ? "red"
                            : "slate"
                        }
                      >
                        {d.status}
                      </Badge>
                    </td>
                    <td className="px-3 py-2.5 font-mono text-xs text-muted-foreground">
                      {d.status_code ?? "—"}
                    </td>
                    <td className="px-3 py-2.5 text-xs text-muted-foreground">
                      {d.attempts}
                    </td>
                    <td className="px-3 py-2.5 text-xs text-muted-foreground">
                      {fmtTime(d.last_attempt_at)}
                    </td>
                    <td className="max-w-[200px] truncate px-3 py-2.5 font-mono text-xs text-muted-foreground">
                      {d.error ?? "—"}
                    </td>
                  </tr>
                ))
              ) : (
                <tr>
                  <td colSpan={6} className="px-3 py-10 text-center text-sm text-muted-foreground">
                    {t("deliveries.empty")}
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
        <DialogFooter>
          <Button variant="ghost" onClick={onClose}>
            {t("actions.cancel", { ns: "common" })}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// --- Page ------------------------------------------------------------------

export function WebhooksPage() {
  const { t } = useTranslation("webhooks");
  const { data: webhooks, isLoading } = useWebhooks();
  const deleteWebhook = useDeleteWebhook();
  const testWebhook = useTestWebhook();
  const ref = useReveal<HTMLDivElement>();

  const [revealedSecret, setRevealedSecret] = useState<string | null>(null);
  const [toDelete, setToDelete] = useState<{ id: number; name: string } | null>(null);
  const [deliveriesFor, setDeliveriesFor] = useState<{ id: number; name: string } | null>(null);

  async function confirmDelete() {
    if (!toDelete) return;
    try {
      await deleteWebhook.mutateAsync(toDelete.id);
      toast.success(t("toast.deleted"));
      setToDelete(null);
    } catch (err) {
      toast.error(apiError(err, t("toast.deleteFailed")));
    }
  }

  async function handleTest(id: number) {
    try {
      await testWebhook.mutateAsync(id);
      toast.success(t("toast.testQueued"));
    } catch (err) {
      toast.error(apiError(err, t("toast.testFailed")));
    }
  }

  return (
    <div ref={ref} className="w-full">
      <div data-reveal className="mb-6 flex flex-wrap items-end justify-between gap-4">
        <div>
          <div className="font-mono text-xs uppercase tracking-[0.25em] text-muted-foreground">
            {t("page.eyebrow")}
          </div>
          <h1 className="mt-1 text-3xl">{t("page.title")}</h1>
        </div>
        <CreateWebhookDialog
          onCreated={(secret) => {
            setRevealedSecret(secret);
          }}
        />
      </div>

      {revealedSecret && (
        <div data-reveal className="mb-6">
          <SecretRevealBanner
            secret={revealedSecret}
            onClose={() => setRevealedSecret(null)}
          />
        </div>
      )}

      <Card data-reveal className="overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border text-left font-mono text-[11px] uppercase tracking-wider text-muted-foreground">
              <th className="px-4 py-3 font-medium">{t("table.name")}</th>
              <th className="px-4 py-3 font-medium">{t("table.url")}</th>
              <th className="px-4 py-3 font-medium">{t("table.events")}</th>
              <th className="px-4 py-3 font-medium">{t("table.status")}</th>
              <th className="px-4 py-3" />
            </tr>
          </thead>
          <tbody>
            {isLoading ? (
              Array.from({ length: 3 }).map((_, i) => (
                <tr key={i} className="border-b border-border/60">
                  {Array.from({ length: 5 }).map((__, j) => (
                    <td key={j} className="px-4 py-3.5">
                      <Skeleton className="h-4 w-full" />
                    </td>
                  ))}
                </tr>
              ))
            ) : webhooks && webhooks.length > 0 ? (
              webhooks.map((wh) => (
                <tr
                  key={wh.id}
                  className="border-b border-border/60 transition-colors last:border-0 hover:bg-accent/50"
                >
                  <td className="px-4 py-3.5 font-medium text-foreground">
                    {wh.name}
                  </td>
                  <td className="max-w-[220px] truncate px-4 py-3.5 font-mono text-xs text-muted-foreground">
                    {wh.url}
                  </td>
                  <td className="px-4 py-3.5">
                    <div className="flex flex-wrap gap-1">
                      {(wh.events ?? []).map((evt) => (
                        <Badge key={evt} tone="slate" className="font-mono text-[10px]">
                          {evt}
                        </Badge>
                      ))}
                    </div>
                  </td>
                  <td className="px-4 py-3.5">
                    <Badge tone={wh.active ? "green" : "slate"}>
                      {wh.active ? t("badges.active") : t("badges.inactive")}
                    </Badge>
                  </td>
                  <td className="px-2 py-3.5">
                    <div className="flex items-center justify-end gap-1">
                      <Button
                        variant="ghost"
                        size="sm"
                        title={t("actions.testTitle")}
                        disabled={testWebhook.isPending}
                        onClick={() => handleTest(wh.id)}
                      >
                        <FlaskConical className="size-4" />
                        {t("actions.test")}
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        title={t("actions.deliveriesTitle")}
                        onClick={() => setDeliveriesFor({ id: wh.id, name: wh.name })}
                      >
                        <BookOpen className="size-4" />
                        {t("actions.deliveries")}
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="text-destructive hover:text-destructive"
                        title={t("actions.deleteTitle")}
                        onClick={() => setToDelete({ id: wh.id, name: wh.name })}
                      >
                        <Trash2 className="size-4" />
                        {t("actions.delete")}
                      </Button>
                    </div>
                  </td>
                </tr>
              ))
            ) : (
              <tr>
                <td colSpan={5} className="px-4 py-16 text-center">
                  <Webhook className="mx-auto size-8 text-muted-foreground/40" />
                  <p className="mt-3 text-sm text-muted-foreground">
                    {t("empty")}
                  </p>
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </Card>

      {/* Delete confirmation dialog */}
      <Dialog
        open={toDelete != null}
        onOpenChange={(v) => !v && setToDelete(null)}
      >
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>{t("deleteDialog.title")}</DialogTitle>
            <DialogDescription>
              {t("deleteDialog.description", { name: toDelete?.name })}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              type="button"
              variant="ghost"
              onClick={() => setToDelete(null)}
              disabled={deleteWebhook.isPending}
            >
              {t("actions.cancel", { ns: "common" })}
            </Button>
            <Button
              type="button"
              variant="destructive"
              disabled={deleteWebhook.isPending}
              onClick={confirmDelete}
            >
              {deleteWebhook.isPending
                ? t("deleteDialog.confirmPending")
                : t("deleteDialog.confirm")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Deliveries dialog */}
      <DeliveriesDialog
        webhookId={deliveriesFor?.id ?? null}
        webhookName={deliveriesFor?.name ?? ""}
        open={deliveriesFor != null}
        onClose={() => setDeliveriesFor(null)}
      />
    </div>
  );
}
