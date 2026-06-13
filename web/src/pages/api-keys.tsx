import { useState } from "react";
import { KeyRound, Plus, Ban, Copy, Check, X } from "lucide-react";
import { toast } from "sonner";
import { useTranslation } from "react-i18next";
import {
  useApiKeys,
  useCreateApiKey,
  useRevokeApiKey,
} from "@/features/apikeys/api";
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

// --- One-time key reveal banner -------------------------------------------

function KeyRevealBanner({
  secretKey,
  onClose,
}: {
  secretKey: string;
  onClose: () => void;
}) {
  const { t } = useTranslation("apikeys");
  const [copied, setCopied] = useState(false);

  async function handleCopy() {
    try {
      await navigator.clipboard.writeText(secretKey);
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
              {secretKey}
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

// --- Create form dialog ----------------------------------------------------

function CreateKeyDialog({ onCreated }: { onCreated: (key: string) => void }) {
  const { t } = useTranslation("apikeys");
  const [open, setOpen] = useState(false);
  const [name, setName] = useState("");
  const [userId, setUserId] = useState("");
  const create = useCreateApiKey();

  function handleOpen(v: boolean) {
    if (v) {
      setName("");
      setUserId("");
    }
    setOpen(v);
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    const uid = parseInt(userId, 10);
    if (!name.trim()) {
      toast.error(t("validation.nameRequired"));
      return;
    }
    if (!userId || isNaN(uid) || uid <= 0) {
      toast.error(t("validation.userIdRequired"));
      return;
    }
    try {
      const result = await create.mutateAsync({ name: name.trim(), user_id: uid });
      toast.success(t("toast.created"));
      setOpen(false);
      onCreated(result.key);
    } catch (err) {
      toast.error(apiError(err, t("toast.createFailed")));
    }
  }

  return (
    <Dialog open={open} onOpenChange={handleOpen}>
      <Button onClick={() => handleOpen(true)}>
        <Plus /> {t("newKey")}
      </Button>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>{t("form.title")}</DialogTitle>
          <DialogDescription>{t("form.description")}</DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-1.5">
            <Label htmlFor="ak-name">{t("form.name")}</Label>
            <Input
              id="ak-name"
              placeholder={t("form.namePlaceholder")}
              value={name}
              onChange={(e) => setName(e.target.value)}
              maxLength={100}
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="ak-uid">{t("form.userId")}</Label>
            <Input
              id="ak-uid"
              type="number"
              min={1}
              placeholder="1"
              value={userId}
              onChange={(e) => setUserId(e.target.value)}
            />
            <p className="text-xs text-muted-foreground">{t("form.userIdHint")}</p>
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

// --- Page ------------------------------------------------------------------

export function ApiKeysPage() {
  const { t } = useTranslation("apikeys");
  const { data: keys, isLoading } = useApiKeys();
  const revoke = useRevokeApiKey();
  const ref = useReveal<HTMLDivElement>();

  // After creating, store the revealed plaintext key here.
  const [revealedKey, setRevealedKey] = useState<string | null>(null);
  // Revoke confirm dialog.
  const [toRevoke, setToRevoke] = useState<{ id: number; name: string } | null>(null);

  async function confirmRevoke() {
    if (!toRevoke) return;
    try {
      await revoke.mutateAsync(toRevoke.id);
      toast.success(t("toast.revoked"));
      setToRevoke(null);
    } catch (err) {
      toast.error(apiError(err, t("toast.revokeFailed")));
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
        <CreateKeyDialog
          onCreated={(key) => {
            setRevealedKey(key);
          }}
        />
      </div>

      {revealedKey && (
        <div data-reveal className="mb-6">
          <KeyRevealBanner
            secretKey={revealedKey}
            onClose={() => setRevealedKey(null)}
          />
        </div>
      )}

      <Card data-reveal className="overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border text-left font-mono text-[11px] uppercase tracking-wider text-muted-foreground">
              <th className="px-4 py-3 font-medium">{t("table.name")}</th>
              <th className="px-4 py-3 font-medium">{t("table.prefix")}</th>
              <th className="px-4 py-3 font-medium">{t("table.userId")}</th>
              <th className="px-4 py-3 font-medium">{t("table.status")}</th>
              <th className="px-4 py-3 font-medium">{t("table.lastUsed")}</th>
              <th className="px-4 py-3 font-medium">{t("table.created")}</th>
              <th className="px-4 py-3" />
            </tr>
          </thead>
          <tbody>
            {isLoading ? (
              Array.from({ length: 3 }).map((_, i) => (
                <tr key={i} className="border-b border-border/60">
                  {Array.from({ length: 7 }).map((__, j) => (
                    <td key={j} className="px-4 py-3.5">
                      <Skeleton className="h-4 w-full" />
                    </td>
                  ))}
                </tr>
              ))
            ) : keys && keys.length > 0 ? (
              keys.map((k) => (
                <tr
                  key={k.id}
                  className="border-b border-border/60 transition-colors last:border-0 hover:bg-accent/50"
                >
                  <td className="px-4 py-3.5 font-medium text-foreground">
                    {k.name}
                  </td>
                  <td className="px-4 py-3.5 font-mono text-xs text-muted-foreground">
                    {k.key_prefix}…
                  </td>
                  <td className="px-4 py-3.5 font-mono text-xs text-muted-foreground">
                    {k.user_id}
                  </td>
                  <td className="px-4 py-3.5">
                    <Badge tone={k.is_active ? "green" : "slate"}>
                      {k.is_active ? t("badges.active") : t("badges.revoked")}
                    </Badge>
                  </td>
                  <td className="px-4 py-3.5 text-xs text-muted-foreground">
                    {fmtTime(k.last_used_at)}
                  </td>
                  <td className="px-4 py-3.5 text-xs text-muted-foreground">
                    {fmtTime(k.created_at)}
                  </td>
                  <td className="px-2 py-3.5 text-right">
                    {k.is_active && (
                      <Button
                        variant="ghost"
                        size="sm"
                        className="text-destructive hover:text-destructive"
                        title={t("actions.revokeTitle")}
                        onClick={() => setToRevoke({ id: k.id, name: k.name })}
                      >
                        <Ban className="size-4" />
                        {t("actions.revoke")}
                      </Button>
                    )}
                  </td>
                </tr>
              ))
            ) : (
              <tr>
                <td colSpan={7} className="px-4 py-16 text-center">
                  <KeyRound className="mx-auto size-8 text-muted-foreground/40" />
                  <p className="mt-3 text-sm text-muted-foreground">
                    {t("empty")}
                  </p>
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </Card>

      {/* Revoke confirmation dialog */}
      <Dialog
        open={toRevoke != null}
        onOpenChange={(v) => !v && setToRevoke(null)}
      >
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>{t("revokeDialog.title")}</DialogTitle>
            <DialogDescription>
              {t("revokeDialog.description", { name: toRevoke?.name })}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              type="button"
              variant="ghost"
              onClick={() => setToRevoke(null)}
              disabled={revoke.isPending}
            >
              {t("actions.cancel", { ns: "common" })}
            </Button>
            <Button
              type="button"
              variant="destructive"
              disabled={revoke.isPending}
              onClick={confirmRevoke}
            >
              {revoke.isPending
                ? t("revokeDialog.confirmPending")
                : t("revokeDialog.confirm")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
