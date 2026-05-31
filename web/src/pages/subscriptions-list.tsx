import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import {
  Search,
  ChevronLeft,
  ChevronRight,
  CreditCard,
  Plus,
  Trash2,
} from "lucide-react";
import { toast } from "sonner";
import {
  useSubscriptions,
  useCreateSubscription,
  useDeleteSubscription,
  type SubscriptionFilters,
  type CreateSubscriptionInput,
} from "@/features/subscriptions/api";
import { useCustomers } from "@/features/customers/api";
import { useProducts } from "@/features/products/api";
import { useSLATemplates } from "@/features/sla/api";
import { apiError } from "@/lib/api";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Label } from "@/components/ui/label";
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

const ALL = "__all__";
const NONE = "__none__";

function shortDate(iso?: string | null): string {
  if (!iso) return "—";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "—";
  return d.toLocaleDateString();
}

function todayISODate(): string {
  return new Date().toISOString().slice(0, 10);
}

function plusOneYearISODate(): string {
  const d = new Date();
  d.setFullYear(d.getFullYear() + 1);
  return d.toISOString().slice(0, 10);
}

// ── Create dialog ─────────────────────────────────────────────────────────────

const schema = z.object({
  customer_id: z.string().min(1, "Customer is required"),
  product_id: z.string().min(1, "Product is required"),
  sla_template_id: z.string().optional(),
  plan: z.string().optional(),
  billing_unit: z.enum([
    "per_node",
    "per_cluster",
    "per_seat",
    "per_user",
    "per_agent",
    "per_device",
    "flat",
  ]),
  node_count: z.string().optional(),
  billing_period: z.enum(["annual", "monthly"]),
  starts_at: z.string().min(1, "Start date is required"),
  expires_at: z.string().min(1, "Expiry date is required"),
  unit_price: z.string().optional(),
  currency: z.string().optional(),
  notes: z.string().optional(),
});
type FormValues = z.infer<typeof schema>;

function defaults(): FormValues {
  return {
    customer_id: "",
    product_id: "",
    sla_template_id: "",
    plan: "",
    billing_unit: "per_node",
    node_count: "3",
    billing_period: "annual",
    starts_at: todayISODate(),
    expires_at: plusOneYearISODate(),
    unit_price: "",
    currency: "USD",
    notes: "",
  };
}

function SubscriptionFormDialog() {
  const [open, setOpen] = useState(false);
  const create = useCreateSubscription();

  // Selectors. Pull a generous page so the selects are usable.
  const customers = useCustomers({ page: 1, page_size: 100 });
  const products = useProducts({ page: 1, page_size: 100 });
  const slaTemplates = useSLATemplates({ page: 1, page_size: 100 });

  const {
    register,
    handleSubmit,
    setValue,
    watch,
    reset,
    formState: { errors },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: defaults(),
  });

  useEffect(() => {
    if (open) reset(defaults());
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  async function onSubmit(values: FormValues) {
    const payload: CreateSubscriptionInput = {
      customer_id: Number(values.customer_id),
      product_id: Number(values.product_id),
      billing_unit: values.billing_unit,
      billing_period: values.billing_period,
      starts_at: new Date(values.starts_at).toISOString(),
      expires_at: new Date(values.expires_at).toISOString(),
      status: "active",
      currency: values.currency?.trim() || "USD",
    };
    if (values.sla_template_id && values.sla_template_id !== NONE) {
      payload.sla_template_id = Number(values.sla_template_id);
    }
    if (values.plan && values.plan.trim()) payload.plan = values.plan.trim();
    if (values.node_count) {
      const n = Number(values.node_count);
      if (!Number.isNaN(n)) payload.node_count = n;
    }
    if (values.unit_price) {
      const p = Number(values.unit_price);
      if (!Number.isNaN(p)) payload.unit_price = p;
    }
    if (values.notes && values.notes.trim()) payload.notes = values.notes.trim();

    try {
      await create.mutateAsync(payload);
      toast.success("Subscription created");
      setOpen(false);
    } catch (err) {
      toast.error(apiError(err, "Could not create subscription"));
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button>
          <Plus /> New subscription
        </Button>
      </DialogTrigger>
      <DialogContent className="max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>New subscription</DialogTitle>
          <DialogDescription>
            Record a customer's support subscription for a product. Per-node and
            annual terms are the defaults.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label>Customer</Label>
              <Select
                value={watch("customer_id")}
                onValueChange={(v) => setValue("customer_id", v)}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Select customer" />
                </SelectTrigger>
                <SelectContent>
                  {customers.data?.items.map((c) => (
                    <SelectItem key={c.id} value={String(c.id)}>
                      {c.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              {errors.customer_id && (
                <p className="text-xs text-destructive">
                  {errors.customer_id.message}
                </p>
              )}
            </div>
            <div className="space-y-1.5">
              <Label>Product</Label>
              <Select
                value={watch("product_id")}
                onValueChange={(v) => setValue("product_id", v)}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Select product" />
                </SelectTrigger>
                <SelectContent>
                  {products.data?.items.map((p) => (
                    <SelectItem key={p.id} value={String(p.id)}>
                      {p.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              {errors.product_id && (
                <p className="text-xs text-destructive">
                  {errors.product_id.message}
                </p>
              )}
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label>SLA template</Label>
              <Select
                value={watch("sla_template_id") || NONE}
                onValueChange={(v) =>
                  setValue("sla_template_id", v === NONE ? "" : v)
                }
              >
                <SelectTrigger>
                  <SelectValue placeholder="Optional" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value={NONE}>None</SelectItem>
                  {slaTemplates.data?.items.map((t) => (
                    <SelectItem key={t.id} value={String(t.id)}>
                      {t.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="s-plan">Plan</Label>
              <Input id="s-plan" placeholder="e.g. Standard" {...register("plan")} />
            </div>
          </div>

          <div className="grid grid-cols-3 gap-4">
            <div className="space-y-1.5">
              <Label>Billing unit</Label>
              <Select
                value={watch("billing_unit")}
                onValueChange={(v) =>
                  setValue("billing_unit", v as FormValues["billing_unit"])
                }
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="per_node">Per node</SelectItem>
                  <SelectItem value="per_cluster">Per cluster</SelectItem>
                  <SelectItem value="per_seat">Per seat</SelectItem>
                  <SelectItem value="per_user">Per user</SelectItem>
                  <SelectItem value="per_agent">Per agent</SelectItem>
                  <SelectItem value="per_device">Per device</SelectItem>
                  <SelectItem value="flat">Flat</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="s-nodes">Node count</Label>
              <Input
                id="s-nodes"
                type="number"
                placeholder="3"
                {...register("node_count")}
              />
            </div>
            <div className="space-y-1.5">
              <Label>Billing period</Label>
              <Select
                value={watch("billing_period")}
                onValueChange={(v) =>
                  setValue("billing_period", v as FormValues["billing_period"])
                }
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="annual">Annual</SelectItem>
                  <SelectItem value="monthly">Monthly</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label htmlFor="s-start">Starts at</Label>
              <Input id="s-start" type="date" {...register("starts_at")} />
              {errors.starts_at && (
                <p className="text-xs text-destructive">
                  {errors.starts_at.message}
                </p>
              )}
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="s-expire">Expires at</Label>
              <Input id="s-expire" type="date" {...register("expires_at")} />
              {errors.expires_at && (
                <p className="text-xs text-destructive">
                  {errors.expires_at.message}
                </p>
              )}
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label htmlFor="s-price">Unit price</Label>
              <Input
                id="s-price"
                type="number"
                step="0.01"
                placeholder="optional"
                {...register("unit_price")}
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="s-currency">Currency</Label>
              <Input id="s-currency" placeholder="USD" {...register("currency")} />
            </div>
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="s-notes">Notes</Label>
            <Input id="s-notes" placeholder="optional" {...register("notes")} />
          </div>

          <DialogFooter>
            <DialogClose asChild>
              <Button type="button" variant="ghost">
                Cancel
              </Button>
            </DialogClose>
            <Button type="submit" disabled={create.isPending}>
              {create.isPending ? "Creating…" : "Create subscription"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

// ── Page ──────────────────────────────────────────────────────────────────────

export function SubscriptionsPage() {
  const [filters, setFilters] = useState<SubscriptionFilters>({
    page: 1,
    page_size: 15,
  });
  const { data, isLoading, isFetching } = useSubscriptions(filters);
  const deleteSubscription = useDeleteSubscription();

  const [toDelete, setToDelete] = useState<{ id: number; label: string } | null>(
    null
  );

  const set = (patch: Partial<SubscriptionFilters>) =>
    setFilters((f) => ({ ...f, page: 1, ...patch }));

  async function confirmDelete() {
    if (!toDelete) return;
    try {
      await deleteSubscription.mutateAsync(toDelete.id);
      toast.success("Subscription deleted");
      setToDelete(null);
    } catch (err) {
      toast.error(apiError(err, "Could not delete subscription"));
    }
  }

  return (
    <div className="w-full">
      <div className="mb-6 flex flex-wrap items-end justify-between gap-4">
        <div>
          <div className="font-mono text-xs uppercase tracking-[0.25em] text-muted-foreground">
            commercial
          </div>
          <h1 className="mt-1 text-3xl">Subscriptions</h1>
        </div>
        <SubscriptionFormDialog />
      </div>

      {/* Filter bar */}
      <div className="mb-4 flex flex-wrap items-center gap-3">
        <div className="relative min-w-56 flex-1">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="pl-9"
            type="number"
            placeholder="Filter by customer ID…"
            value={filters.customer_id ?? ""}
            onChange={(e) =>
              set({
                customer_id: e.target.value
                  ? Number(e.target.value)
                  : undefined,
              })
            }
          />
        </div>
        <Select
          value={filters.status ?? ALL}
          onValueChange={(v) => set({ status: v === ALL ? undefined : v })}
        >
          <SelectTrigger className="w-40">
            <SelectValue placeholder="Status" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={ALL}>All statuses</SelectItem>
            <SelectItem value="active">Active</SelectItem>
            <SelectItem value="expired">Expired</SelectItem>
            <SelectItem value="cancelled">Cancelled</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <Card className="overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border text-left font-mono text-[11px] uppercase tracking-wider text-muted-foreground">
              <th className="px-4 py-3 font-medium">Customer</th>
              <th className="px-4 py-3 font-medium">Product</th>
              <th className="px-4 py-3 font-medium">Plan</th>
              <th className="px-4 py-3 font-medium">Nodes</th>
              <th className="px-4 py-3 font-medium">Term</th>
              <th className="px-4 py-3 font-medium">Period</th>
              <th className="px-4 py-3 font-medium">SLA</th>
              <th className="px-4 py-3 font-medium">Status</th>
              <th className="px-4 py-3" />
            </tr>
          </thead>
          <tbody>
            {isLoading ? (
              Array.from({ length: 6 }).map((_, i) => (
                <tr key={i} className="border-b border-border/60">
                  {Array.from({ length: 9 }).map((__, j) => (
                    <td key={j} className="px-4 py-3.5">
                      <Skeleton className="h-4 w-full" />
                    </td>
                  ))}
                </tr>
              ))
            ) : data && data.items.length > 0 ? (
              data.items.map((s) => {
                const expired = s.is_expired || s.status === "expired";
                return (
                  <tr
                    key={s.id}
                    className="border-b border-border/60 transition-colors last:border-0 hover:bg-accent/50"
                  >
                    <td className="px-4 py-3.5 font-medium text-foreground">
                      {s.customer_name || `#${s.customer_id}`}
                    </td>
                    <td className="px-4 py-3.5 text-muted-foreground">
                      {s.product_name || `#${s.product_id}`}
                    </td>
                    <td className="px-4 py-3.5 text-muted-foreground">
                      {s.plan || "—"}
                    </td>
                    <td className="px-4 py-3.5 font-mono text-xs text-muted-foreground">
                      {s.total_units}
                      <span className="text-muted-foreground/60">
                        {" "}
                        {s.billing_unit.replace("per_", "/").replace("flat", "flat")}
                      </span>
                    </td>
                    <td className="px-4 py-3.5 font-mono text-xs text-muted-foreground">
                      {s.billing_period}
                    </td>
                    <td className="px-4 py-3.5 font-mono text-xs text-muted-foreground">
                      {shortDate(s.starts_at)} → {shortDate(s.expires_at)}
                    </td>
                    <td className="px-4 py-3.5 text-muted-foreground">
                      {s.sla_template_name || "—"}
                    </td>
                    <td className="px-4 py-3.5">
                      {s.status === "cancelled" ? (
                        <Badge tone="slate">cancelled</Badge>
                      ) : expired ? (
                        <Badge tone="red">expired</Badge>
                      ) : (
                        <Badge tone="green">active</Badge>
                      )}
                    </td>
                    <td className="px-2 py-3.5">
                      <div className="flex items-center justify-end">
                        <Button
                          variant="ghost"
                          size="icon"
                          title="Delete subscription"
                          onClick={() =>
                            setToDelete({
                              id: s.id,
                              label: `${s.customer_name || s.customer_id} · ${
                                s.product_name || s.product_id
                              }`,
                            })
                          }
                        >
                          <Trash2 />
                        </Button>
                      </div>
                    </td>
                  </tr>
                );
              })
            ) : (
              <tr>
                <td colSpan={9} className="px-4 py-16 text-center">
                  <CreditCard className="mx-auto size-8 text-muted-foreground/40" />
                  <p className="mt-3 text-sm text-muted-foreground">
                    No subscriptions match these filters.
                  </p>
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </Card>

      {/* Pagination */}
      {data && data.total_pages > 1 && (
        <div className="mt-4 flex items-center justify-between">
          <div className="font-mono text-xs text-muted-foreground">
            {data.total} subscriptions · page {data.page}/{data.total_pages}
            {isFetching && " · syncing…"}
          </div>
          <div className="flex gap-2">
            <Button
              variant="secondary"
              size="sm"
              disabled={filters.page <= 1}
              onClick={() => setFilters((f) => ({ ...f, page: f.page - 1 }))}
            >
              <ChevronLeft /> Prev
            </Button>
            <Button
              variant="secondary"
              size="sm"
              disabled={data.page >= data.total_pages}
              onClick={() => setFilters((f) => ({ ...f, page: f.page + 1 }))}
            >
              Next <ChevronRight />
            </Button>
          </div>
        </div>
      )}

      <Dialog
        open={toDelete != null}
        onOpenChange={(v) => !v && setToDelete(null)}
      >
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>Delete subscription?</DialogTitle>
            <DialogDescription>
              This permanently removes the subscription for "{toDelete?.label}".
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <DialogClose asChild>
              <Button type="button" variant="ghost">
                Cancel
              </Button>
            </DialogClose>
            <Button
              type="button"
              variant="destructive"
              disabled={deleteSubscription.isPending}
              onClick={confirmDelete}
            >
              {deleteSubscription.isPending ? "Deleting…" : "Delete"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
