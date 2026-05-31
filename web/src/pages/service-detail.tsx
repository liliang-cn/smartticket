import { useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { ArrowLeft, Pencil, Trash2, Power, PowerOff, Package } from "lucide-react";
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
      toast.success("Service deleted");
      setConfirmOpen(false);
      navigate("/services");
    } catch (err) {
      toast.error(apiError(err, "Could not delete service"));
    }
  }

  async function onToggleActive() {
    if (serviceId == null) return;
    try {
      if (isActive) {
        await deactivate.mutateAsync(serviceId);
        toast.success("Service deactivated");
      } else {
        await activate.mutateAsync(serviceId);
        toast.success("Service activated");
      }
    } catch (err) {
      toast.error(apiError(err, "Could not update service status"));
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
        Service not found.
        <div className="mt-4">
          <Button variant="secondary" asChild>
            <Link to="/services">
              <ArrowLeft /> Back to services
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
        <ArrowLeft className="size-4" /> Services
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
                <Pencil /> Edit
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
                <PowerOff /> Deactivate
              </>
            ) : (
              <>
                <Power /> Activate
              </>
            )}
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
            <Label>Description</Label>
            <p className="mt-2 whitespace-pre-wrap text-sm leading-relaxed text-foreground/90">
              {service.description || "No description provided."}
            </p>
          </Card>

          <Card data-reveal className="p-5">
            <Label>Support channels</Label>
            {service.support_channels.length > 0 ? (
              <div className="mt-2 flex flex-wrap gap-1.5">
                {service.support_channels.map((c) => (
                  <Badge key={c} tone="blue">
                    {c}
                  </Badge>
                ))}
              </div>
            ) : (
              <p className="mt-2 text-sm text-muted-foreground">
                No support channels configured.
              </p>
            )}
          </Card>
        </div>

        {/* Meta sidebar */}
        <aside data-reveal className="space-y-4">
          <Card className="p-5">
            <MetaRow
              label="Product"
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
              label="Code"
              value={
                service.code ? (
                  <span className="font-mono text-xs">{service.code}</span>
                ) : (
                  "—"
                )
              }
            />
            <Separator />
            <MetaRow label="Type" value={service.type || "—"} />
            <Separator />
            <MetaRow
              label="Availability"
              value={
                <span className="font-mono text-xs">
                  {service.availability || "—"}
                </span>
              }
            />
            <Separator />
            <MetaRow
              label="Status"
              value={
                <Badge tone={isActive ? "green" : "slate"}>
                  {service.status || "—"}
                </Badge>
              }
            />
            <MetaRow label="Created" value={relativeTime(service.created_at)} />
            <MetaRow label="Updated" value={relativeTime(service.updated_at)} />
          </Card>

          {service.tags.length > 0 && (
            <Card className="p-5">
              <Label>Tags</Label>
              <div className="mt-2 flex flex-wrap gap-1.5">
                {service.tags.map((t) => (
                  <Badge key={t} tone="neutral">
                    {t}
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
            <DialogTitle>Delete service</DialogTitle>
            <DialogDescription>
              Delete{" "}
              <span className="font-medium text-foreground">{service.name}</span>?
              This removes the service from the catalog.
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
              {del.isPending ? "Deleting…" : "Delete service"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
