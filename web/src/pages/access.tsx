import { useState } from "react";
import { useTranslation } from "react-i18next";
import {
  ShieldCheck,
  KeyRound,
  Lock,
  Pencil,
  Trash2,
  SlidersHorizontal,
} from "lucide-react";
import { toast } from "sonner";
import {
  useRoles,
  usePermissions,
  useDeleteRole,
  useDeletePermission,
  type RbacPermissionFull,
} from "@/features/rbac/api";
import { RoleFormDialog } from "@/features/rbac/role-form-dialog";
import { PermissionFormDialog } from "@/features/rbac/permission-form-dialog";
import { RolePermissionsDialog } from "@/features/rbac/role-permissions-dialog";
import { apiError } from "@/lib/api";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/misc";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogClose,
} from "@/components/ui/dialog";
import { useReveal } from "@/lib/use-reveal";

function groupByCategory(
  perms: RbacPermissionFull[]
): [string, RbacPermissionFull[]][] {
  const map = new Map<string, RbacPermissionFull[]>();
  for (const p of perms) {
    const key = p.category || "uncategorized";
    const arr = map.get(key) ?? [];
    arr.push(p);
    map.set(key, arr);
  }
  return Array.from(map.entries()).sort((a, b) => a[0].localeCompare(b[0]));
}

/** Small inline confirm dialog driven by an async onConfirm. */
function ConfirmDeleteDialog({
  open,
  onOpenChange,
  title,
  description,
  pending,
  onConfirm,
}: {
  open: boolean;
  onOpenChange: (v: boolean) => void;
  title: string;
  description: string;
  pending: boolean;
  onConfirm: () => void;
}) {
  const { t } = useTranslation("rbac");
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>{description}</DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <DialogClose asChild>
            <Button type="button" variant="ghost">
              {t("confirm_dialog.cancel")}
            </Button>
          </DialogClose>
          <Button
            type="button"
            variant="destructive"
            disabled={pending}
            onClick={onConfirm}
          >
            {pending ? t("confirm_dialog.deleting") : t("confirm_dialog.delete")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export function AccessPage() {
  const { t } = useTranslation("rbac");
  const { data: roles, isLoading: rolesLoading } = useRoles();
  const { data: permissions, isLoading: permsLoading } = usePermissions();
  const deleteRole = useDeleteRole();
  const deletePermission = useDeletePermission();
  const ref = useReveal<HTMLDivElement>();

  // Pending deletions (id of the entity being confirmed for delete).
  const [roleToDelete, setRoleToDelete] = useState<{
    id: number;
    name: string;
  } | null>(null);
  const [permToDelete, setPermToDelete] = useState<{
    id: number;
    code: string;
  } | null>(null);

  const grouped = groupByCategory(permissions ?? []);

  async function confirmDeleteRole() {
    if (!roleToDelete) return;
    try {
      await deleteRole.mutateAsync(roleToDelete.id);
      toast.success(t("roles.toast.deleted"));
      setRoleToDelete(null);
    } catch (err) {
      toast.error(apiError(err, t("roles.toast.delete_error")));
    }
  }

  async function confirmDeletePermission() {
    if (!permToDelete) return;
    try {
      await deletePermission.mutateAsync(permToDelete.id);
      toast.success(t("permissions.toast.deleted"));
      setPermToDelete(null);
    } catch (err) {
      toast.error(apiError(err, t("permissions.toast.delete_error")));
    }
  }

  return (
    <div ref={ref} className="w-full">
      <div data-reveal className="mb-6">
        <div className="font-mono text-xs uppercase tracking-[0.25em] text-muted-foreground">
          {t("page.eyebrow")}
        </div>
        <h1 className="mt-1 text-3xl">{t("page.title")}</h1>
      </div>

      {/* Roles */}
      <section data-reveal className="mb-10">
        <div className="mb-3 flex items-center gap-2">
          <ShieldCheck className="size-4 text-primary" />
          <h2 className="text-sm font-medium">{t("roles.heading")}</h2>
          {roles && (
            <span className="font-mono text-xs text-muted-foreground">
              {roles.length}
            </span>
          )}
          <div className="ml-auto">
            <RoleFormDialog />
          </div>
        </div>

        {rolesLoading ? (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {Array.from({ length: 3 }).map((_, i) => (
              <Card key={i} className="p-5">
                <Skeleton className="h-5 w-32" />
                <Skeleton className="mt-3 h-4 w-full" />
                <Skeleton className="mt-2 h-4 w-2/3" />
              </Card>
            ))}
          </div>
        ) : roles && roles.length > 0 ? (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {roles.map((role) => (
              <Card key={role.id} className="flex flex-col p-5">
                <div className="flex items-start justify-between gap-2">
                  <h3 className="font-medium capitalize text-foreground">
                    {role.name}
                  </h3>
                  {role.is_system && (
                    <Badge tone="amber">
                      <Lock className="size-3" /> {t("roles.system_badge")}
                    </Badge>
                  )}
                </div>
                {role.description && (
                  <p className="mt-1.5 text-sm text-muted-foreground">
                    {role.description}
                  </p>
                )}
                <div className="mt-3 flex flex-1 flex-wrap items-end gap-1.5">
                  {role.permissions && role.permissions.length > 0 ? (
                    <>
                      {role.permissions.slice(0, 8).map((p) => (
                        <span
                          key={p.id}
                          className="rounded-sm bg-muted px-1.5 py-0.5 font-mono text-[10px] text-muted-foreground"
                          title={p.name}
                        >
                          {p.code}
                        </span>
                      ))}
                      {role.permissions.length > 8 && (
                        <span className="font-mono text-[10px] text-muted-foreground/70">
                          {t("roles.more", { count: role.permissions.length - 8 })}
                        </span>
                      )}
                    </>
                  ) : (
                    <span className="font-mono text-[11px] text-muted-foreground/60">
                      {t("roles.no_permissions")}
                    </span>
                  )}
                </div>

                <div className="mt-4 flex items-center gap-1.5 border-t border-border pt-3">
                  <RolePermissionsDialog
                    role={role}
                    trigger={
                      <Button variant="outline" size="sm">
                        <SlidersHorizontal /> {t("roles.permissions_button")}
                      </Button>
                    }
                  />
                  <RoleFormDialog
                    role={role}
                    trigger={
                      <Button variant="ghost" size="icon" title={t("roles.edit_title")}>
                        <Pencil />
                      </Button>
                    }
                  />
                  <Button
                    variant="ghost"
                    size="icon"
                    title={
                      role.is_system
                        ? t("roles.delete_system_hint")
                        : t("roles.delete_title")
                    }
                    disabled={role.is_system}
                    onClick={() =>
                      setRoleToDelete({ id: role.id, name: role.name })
                    }
                  >
                    <Trash2 />
                  </Button>
                </div>
              </Card>
            ))}
          </div>
        ) : (
          <Card className="py-16 text-center">
            <ShieldCheck className="mx-auto size-8 text-muted-foreground/40" />
            <p className="mt-3 text-sm text-muted-foreground">
              {t("roles.empty")}
            </p>
          </Card>
        )}
      </section>

      {/* Permissions */}
      <section data-reveal>
        <div className="mb-3 flex items-center gap-2">
          <KeyRound className="size-4 text-primary" />
          <h2 className="text-sm font-medium">{t("permissions.heading")}</h2>
          {permissions && (
            <span className="font-mono text-xs text-muted-foreground">
              {permissions.length}
            </span>
          )}
          <div className="ml-auto">
            <PermissionFormDialog />
          </div>
        </div>

        {permsLoading ? (
          <Card className="p-5">
            <Skeleton className="h-4 w-40" />
            <Skeleton className="mt-3 h-4 w-full" />
            <Skeleton className="mt-2 h-4 w-3/4" />
          </Card>
        ) : grouped.length > 0 ? (
          <div className="space-y-5">
            {grouped.map(([category, perms]) => (
              <Card key={category} className="p-5">
                <div className="mb-3 flex items-center gap-2">
                  <span className="font-mono text-[11px] uppercase tracking-wider text-primary/80">
                    {category}
                  </span>
                  <span className="font-mono text-[11px] text-muted-foreground/60">
                    {perms.length}
                  </span>
                </div>
                <div className="grid gap-x-6 gap-y-1 sm:grid-cols-2 lg:grid-cols-3">
                  {perms.map((p) => (
                    <div
                      key={p.id}
                      className="group flex min-w-0 items-center gap-2 rounded-md px-1.5 py-1 transition-colors hover:bg-accent/40"
                    >
                      <div className="min-w-0 flex-1">
                        <div className="flex items-center gap-1.5">
                          <span className="truncate font-mono text-xs text-foreground">
                            {p.code}
                          </span>
                          {p.is_system && (
                            <Lock className="size-3 shrink-0 text-muted-foreground/60" />
                          )}
                        </div>
                        <div className="truncate text-[11px] text-muted-foreground">
                          {p.name}
                        </div>
                      </div>
                      <div className="flex shrink-0 items-center opacity-0 transition-opacity group-hover:opacity-100">
                        <PermissionFormDialog
                          permission={p}
                          trigger={
                            <Button
                              variant="ghost"
                              size="icon"
                              className="size-7"
                              title={t("permissions.edit_title")}
                            >
                              <Pencil />
                            </Button>
                          }
                        />
                        <Button
                          variant="ghost"
                          size="icon"
                          className="size-7"
                          title={
                            p.is_system
                              ? t("permissions.delete_system_hint")
                              : t("permissions.delete_title")
                          }
                          disabled={p.is_system}
                          onClick={() =>
                            setPermToDelete({ id: p.id, code: p.code })
                          }
                        >
                          <Trash2 />
                        </Button>
                      </div>
                    </div>
                  ))}
                </div>
              </Card>
            ))}
          </div>
        ) : (
          <Card className="py-16 text-center">
            <KeyRound className="mx-auto size-8 text-muted-foreground/40" />
            <p className="mt-3 text-sm text-muted-foreground">
              {t("permissions.empty")}
            </p>
          </Card>
        )}
      </section>

      {/* Delete confirmations */}
      <ConfirmDeleteDialog
        open={roleToDelete != null}
        onOpenChange={(v) => !v && setRoleToDelete(null)}
        title={t("roles.confirm_delete.title")}
        description={t("roles.confirm_delete.description", { name: roleToDelete?.name ?? "" })}
        pending={deleteRole.isPending}
        onConfirm={confirmDeleteRole}
      />
      <ConfirmDeleteDialog
        open={permToDelete != null}
        onOpenChange={(v) => !v && setPermToDelete(null)}
        title={t("permissions.confirm_delete.title")}
        description={t("permissions.confirm_delete.description", { name: permToDelete?.code ?? "" })}
        pending={deletePermission.isPending}
        onConfirm={confirmDeletePermission}
      />
    </div>
  );
}
