import { useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { ArrowLeft, Power, Trash2 } from "lucide-react";
import { toast } from "sonner";
import {
  useUser,
  useDeleteUser,
  useSetUserActive,
} from "@/features/users/api";
import { apiError } from "@/lib/api";
import { relativeTime } from "@/lib/utils";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Skeleton, Separator } from "@/components/ui/misc";
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
      <div className="mx-auto max-w-5xl space-y-4">
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-48 w-full" />
      </div>
    );
  }

  if (!user) {
    return (
      <div className="mx-auto max-w-5xl py-20 text-center text-muted-foreground">
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
    <div ref={ref} className="mx-auto max-w-5xl">
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
