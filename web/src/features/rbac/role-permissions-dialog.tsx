import { useMemo, useState } from "react";
import { toast } from "sonner";
import {
  usePermissions,
  useRolePermissions,
  useAssignPermissionToRole,
  useRemovePermissionFromRole,
  type RbacPermissionFull,
  type RbacRoleFull,
} from "@/features/rbac/api";
import { apiError } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/misc";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
  DialogClose,
} from "@/components/ui/dialog";

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

interface RolePermissionsDialogProps {
  role: RbacRoleFull;
  trigger: React.ReactNode;
}

export function RolePermissionsDialog({
  role,
  trigger,
}: RolePermissionsDialogProps) {
  const [open, setOpen] = useState(false);

  const { data: allPerms, isLoading: permsLoading } = usePermissions();
  const { data: rolePerms, isLoading: rolePermsLoading } = useRolePermissions(
    open ? role.id : undefined
  );
  const assign = useAssignPermissionToRole(role.id);
  const remove = useRemovePermissionFromRole(role.id);

  // Track which permission row is mid-flight to disable just that checkbox.
  const [busyId, setBusyId] = useState<number | null>(null);

  const assignedIds = useMemo(
    () => new Set((rolePerms ?? []).map((p) => p.id)),
    [rolePerms]
  );
  const grouped = useMemo(() => groupByCategory(allPerms ?? []), [allPerms]);

  async function toggle(perm: RbacPermissionFull, checked: boolean) {
    setBusyId(perm.id);
    try {
      if (checked) {
        await assign.mutateAsync(perm.id);
        toast.success(`Granted ${perm.code}`);
      } else {
        await remove.mutateAsync(perm.id);
        toast.success(`Revoked ${perm.code}`);
      }
    } catch (err) {
      toast.error(apiError(err, "Could not update permissions"));
    } finally {
      setBusyId(null);
    }
  }

  const loading = permsLoading || rolePermsLoading;

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>{trigger}</DialogTrigger>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>
            Permissions for <span className="capitalize">{role.name}</span>
          </DialogTitle>
          <DialogDescription>
            Toggle the permissions granted to this role. Changes save
            immediately.
          </DialogDescription>
        </DialogHeader>

        <div className="max-h-[60vh] space-y-5 overflow-y-auto pr-1">
          {loading ? (
            <div className="space-y-3">
              <Skeleton className="h-4 w-32" />
              <Skeleton className="h-4 w-full" />
              <Skeleton className="h-4 w-3/4" />
            </div>
          ) : grouped.length > 0 ? (
            grouped.map(([category, perms]) => (
              <div key={category}>
                <div className="mb-2 font-mono text-[11px] uppercase tracking-wider text-primary/80">
                  {category}
                </div>
                <div className="space-y-1">
                  {perms.map((p) => {
                    const checked = assignedIds.has(p.id);
                    return (
                      <label
                        key={p.id}
                        className="flex cursor-pointer items-start gap-3 rounded-md px-2 py-1.5 transition-colors hover:bg-accent/50"
                      >
                        <input
                          type="checkbox"
                          className="mt-1 size-4 shrink-0 accent-primary"
                          checked={checked}
                          disabled={busyId === p.id}
                          onChange={(e) => toggle(p, e.target.checked)}
                        />
                        <span className="min-w-0">
                          <span className="block truncate font-mono text-xs text-foreground">
                            {p.code}
                          </span>
                          <span className="block truncate text-[11px] text-muted-foreground">
                            {p.name}
                          </span>
                        </span>
                      </label>
                    );
                  })}
                </div>
              </div>
            ))
          ) : (
            <p className="py-8 text-center text-sm text-muted-foreground">
              No permissions defined yet.
            </p>
          )}
        </div>

        <DialogFooter>
          <DialogClose asChild>
            <Button type="button" variant="ghost">
              Done
            </Button>
          </DialogClose>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
