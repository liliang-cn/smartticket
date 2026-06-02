import { useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { ArrowLeft, Power, Trash2, ShieldCheck, X, Plus, Lock } from "lucide-react";
import { toast } from "sonner";
import { useTranslation } from "react-i18next";
import {
  useUser,
  useDeleteUser,
  useSetUserActive,
} from "@/features/users/api";
import {
  useRoles,
  useUserRoles,
  useAssignRole,
  useRemoveRole,
} from "@/features/rbac/api";
import { apiError } from "@/lib/api";
import { relativeTime } from "@/lib/utils";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Skeleton, Separator } from "@/components/ui/misc";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogClose,
} from "@/components/ui/dialog";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
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

function RolesCard({ userId }: { userId: number }) {
  const { t } = useTranslation("users");
  const { data: assigned, isLoading } = useUserRoles(userId);
  const { data: allRoles } = useRoles();
  const assign = useAssignRole(userId);
  const remove = useRemoveRole(userId);
  const [selected, setSelected] = useState<string>("");
  const [toRemove, setToRemove] = useState<{ id: number; name: string } | null>(
    null
  );

  const assignedIds = new Set((assigned ?? []).map((r) => r.id));
  const available = (allRoles ?? []).filter((r) => !assignedIds.has(r.id));

  async function onAssign() {
    if (!selected) return;
    try {
      await assign.mutateAsync(Number(selected));
      toast.success(t("toasts.role_assigned"));
      setSelected("");
    } catch (err) {
      toast.error(apiError(err, t("toasts.role_assign_error")));
    }
  }

  async function confirmRemove() {
    if (!toRemove) return;
    try {
      await remove.mutateAsync(toRemove.id);
      toast.success(t("toasts.role_removed", { name: toRemove.name }));
      setToRemove(null);
    } catch (err) {
      toast.error(apiError(err, t("toasts.role_remove_error")));
    }
  }

  return (
    <Card data-reveal className="p-5">
      <div className="flex items-center gap-2">
        <ShieldCheck className="size-4 text-primary" />
        <Label>{t("roles_card.title")}</Label>
      </div>

      <div className="mt-3 flex flex-wrap gap-2">
        {isLoading ? (
          <>
            <Skeleton className="h-7 w-24 rounded-full" />
            <Skeleton className="h-7 w-20 rounded-full" />
          </>
        ) : assigned && assigned.length > 0 ? (
          assigned.map((r) => (
            <span
              key={r.id}
              className="inline-flex items-center gap-1.5 rounded-full border border-border bg-muted px-2.5 py-1 text-xs font-medium font-mono tracking-tight"
            >
              {r.is_system && <Lock className="size-3 text-primary" />}
              <span className="capitalize">{r.name}</span>
              <button
                type="button"
                onClick={() => setToRemove({ id: r.id, name: r.name })}
                disabled={remove.isPending}
                className="text-muted-foreground transition-colors hover:text-red-300 disabled:opacity-50"
                aria-label={t("roles_card.remove_aria", { name: r.name })}
              >
                <X className="size-3" />
              </button>
            </span>
          ))
        ) : (
          <span className="text-sm text-muted-foreground">
            {t("roles_card.no_roles")}
          </span>
        )}
      </div>

      <div className="mt-4 flex items-center gap-2">
        <Select value={selected} onValueChange={setSelected}>
          <SelectTrigger className="flex-1">
            <SelectValue placeholder={t("roles_card.select_placeholder")} />
          </SelectTrigger>
          <SelectContent>
            {available.length > 0 ? (
              available.map((r) => (
                <SelectItem key={r.id} value={String(r.id)} className="capitalize">
                  {r.name}
                </SelectItem>
              ))
            ) : (
              <div className="px-2 py-2 text-xs text-muted-foreground">
                {t("roles_card.no_more_roles")}
              </div>
            )}
          </SelectContent>
        </Select>
        <Button
          size="sm"
          onClick={onAssign}
          disabled={!selected || assign.isPending}
        >
          <Plus /> {t("actions.assign")}
        </Button>
      </div>

      <ConfirmDialog
        open={!!toRemove}
        onOpenChange={(o) => !o && setToRemove(null)}
        title={t("roles_card.remove_dialog.title")}
        description={
          toRemove
            ? t("roles_card.remove_dialog.description", { name: toRemove.name })
            : undefined
        }
        confirmLabel={t("roles_card.remove_dialog.confirm")}
        pending={remove.isPending}
        onConfirm={confirmRemove}
      />
    </Card>
  );
}

export function UserDetailPage() {
  const { t } = useTranslation("users");
  const { id } = useParams();
  const navigate = useNavigate();
  const userId = id ? Number(id) : undefined;
  const { data: user, isLoading } = useUser(userId);
  const del = useDeleteUser();
  const setActive = useSetUserActive(userId ?? 0);
  const [confirmOpen, setConfirmOpen] = useState(false);
  const ref = useReveal(user?.id);

  async function onToggleActive() {
    if (!user) return;
    try {
      await setActive.mutateAsync(!user.is_active);
      toast.success(user.is_active ? t("toasts.toggle_deactivated") : t("toasts.toggle_activated"));
    } catch (err) {
      toast.error(apiError(err, t("toasts.toggle_error")));
    }
  }

  async function onDelete() {
    if (userId == null) return;
    try {
      await del.mutateAsync(userId);
      toast.success(t("toasts.deleted"));
      setConfirmOpen(false);
      navigate("/users");
    } catch (err) {
      toast.error(apiError(err, t("toasts.delete_error")));
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

  if (!user) {
    return (
      <div className="w-full py-20 text-center text-muted-foreground">
        {t("detail.not_found")}
        <div className="mt-4">
          <Button variant="secondary" asChild>
            <Link to="/users">
              <ArrowLeft /> {t("detail.back_to_users")}
            </Link>
          </Button>
        </div>
      </div>
    );
  }

  const name = `${user.first_name} ${user.last_name}`.trim() || user.username;

  return (
    <div ref={ref} className="w-full">
      <Link
        to="/users"
        className="mb-4 inline-flex items-center gap-1.5 text-sm text-muted-foreground transition-colors hover:text-foreground"
      >
        <ArrowLeft className="size-4" /> {t("actions.back_to_users")}
      </Link>

      <div data-reveal className="mb-6 flex flex-wrap items-start justify-between gap-4">
        <div>
          <div className="flex items-center gap-2">
            <Badge tone="neutral">
              {t(`roles.${user.role}`, user.role)}
            </Badge>
            <Badge tone={user.is_active ? "green" : "slate"}>
              {user.is_active ? t("status.active") : t("status.inactive")}
            </Badge>
          </div>
          <h1 className="mt-1 text-2xl">{name}</h1>
          <div className="mt-0.5 font-mono text-xs text-muted-foreground">
            @{user.username}
          </div>
        </div>
        <div className="flex gap-2">
          <Button
            variant="secondary"
            size="sm"
            onClick={onToggleActive}
            disabled={setActive.isPending}
          >
            <Power /> {user.is_active ? t("actions.deactivate") : t("actions.activate")}
          </Button>
          <Button
            variant="destructive"
            size="sm"
            onClick={() => setConfirmOpen(true)}
          >
            <Trash2 /> {t("actions.delete")}
          </Button>
        </div>
      </div>

      <div className="grid gap-6 lg:grid-cols-[1fr_18rem]">
        {/* Main */}
        <div className="space-y-5">
          <Card data-reveal className="p-5">
            <Label>{t("detail.profile")}</Label>
            <div className="mt-2 divide-y divide-border/60">
              <MetaRow label={t("detail.meta.first_name")} value={user.first_name || "—"} />
              <MetaRow label={t("detail.meta.last_name")} value={user.last_name || "—"} />
              <MetaRow
                label={t("detail.meta.email")}
                value={<span className="font-mono text-xs">{user.email}</span>}
              />
              <MetaRow
                label={t("detail.meta.username")}
                value={
                  <span className="font-mono text-xs">{user.username}</span>
                }
              />
            </div>
          </Card>

          <RolesCard userId={user.id} />
        </div>

        {/* Meta sidebar */}
        <aside data-reveal className="space-y-4">
          <Card className="p-5">
            <MetaRow
              label={t("detail.meta.role")}
              value={
                <Badge tone="neutral">
                  {t(`roles.${user.role}`, user.role)}
                </Badge>
              }
            />
            <Separator />
            <MetaRow
              label={t("detail.meta.status")}
              value={
                <Badge tone={user.is_active ? "green" : "slate"}>
                  {user.is_active ? t("status.active") : t("status.inactive")}
                </Badge>
              }
            />
            <Separator />
            <MetaRow
              label={t("detail.meta.last_login")}
              value={
                user.last_login_at ? relativeTime(user.last_login_at) : t("detail.meta.never")
              }
            />
            {user.created_at && (
              <MetaRow label={t("detail.meta.created")} value={relativeTime(user.created_at)} />
            )}
          </Card>
        </aside>
      </div>

      {/* Delete confirmation */}
      <Dialog open={confirmOpen} onOpenChange={setConfirmOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("delete_dialog.title")}</DialogTitle>
            <DialogDescription>
              {t("delete_dialog.description", { name })}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <DialogClose asChild>
              <Button type="button" variant="ghost">
                {t("actions.cancel", { ns: "common" })}
              </Button>
            </DialogClose>
            <Button
              variant="destructive"
              onClick={onDelete}
              disabled={del.isPending}
            >
              {del.isPending ? t("delete_dialog.confirm_pending") : t("delete_dialog.confirm")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
