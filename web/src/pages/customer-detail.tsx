import { useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { ArrowLeft, Pencil, Trash2, Users } from "lucide-react";
import { toast } from "sonner";
import {
  useCustomer,
  useCustomerUsers,
  useDeleteCustomer,
} from "@/features/customers/api";
import { CustomerFormDialog } from "@/features/customers/customer-form-dialog";
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

export function CustomerDetailPage() {
  const { id } = useParams();
  const navigate = useNavigate();
  const customerId = id ? Number(id) : undefined;
  const { data: customer, isLoading } = useCustomer(customerId);
  const { data: users } = useCustomerUsers(customerId);
  const del = useDeleteCustomer();
  const [confirmOpen, setConfirmOpen] = useState(false);
  const ref = useReveal(customer?.id);

  async function onDelete() {
    if (customerId == null) return;
    try {
      await del.mutateAsync(customerId);
      toast.success("Customer deleted");
      setConfirmOpen(false);
      navigate("/customers");
    } catch (err) {
      toast.error(apiError(err, "Could not delete customer"));
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

  if (!customer) {
    return (
      <div className="mx-auto max-w-5xl py-20 text-center text-muted-foreground">
        Customer not found.
        <div className="mt-4">
          <Button variant="secondary" asChild>
            <Link to="/customers">
              <ArrowLeft /> Back to customers
            </Link>
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div ref={ref} className="mx-auto max-w-5xl">
      <Link
        to="/customers"
        className="mb-4 inline-flex items-center gap-1.5 text-sm text-muted-foreground transition-colors hover:text-foreground"
      >
        <ArrowLeft className="size-4" /> Customers
      </Link>

      <div data-reveal className="mb-6 flex flex-wrap items-start justify-between gap-4">
        <div>
          <div className="flex items-center gap-2">
            {customer.code && (
              <span className="font-mono text-xs text-primary/80">
                {customer.code}
              </span>
            )}
            <Badge tone={customer.is_active ? "green" : "slate"}>
              {customer.is_active ? "active" : "inactive"}
            </Badge>
          </div>
          <h1 className="mt-1 text-2xl">{customer.name}</h1>
        </div>
        <div className="flex gap-2">
          <CustomerFormDialog
            customer={customer}
            trigger={
              <Button variant="secondary" size="sm">
                <Pencil /> Edit
              </Button>
            }
          />
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
            <Label>Description</Label>
            <p className="mt-2 whitespace-pre-wrap text-sm leading-relaxed text-foreground/90">
              {customer.description || "No description provided."}
            </p>
          </Card>

          <div data-reveal>
            <h2 className="mb-3 flex items-center gap-2 text-sm font-semibold">
              <Users className="size-4 text-muted-foreground" />
              Contacts{" "}
              <span className="font-mono text-xs text-muted-foreground">
                ({users?.length ?? 0})
              </span>
            </h2>
            <Card className="overflow-hidden">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border text-left font-mono text-[11px] uppercase tracking-wider text-muted-foreground">
                    <th className="px-4 py-3 font-medium">User</th>
                    <th className="px-4 py-3 font-medium">Email</th>
                    <th className="px-4 py-3 font-medium">Role</th>
                    <th className="px-4 py-3 font-medium">Active</th>
                  </tr>
                </thead>
                <tbody>
                  {users && users.length > 0 ? (
                    users.map((u) => (
                      <tr
                        key={u.id}
                        className="border-b border-border/60 last:border-0"
                      >
                        <td className="px-4 py-3 font-medium">{u.username}</td>
                        <td className="px-4 py-3 font-mono text-xs text-muted-foreground">
                          {u.email}
                        </td>
                        <td className="px-4 py-3">
                          <Badge tone="neutral" className="capitalize">
                            {u.role}
                          </Badge>
                        </td>
                        <td className="px-4 py-3">
                          <Badge tone={u.is_active ? "green" : "slate"}>
                            {u.is_active ? "active" : "inactive"}
                          </Badge>
                        </td>
                      </tr>
                    ))
                  ) : (
                    <tr>
                      <td
                        colSpan={4}
                        className="px-4 py-12 text-center text-sm text-muted-foreground"
                      >
                        No contacts linked to this customer.
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </Card>
          </div>
        </div>

        {/* Meta sidebar */}
        <aside data-reveal className="space-y-4">
          <Card className="p-5">
            <MetaRow
              label="Code"
              value={
                customer.code ? (
                  <span className="font-mono text-xs">{customer.code}</span>
                ) : (
                  "—"
                )
              }
            />
            <Separator />
            <MetaRow
              label="Domain"
              value={
                <span className="font-mono text-xs">
                  {customer.domain || "—"}
                </span>
              }
            />
            <Separator />
            <MetaRow
              label="Status"
              value={
                <Badge tone={customer.is_active ? "green" : "slate"}>
                  {customer.is_active ? "active" : "inactive"}
                </Badge>
              }
            />
            <MetaRow label="Created" value={relativeTime(customer.created_at)} />
            <MetaRow label="Updated" value={relativeTime(customer.updated_at)} />
          </Card>
        </aside>
      </div>

      {/* Delete confirmation */}
      <Dialog open={confirmOpen} onOpenChange={setConfirmOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete customer</DialogTitle>
            <DialogDescription>
              Delete <span className="font-medium text-foreground">{customer.name}</span>?
              This soft-deletes the organization. Their contacts and tickets are
              not removed.
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
              {del.isPending ? "Deleting…" : "Delete customer"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
