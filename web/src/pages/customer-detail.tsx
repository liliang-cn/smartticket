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
import { useTranslation, Trans } from "react-i18next";
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
  const { t } = useTranslation("customers");
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
      toast.success(
        user.is_active
          ? t("contact_row.toast.deactivated")
          : t("contact_row.toast.activated"),
      );
    } catch (err) {
      toast.error(apiError(err));
    }
  }

  async function onRemove() {
    try {
      await del.mutateAsync(user.id);
      refreshContacts();
      setConfirmOpen(false);
      toast.success(t("contact_row.toast.removed"));
    } catch (err) {
      toast.error(apiError(err, t("contact_row.toast.remove_error")));
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
          {user.is_active ? t("status.active") : t("status.inactive")}
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
                <UserX /> {t("contact_row.deactivate")}
              </>
            ) : (
              <>
                <UserCheck /> {t("contact_row.activate")}
              </>
            )}
          </Button>
          <Button
            size="icon"
            variant="ghost"
            className="text-destructive hover:text-destructive"
            onClick={() => setConfirmOpen(true)}
            aria-label={t("contact_row.remove_aria", { name: displayName })}
          >
            <Trash2 />
          </Button>
        </div>

        <Dialog open={confirmOpen} onOpenChange={setConfirmOpen}>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>{t("contact_row.remove_dialog.title")}</DialogTitle>
              <DialogDescription asChild>
                <p>
                  <Trans
                    ns="customers"
                    i18nKey="contact_row.remove_dialog.description"
                    values={{ name: displayName }}
                    components={{ bold: <span className="font-medium text-foreground" /> }}
                  />
                </p>
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
                onClick={onRemove}
                disabled={del.isPending}
              >
                {del.isPending
                  ? t("contact_row.remove_dialog.removing")
                  : t("contact_row.remove_dialog.confirm")}
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
  const { t } = useTranslation("customers");
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
          {t("subscriptions.heading")}{" "}
          <span className="font-mono text-xs text-muted-foreground">
            ({subs.length})
          </span>
        </h2>
      </div>
      <Card className="overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border text-left font-mono text-[11px] uppercase tracking-wider text-muted-foreground">
              <th className="px-4 py-3 font-medium">{t("subscriptions.col_product")}</th>
              <th className="px-4 py-3 font-medium">{t("subscriptions.col_plan")}</th>
              <th className="px-4 py-3 font-medium">{t("subscriptions.col_sla")}</th>
              <th className="px-4 py-3 font-medium">{t("subscriptions.col_nodes")}</th>
              <th className="px-4 py-3 font-medium">{t("subscriptions.col_status")}</th>
              <th className="px-4 py-3 font-medium">{t("subscriptions.col_expires")}</th>
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
                    {s.billing_unit === "per_node" ? s.node_count : t("subscriptions.cluster")}
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
                    ? t("subscriptions.empty_loading")
                    : t("subscriptions.empty")}
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
  const { t } = useTranslation("customers");
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
      toast.success(t("detail.toast.deleted"));
      setConfirmOpen(false);
      navigate("/customers");
    } catch (err) {
      toast.error(apiError(err, t("detail.toast.delete_error")));
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
        {t("detail.not_found")}
        <div className="mt-4">
          <Button variant="secondary" asChild>
            <Link to="/customers">
              <ArrowLeft /> {t("detail.back_to_customers")}
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
        <ArrowLeft className="size-4" /> {t("detail.back")}
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
              {customer.is_active ? t("status.active") : t("status.inactive")}
            </Badge>
          </div>
          <h1 className="mt-1 text-2xl">{customer.name}</h1>
        </div>
        <div className="flex gap-2">
          <CustomerFormDialog
            customer={customer}
            trigger={
              <Button variant="secondary" size="sm">
                <Pencil /> {t("detail.edit")}
              </Button>
            }
          />
          <Button
            variant="destructive"
            size="sm"
            onClick={() => setConfirmOpen(true)}
          >
            <Trash2 /> {t("detail.delete")}
          </Button>
        </div>
      </div>

      <div className="grid gap-6 lg:grid-cols-[1fr_18rem]">
        {/* Main */}
        <div className="space-y-5">
          <Card data-reveal className="p-5">
            <Label>{t("detail.description_label")}</Label>
            <p className="mt-2 whitespace-pre-wrap text-sm leading-relaxed text-foreground/90">
              {customer.description || t("detail.no_description")}
            </p>
          </Card>

          <div data-reveal>
            <div className="mb-3 flex items-center justify-between gap-2">
              <h2 className="flex items-center gap-2 text-sm font-semibold">
                <Users className="size-4 text-muted-foreground" />
                {t("detail.contacts_heading")}{" "}
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
                    <th className="px-4 py-3 font-medium">{t("detail.col_user")}</th>
                    <th className="px-4 py-3 font-medium">{t("detail.col_email")}</th>
                    <th className="px-4 py-3 font-medium">{t("detail.col_role")}</th>
                    <th className="px-4 py-3 font-medium">{t("detail.col_active")}</th>
                    <th className="px-4 py-3 text-right font-medium">{t("detail.col_actions")}</th>
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
                        {t("detail.no_contacts")}
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
              label={t("detail.meta_code")}
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
              label={t("detail.meta_domain")}
              value={
                <span className="font-mono text-xs">
                  {customer.domain || "—"}
                </span>
              }
            />
            <Separator />
            <MetaRow
              label={t("detail.meta_status")}
              value={
                <Badge tone={customer.is_active ? "green" : "slate"}>
                  {customer.is_active ? t("status.active") : t("status.inactive")}
                </Badge>
              }
            />
            <MetaRow label={t("detail.meta_created")} value={relativeTime(customer.created_at)} />
            <MetaRow label={t("detail.meta_updated")} value={relativeTime(customer.updated_at)} />
          </Card>
        </aside>
      </div>

      {/* Delete confirmation */}
      <Dialog open={confirmOpen} onOpenChange={setConfirmOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("detail.delete_dialog.title")}</DialogTitle>
            <DialogDescription asChild>
              <p>
                <Trans
                  ns="customers"
                  i18nKey="detail.delete_dialog.description"
                  values={{ name: customer.name }}
                  components={{ bold: <span className="font-medium text-foreground" /> }}
                />
              </p>
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
              {del.isPending
                ? t("detail.delete_dialog.deleting")
                : t("detail.delete_dialog.confirm")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
