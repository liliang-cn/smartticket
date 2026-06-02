import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
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
import { formatDate } from "@/lib/utils";
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

function todayISODate(): string {
  return new Date().toISOString().slice(0, 10);
}

function plusOneYearISODate(): string {
  const d = new Date();
  d.setFullYear(d.getFullYear() + 1);
  return d.toISOString().slice(0, 10);
}

// ── Create dialog ─────────────────────────────────────────────────────────────

// BILLING_UNITS is the single source of truth for the form's unit options. It
// mirrors the backend's billingUnits set (internal/subscription). `single` marks
// units that always bill as one unit (so the quantity field is hidden).
export const BILLING_UNITS: { value: string; label: string; single?: boolean }[] = [
  { value: "per_node", label: "Per node" },
  { value: "per_cluster", label: "Per cluster", single: true },
  { value: "per_core", label: "Per core" },
  { value: "per_instance", label: "Per instance" },
  { value: "per_seat", label: "Per seat" },
  { value: "per_user", label: "Per user" },
  { value: "per_agent", label: "Per agent" },
  { value: "per_device", label: "Per device" },
  { value: "per_site", label: "Per site" },
  { value: "per_gb", label: "Per GB" },
  { value: "per_request", label: "Per request" },
  { value: "usage", label: "Usage / metered" },
  { value: "per_subscriber", label: "Per subscriber (app)" },
  { value: "per_install", label: "Per install" },
  { value: "per_app", label: "Per app" },
  { value: "flat", label: "Flat fee", single: true },
];
const SINGLE_UNITS = new Set(BILLING_UNITS.filter((u) => u.single).map((u) => u.value));

// Billing cadences. Includes weekly + lifetime (one-time) for indie app subs.
export const BILLING_PERIODS: { value: string; label: string }[] = [
  { value: "weekly", label: "Weekly" },
  { value: "monthly", label: "Monthly" },
  { value: "annual", label: "Annual" },
  { value: "lifetime", label: "Lifetime (one-time)" },
];

// Validation messages are resolved at submit-time via t(); the static schema
// uses English fallbacks so Zod can construct without a hook context.
const schema = z.object({
  customer_id: z.string().min(1, "Customer is required"),
  product_id: z.string().min(1, "Product is required"),
  sla_template_id: z.string().optional(),
  plan: z.string().optional(),
  billing_unit: z.string().min(1, "Billing unit is required"),
  node_count: z.string().optional(),
  billing_period: z.string().min(1, "Billing period is required"),
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
  const { t } = useTranslation("subscriptions");
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
      toast.success(t("toast.created"));
      setOpen(false);
    } catch (err) {
      toast.error(apiError(err, t("toast.create_failed")));
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button>
          <Plus /> {t("actions.new")}
        </Button>
      </DialogTrigger>
      <DialogContent className="max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{t("form.title")}</DialogTitle>
          <DialogDescription>{t("form.description")}</DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label>{t("form.customer")}</Label>
              <Select
                value={watch("customer_id")}
                onValueChange={(v) => setValue("customer_id", v)}
              >
                <SelectTrigger>
                  <SelectValue placeholder={t("form.customer_placeholder")} />
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
                  {t("validation.customer_required")}
                </p>
              )}
            </div>
            <div className="space-y-1.5">
              <Label>{t("form.product")}</Label>
              <Select
                value={watch("product_id")}
                onValueChange={(v) => setValue("product_id", v)}
              >
                <SelectTrigger>
                  <SelectValue placeholder={t("form.product_placeholder")} />
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
                  {t("validation.product_required")}
                </p>
              )}
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label>{t("form.sla_template")}</Label>
              <Select
                value={watch("sla_template_id") || NONE}
                onValueChange={(v) =>
                  setValue("sla_template_id", v === NONE ? "" : v)
                }
              >
                <SelectTrigger>
                  <SelectValue placeholder={t("form.sla_placeholder")} />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value={NONE}>{t("form.sla_none")}</SelectItem>
                  {slaTemplates.data?.items.map((tmpl) => (
                    <SelectItem key={tmpl.id} value={String(tmpl.id)}>
                      {tmpl.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="s-plan">{t("form.plan")}</Label>
              <Input id="s-plan" placeholder={t("form.plan_placeholder")} {...register("plan")} />
            </div>
          </div>

          <div className="grid grid-cols-3 gap-4">
            <div className="space-y-1.5">
              <Label>{t("form.billing_unit")}</Label>
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
                  {BILLING_UNITS.map((u) => (
                    <SelectItem key={u.value} value={u.value}>
                      {t(`billing_unit.${u.value}`)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            {!SINGLE_UNITS.has(watch("billing_unit")) && (
              <div className="space-y-1.5">
                <Label htmlFor="s-nodes">{t("form.quantity")}</Label>
                <Input
                  id="s-nodes"
                  type="number"
                  placeholder="3"
                  {...register("node_count")}
                />
              </div>
            )}
            <div className="space-y-1.5">
              <Label>{t("form.billing_period")}</Label>
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
                  {BILLING_PERIODS.map((p) => (
                    <SelectItem key={p.value} value={p.value}>
                      {t(`billing_period.${p.value}`)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label htmlFor="s-start">{t("form.starts_at")}</Label>
              <Input id="s-start" type="date" {...register("starts_at")} />
              {errors.starts_at && (
                <p className="text-xs text-destructive">
                  {t("validation.starts_at_required")}
                </p>
              )}
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="s-expire">{t("form.expires_at")}</Label>
              <Input id="s-expire" type="date" {...register("expires_at")} />
              {errors.expires_at && (
                <p className="text-xs text-destructive">
                  {t("validation.expires_at_required")}
                </p>
              )}
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label htmlFor="s-price">{t("form.unit_price")}</Label>
              <Input
                id="s-price"
                type="number"
                step="0.01"
                placeholder={t("form.unit_price_placeholder")}
                {...register("unit_price")}
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="s-currency">{t("form.currency")}</Label>
              <Input id="s-currency" placeholder="USD" {...register("currency")} />
            </div>
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="s-notes">{t("notes.label")}</Label>
            <Input id="s-notes" placeholder={t("notes.placeholder")} {...register("notes")} />
          </div>

          <DialogFooter>
            <DialogClose asChild>
              <Button type="button" variant="ghost">
                {t("actions.cancel", { ns: "common" })}
              </Button>
            </DialogClose>
            <Button type="submit" disabled={create.isPending}>
              {create.isPending ? t("actions.creating") : t("actions.create")}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

// ── Page ──────────────────────────────────────────────────────────────────────

export function SubscriptionsPage() {
  const { t } = useTranslation("subscriptions");
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
      toast.success(t("toast.deleted"));
      setToDelete(null);
    } catch (err) {
      toast.error(apiError(err, t("toast.delete_failed")));
    }
  }

  return (
    <div className="w-full">
      <div className="mb-6 flex flex-wrap items-end justify-between gap-4">
        <div>
          <div className="font-mono text-xs uppercase tracking-[0.25em] text-muted-foreground">
            {t("page.section")}
          </div>
          <h1 className="mt-1 text-3xl">{t("page.title")}</h1>
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
            placeholder={t("filter.customer_id_placeholder")}
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
            <SelectValue placeholder={t("filter.status_placeholder")} />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={ALL}>{t("filter.all_statuses")}</SelectItem>
            <SelectItem value="active">{t("status.active")}</SelectItem>
            <SelectItem value="expired">{t("status.expired")}</SelectItem>
            <SelectItem value="cancelled">{t("status.cancelled")}</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <Card className="overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border text-left font-mono text-[11px] uppercase tracking-wider text-muted-foreground">
              <th className="px-4 py-3 font-medium">{t("table.customer")}</th>
              <th className="px-4 py-3 font-medium">{t("table.product")}</th>
              <th className="px-4 py-3 font-medium">{t("table.plan")}</th>
              <th className="px-4 py-3 font-medium">{t("table.nodes")}</th>
              <th className="px-4 py-3 font-medium">{t("table.term")}</th>
              <th className="px-4 py-3 font-medium">{t("table.period")}</th>
              <th className="px-4 py-3 font-medium">{t("table.sla")}</th>
              <th className="px-4 py-3 font-medium">{t("table.status")}</th>
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
                      {t(`billing_period.${s.billing_period}`, { defaultValue: s.billing_period })}
                    </td>
                    <td className="px-4 py-3.5 font-mono text-xs text-muted-foreground">
                      {formatDate(s.starts_at) || "—"} → {formatDate(s.expires_at) || "—"}
                    </td>
                    <td className="px-4 py-3.5 text-muted-foreground">
                      {s.sla_template_name || "—"}
                    </td>
                    <td className="px-4 py-3.5">
                      {s.status === "cancelled" ? (
                        <Badge tone="slate">{t("status.cancelled")}</Badge>
                      ) : expired ? (
                        <Badge tone="red">{t("status.expired")}</Badge>
                      ) : (
                        <Badge tone="green">{t("status.active")}</Badge>
                      )}
                    </td>
                    <td className="px-2 py-3.5">
                      <div className="flex items-center justify-end">
                        <Button
                          variant="ghost"
                          size="icon"
                          title={t("actions.delete")}
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
                    {t("empty.message")}
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
            {t("pagination.summary", {
              total: data.total,
              page: data.page,
              totalPages: data.total_pages,
            })}
            {isFetching && t("pagination.syncing")}
          </div>
          <div className="flex gap-2">
            <Button
              variant="secondary"
              size="sm"
              disabled={filters.page <= 1}
              onClick={() => setFilters((f) => ({ ...f, page: f.page - 1 }))}
            >
              <ChevronLeft /> {t("pagination.prev")}
            </Button>
            <Button
              variant="secondary"
              size="sm"
              disabled={data.page >= data.total_pages}
              onClick={() => setFilters((f) => ({ ...f, page: f.page + 1 }))}
            >
              {t("pagination.next")} <ChevronRight />
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
            <DialogTitle>{t("confirm_delete.title")}</DialogTitle>
            <DialogDescription>
              {t("confirm_delete.description", { label: toDelete?.label })}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <DialogClose asChild>
              <Button type="button" variant="ghost">
                {t("actions.cancel", { ns: "common" })}
              </Button>
            </DialogClose>
            <Button
              type="button"
              variant="destructive"
              disabled={deleteSubscription.isPending}
              onClick={confirmDelete}
            >
              {deleteSubscription.isPending
                ? t("actions.deleting")
                : t("actions.delete", { ns: "common" })}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
