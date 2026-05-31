import { useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import {
  ArrowLeft,
  Pencil,
  Trash2,
  Users,
  UserCheck,
  UserX,
  CreditCard,
  Package,
  Timer,
} from "lucide-react";
import { useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import {
  useCustomer,
  useCustomerUsers,
  useDeleteCustomer,
} from "@/features/customers/api";
import { useDeleteUser, useSetUserActive } from "@/features/users/api";
import { useSubscriptions } from "@/features/subscriptions/api";
import type { CustomerUser } from "@/lib/types";
import { CustomerFormDialog } from "@/features/customers/customer-form-dialog";
import { AddContactDialog } from "@/features/customers/add-contact-dialog";
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

function ContactRow({
  user,
  customerId,
}: {
  user: CustomerUser;
  customerId: number;
}) {
  const qc = useQueryClient();
  const setActive = useSetUserActive(user.id);
  const del = useDeleteUser();
  const [confirmOpen, setConfirmOpen] = useState(false);

  const displayName =
    `${user.first_name} ${user.last_name}`.trim() || user.username;

  function refreshContacts() {
    qc.invalidateQueries({ queryKey: ["customer-users", customerId] });
  }

  async function onToggleActive() {
    try {
      await setActive.mutateAsync(!user.is_active ? true : false);
      refreshContacts();
      toast.success(user.is_active ? "Contact deactivated" : "Contact activated");
    } catch (err) {
      toast.error(apiError(err));
    }
  }

  async function onRemove() {
    try {
      await del.mutateAsync(user.id);
      refreshContacts();
      setConfirmOpen(false);
      toast.success("Contact removed");
    } catch (err) {
      toast.error(apiError(err, "Could not remove contact"));
    }
  }

  return (
    <tr className="border-b border-border/60 last:border-0">
      <td className="px-4 py-3 font-medium">{user.username}</td>
      <td className="px-4 py-3 font-mono text-xs text-muted-foreground">
        {user.email}
      </td>
      <td className="px-4 py-3">
        <Badge tone="neutral" className="capitalize">
          {user.role}
        </Badge>
      </td>
      <td className="px-4 py-3">
        <Badge tone={user.is_active ? "green" : "slate"}>
          {user.is_active ? "active" : "inactive"}
        </Badge>
      </td>
      <td className="px-4 py-3">
        <div className="flex items-center justify-end gap-1">
          <Button
            size="sm"
            variant="ghost"
            onClick={onToggleActive}
            disabled={setActive.isPending}
          >
            {user.is_active ? (
              <>
                <UserX /> Deactivate
              </>
            ) : (
              <>
                <UserCheck /> Activate
              </>
            )}
          </Button>
          <Button
            size="icon"
            variant="ghost"
            className="text-destructive hover:text-destructive"
            onClick={() => setConfirmOpen(true)}
            aria-label={`Remove ${displayName}`}
          >
            <Trash2 />
          </Button>
        </div>

        <Dialog open={confirmOpen} onOpenChange={setConfirmOpen}>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Remove contact</DialogTitle>
              <DialogDescription>
                Remove{" "}
                <span className="font-medium text-foreground">
                  {displayName}
                </span>{" "}
                from this customer? This deletes the account.
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
                onClick={onRemove}
                disabled={del.isPending}
              >
                {del.isPending ? "Removing…" : "Remove contact"}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </td>
    </tr>
  );
}

/** SubscriptionsSection lists a customer's subscriptions — the products they
 * are entitled to and the SLA template that governs them. This is where a
 * customer's products / services / SLA live. */
function SubscriptionsSection({ customerId }: { customerId: number }) {
  const { data, isLoading } = useSubscriptions({
    page: 1,
    page_size: 100,
    customer_id: customerId,
  });
  const subs = data?.items ?? [];

  return (
    <div data-reveal>
      <div className="mb-3 flex items-center justify-between gap-2">
        <h2 className="flex items-center gap-2 text-sm font-semibold">
          <CreditCard className="size-4 text-muted-foreground" />
          Subscriptions{" "}
          <span className="font-mono text-xs text-muted-foreground">
            ({subs.length})
          </span>
        </h2>
      </div>
      <Card className="overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border text-left font-mono text-[11px] uppercase tracking-wider text-muted-foreground">
              <th className="px-4 py-3 font-medium">Product</th>
              <th className="px-4 py-3 font-medium">Plan</th>
              <th className="px-4 py-3 font-medium">SLA</th>
              <th className="px-4 py-3 font-medium">Nodes</th>
              <th className="px-4 py-3 font-medium">Status</th>
              <th className="px-4 py-3 font-medium">Expires</th>
            </tr>
          </thead>
          <tbody>
            {subs.length > 0 ? (
              subs.map((s) => (
                <tr
                  key={s.id}
                  className="border-b border-border/60 last:border-0"
                >
                  <td className="px-4 py-3">
                    <Link
                      to={`/products/${s.product_id}`}
                      className="inline-flex items-center gap-1.5 font-medium text-primary hover:underline"
                    >
                      <Package className="size-3.5" /> {s.product_name || `#${s.product_id}`}
                    </Link>
                  </td>
                  <td className="px-4 py-3 capitalize text-muted-foreground">
                    {s.plan || "—"}
                    <span className="ml-1 font-mono text-[11px] text-muted-foreground/70">
                      {s.billing_period ? `· ${s.billing_period}` : ""}
                    </span>
                  </td>
                  <td className="px-4 py-3">
                    {s.sla_template_name ? (
                      <span className="inline-flex items-center gap-1.5">
                        <Timer className="size-3.5 text-muted-foreground" />
                        {s.sla_template_name}
                      </span>
                    ) : (
                      <span className="text-muted-foreground">—</span>
                    )}
                  </td>
                  <td className="px-4 py-3 font-mono text-xs text-muted-foreground">
                    {s.billing_unit === "per_node" ? s.node_count : "cluster"}
                  </td>
                  <td className="px-4 py-3">
                    <Badge
                      tone={
                        s.status === "active"
                          ? s.is_expired
                            ? "amber"
                            : "green"
                          : "slate"
                      }
                    >
                      {s.is_expired && s.status === "active" ? "expired" : s.status}
                    </Badge>
                  </td>
                  <td className="px-4 py-3 text-muted-foreground">
                    {s.expires_at ? relativeTime(s.expires_at) : "—"}
                  </td>
                </tr>
              ))
            ) : (
              <tr>
                <td
                  colSpan={6}
                  className="px-4 py-12 text-center text-sm text-muted-foreground"
                >
                  {isLoading
                    ? "Loading…"
                    : "No subscriptions. This customer has no entitled products or SLA yet."}
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </Card>
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
      <div className="w-full space-y-4">
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-48 w-full" />
      </div>
    );
  }

  if (!customer) {
    return (
      <div className="w-full py-20 text-center text-muted-foreground">
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
    <div ref={ref} className="w-full">
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
            <div className="mb-3 flex items-center justify-between gap-2">
              <h2 className="flex items-center gap-2 text-sm font-semibold">
                <Users className="size-4 text-muted-foreground" />
                Contacts{" "}
                <span className="font-mono text-xs text-muted-foreground">
                  ({users?.length ?? 0})
                </span>
              </h2>
              {customerId && (
                <AddContactDialog
                  customerId={customerId}
                  customerName={customer?.name}
                />
              )}
            </div>
            <Card className="overflow-hidden">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border text-left font-mono text-[11px] uppercase tracking-wider text-muted-foreground">
                    <th className="px-4 py-3 font-medium">User</th>
                    <th className="px-4 py-3 font-medium">Email</th>
                    <th className="px-4 py-3 font-medium">Role</th>
                    <th className="px-4 py-3 font-medium">Active</th>
                    <th className="px-4 py-3 text-right font-medium">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {users && users.length > 0 && customerId != null ? (
                    users.map((u) => (
                      <ContactRow key={u.id} user={u} customerId={customerId} />
                    ))
                  ) : (
                    <tr>
                      <td
                        colSpan={5}
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

          {customerId != null && <SubscriptionsSection customerId={customerId} />}
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
