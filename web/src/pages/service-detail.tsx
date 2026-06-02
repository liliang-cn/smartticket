import { useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { ArrowLeft, Pencil, Trash2, Power, PowerOff, Package } from "lucide-react";
import { Trans, useTranslation } from "react-i18next";
import { toast } from "sonner";
import {
  useService,
  useDeleteService,
  useActivateService,
  useDeactivateService,
} from "@/features/services/api";
import { ServiceFormDialog } from "@/features/services/service-form-dialog";
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

// toList normalizes a field the API may return as a string[], a comma/JSON
// string, or null/undefined into a clean string[]. Prevents a white-screen when
// the backend sends null/strings where the UI expects an array.
function toList(v: unknown): string[] {
  if (Array.isArray(v)) return v.map(String);
  if (typeof v === "string" && v.trim()) {
    const s = v.trim();
    if (s.startsWith("[")) {
      try {
        const arr = JSON.parse(s);
        if (Array.isArray(arr)) return arr.map(String);
      } catch {
        /* fall through to comma-split */
      }
    }
    return s.split(",").map((x) => x.trim()).filter(Boolean);
  }
  return [];
}

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

export function ServiceDetailPage() {
  const { t } = useTranslation("services");
  const { id } = useParams();
  const navigate = useNavigate();
  const serviceId = id ? Number(id) : undefined;
  const { data: service, isLoading } = useService(serviceId);
  const del = useDeleteService();
  const activate = useActivateService();
  const deactivate = useDeactivateService();
  const [confirmOpen, setConfirmOpen] = useState(false);
  const ref = useReveal(service?.id);

  const isActive = service?.status === "active";

  async function onDelete() {
    if (serviceId == null) return;
    try {
      await del.mutateAsync(serviceId);
      toast.success(t("detail.toast_deleted"));
      setConfirmOpen(false);
      navigate("/services");
    } catch (err) {
      toast.error(apiError(err, t("detail.toast_delete_error")));
    }
  }

  async function onToggleActive() {
    if (serviceId == null) return;
    try {
      if (isActive) {
        await deactivate.mutateAsync(serviceId);
        toast.success(t("detail.toast_deactivated"));
      } else {
        await activate.mutateAsync(serviceId);
        toast.success(t("detail.toast_activated"));
      }
    } catch (err) {
      toast.error(apiError(err, t("detail.toast_toggle_error")));
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

  if (!service) {
    return (
      <div className="w-full py-20 text-center text-muted-foreground">
        {t("detail.not_found")}
        <div className="mt-4">
          <Button variant="secondary" asChild>
            <Link to="/services">
              <ArrowLeft /> {t("detail.back_to_services")}
            </Link>
          </Button>
        </div>
      </div>
    );
  }

  const togglePending = activate.isPending || deactivate.isPending;

  return (
    <div ref={ref} className="w-full">
      <Link
        to="/services"
        className="mb-4 inline-flex items-center gap-1.5 text-sm text-muted-foreground transition-colors hover:text-foreground"
      >
        <ArrowLeft className="size-4" /> {t("detail.back")}
      </Link>

      <div data-reveal className="mb-6 flex flex-wrap items-start justify-between gap-4">
        <div>
          <div className="flex items-center gap-2">
            {service.code && (
              <span className="font-mono text-xs text-primary/80">
                {service.code}
              </span>
            )}
            <Badge tone={isActive ? "green" : "slate"}>
              {service.status || "—"}
            </Badge>
            {service.type && <Badge tone="neutral">{service.type}</Badge>}
          </div>
          <h1 className="mt-1 text-2xl">{service.name}</h1>
        </div>
        <div className="flex gap-2">
          <ServiceFormDialog
            service={service}
            trigger={
              <Button variant="secondary" size="sm">
                <Pencil /> {t("detail.edit")}
              </Button>
            }
          />
          <Button
            variant="secondary"
            size="sm"
            onClick={onToggleActive}
            disabled={togglePending}
          >
            {isActive ? (
              <>
                <PowerOff /> {t("detail.deactivate")}
              </>
            ) : (
              <>
                <Power /> {t("detail.activate")}
              </>
            )}
          </Button>
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
            <Label>{t("detail.section_description")}</Label>
            <p className="mt-2 whitespace-pre-wrap text-sm leading-relaxed text-foreground/90">
              {service.description || t("detail.no_description")}
            </p>
          </Card>

          <Card data-reveal className="p-5">
            <Label>{t("detail.section_support_channels")}</Label>
            {toList(service.support_channels).length > 0 ? (
              <div className="mt-2 flex flex-wrap gap-1.5">
                {toList(service.support_channels).map((c) => (
                  <Badge key={c} tone="blue">
                    {c}
                  </Badge>
                ))}
              </div>
            ) : (
              <p className="mt-2 text-sm text-muted-foreground">
                {t("detail.no_support_channels")}
              </p>
            )}
          </Card>
        </div>

        {/* Meta sidebar */}
        <aside data-reveal className="space-y-4">
          <Card className="p-5">
            <MetaRow
              label={t("detail.meta_product")}
              value={
                <Link
                  to={`/products/${service.product_id}`}
                  className="inline-flex items-center gap-1.5 text-primary hover:underline"
                >
                  <Package className="size-3.5" /> #{service.product_id}
                </Link>
              }
            />
            <Separator />
            <MetaRow
              label={t("detail.meta_code")}
              value={
                service.code ? (
                  <span className="font-mono text-xs">{service.code}</span>
                ) : (
                  "—"
                )
              }
            />
            <Separator />
            <MetaRow label={t("detail.meta_type")} value={service.type || "—"} />
            <Separator />
            <MetaRow
              label={t("detail.meta_availability")}
              value={
                <span className="font-mono text-xs">
                  {service.availability || "—"}
                </span>
              }
            />
            <Separator />
            <MetaRow
              label={t("detail.meta_status")}
              value={
                <Badge tone={isActive ? "green" : "slate"}>
                  {service.status || "—"}
                </Badge>
              }
            />
            <MetaRow label={t("detail.meta_created")} value={relativeTime(service.created_at)} />
            <MetaRow label={t("detail.meta_updated")} value={relativeTime(service.updated_at)} />
          </Card>

          {toList(service.tags).length > 0 && (
            <Card className="p-5">
              <Label>{t("detail.section_tags")}</Label>
              <div className="mt-2 flex flex-wrap gap-1.5">
                {toList(service.tags).map((tag) => (
                  <Badge key={tag} tone="neutral">
                    {tag}
                  </Badge>
                ))}
              </div>
            </Card>
          )}
        </aside>
      </div>

      {/* Delete confirmation */}
      <Dialog open={confirmOpen} onOpenChange={setConfirmOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("detail.confirm_delete_title")}</DialogTitle>
            <DialogDescription asChild>
              <div>
                <Trans
                  ns="services"
                  i18nKey="detail.confirm_delete_description"
                  values={{ name: service.name }}
                  components={{ strong: <span className="font-medium text-foreground" /> }}
                />
              </div>
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
              {del.isPending ? t("detail.deleting") : t("detail.delete_service")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
