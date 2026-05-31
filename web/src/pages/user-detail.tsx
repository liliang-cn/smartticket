import { useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { ArrowLeft, Power, Trash2, ShieldCheck, X, Plus, Lock } from "lucide-react";
import { toast } from "sonner";
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
  const { data: assigned, isLoading } = useUserRoles(userId);
  const { data: allRoles } = useRoles();
  const assign = useAssignRole(userId);
  const remove = useRemoveRole(userId);
  const [selected, setSelected] = useState<string>("");

  const assignedIds = new Set((assigned ?? []).map((r) => r.id));
  const available = (allRoles ?? []).filter((r) => !assignedIds.has(r.id));

  async function onAssign() {
    if (!selected) return;
    try {
      await assign.mutateAsync(Number(selected));
      toast.success("Role assigned");
      setSelected("");
    } catch (err) {
      toast.error(apiError(err, "Could not assign role"));
    }
  }

  async function onRemove(roleId: number, name: string) {
    try {
      await remove.mutateAsync(roleId);
      toast.success(`Removed ${name}`);
    } catch (err) {
      toast.error(apiError(err, "Could not remove role"));
    }
  }

  return (
    <Card data-reveal className="p-5">
      <div className="flex items-center gap-2">
        <ShieldCheck className="size-4 text-primary" />
        <Label>Roles</Label>
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
                onClick={() => onRemove(r.id, r.name)}
                disabled={remove.isPending}
                className="text-muted-foreground transition-colors hover:text-red-300 disabled:opacity-50"
                aria-label={`Remove ${r.name}`}
              >
                <X className="size-3" />
              </button>
            </span>
          ))
        ) : (
          <span className="text-sm text-muted-foreground">
            No roles assigned.
          </span>
        )}
      </div>

      <div className="mt-4 flex items-center gap-2">
        <Select value={selected} onValueChange={setSelected}>
          <SelectTrigger className="flex-1">
            <SelectValue placeholder="Select a role to assign…" />
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
                No more roles to assign.
              </div>
            )}
          </SelectContent>
        </Select>
        <Button
          size="sm"
          onClick={onAssign}
          disabled={!selected || assign.isPending}
        >
          <Plus /> Assign
        </Button>
      </div>
    </Card>
  );
}

export function UserDetailPage() {
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
      toast.success(user.is_active ? "User deactivated" : "User activated");
    } catch (err) {
      toast.error(apiError(err, "Could not update user"));
    }
  }

  async function onDelete() {
    if (userId == null) return;
    try {
      await del.mutateAsync(userId);
      toast.success("User deleted");
      setConfirmOpen(false);
      navigate("/users");
    } catch (err) {
      toast.error(apiError(err, "Could not delete user"));
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
        User not found.
        <div className="mt-4">
          <Button variant="secondary" asChild>
            <Link to="/users">
              <ArrowLeft /> Back to users
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
        <ArrowLeft className="size-4" /> Users
      </Link>

      <div data-reveal className="mb-6 flex flex-wrap items-start justify-between gap-4">
        <div>
          <div className="flex items-center gap-2">
            <Badge tone="neutral" className="capitalize">
              {user.role}
            </Badge>
            <Badge tone={user.is_active ? "green" : "slate"}>
              {user.is_active ? "active" : "inactive"}
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
            <Power /> {user.is_active ? "Deactivate" : "Activate"}
          </Button>
          <Button
            variant="destructive"
            size="sm"
            onClick={() => setConfirmOpen(true)}
          >
            <Trash2 /> Delete
          </Button>
        </div>
      </div>

      <div className="grid gap-6 lg:grid-cols-[1fr_18rem]">
        {/* Main */}
        <div className="space-y-5">
          <Card data-reveal className="p-5">
            <Label>Profile</Label>
            <div className="mt-2 divide-y divide-border/60">
              <MetaRow label="First name" value={user.first_name || "—"} />
              <MetaRow label="Last name" value={user.last_name || "—"} />
              <MetaRow
                label="Email"
                value={<span className="font-mono text-xs">{user.email}</span>}
              />
              <MetaRow
                label="Username"
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
              label="Role"
              value={
                <Badge tone="neutral" className="capitalize">
                  {user.role}
                </Badge>
              }
            />
            <Separator />
            <MetaRow
              label="Status"
              value={
                <Badge tone={user.is_active ? "green" : "slate"}>
                  {user.is_active ? "active" : "inactive"}
                </Badge>
              }
            />
            <Separator />
            <MetaRow
              label="Last login"
              value={
                user.last_login_at ? relativeTime(user.last_login_at) : "never"
              }
            />
            {user.created_at && (
              <MetaRow label="Created" value={relativeTime(user.created_at)} />
            )}
          </Card>
        </aside>
      </div>

      {/* Delete confirmation */}
      <Dialog open={confirmOpen} onOpenChange={setConfirmOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete user</DialogTitle>
            <DialogDescription>
              Delete <span className="font-medium text-foreground">{name}</span>?
              This soft-deletes the account.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <DialogClose asChild>
              <Button type="button" variant="ghost">
                Cancel
              </Button>
            </DialogClose>
            <Button
              variant="destructive"
              onClick={onDelete}
              disabled={del.isPending}
            >
              {del.isPending ? "Deleting…" : "Delete user"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
