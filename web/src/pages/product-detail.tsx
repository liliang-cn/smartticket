import { useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { ArrowLeft, Pencil, Trash2, Layers, Power, PowerOff } from "lucide-react";
import { toast } from "sonner";
import {
  useProduct,
  useDeleteProduct,
  useActivateProduct,
  useDeactivateProduct,
} from "@/features/products/api";
import { ProductFormDialog } from "@/features/products/product-form-dialog";
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

export function ProductDetailPage() {
  const { id } = useParams();
  const navigate = useNavigate();
  const productId = id ? Number(id) : undefined;
  const { data: product, isLoading } = useProduct(productId);
  const del = useDeleteProduct();
  const activate = useActivateProduct();
  const deactivate = useDeactivateProduct();
  const [confirmOpen, setConfirmOpen] = useState(false);
  const ref = useReveal(product?.id);

  const isActive = product?.status === "active";
  const services = product?.services ?? [];

  async function onDelete() {
    if (productId == null) return;
    try {
      await del.mutateAsync(productId);
      toast.success("Product deleted");
      setConfirmOpen(false);
      navigate("/products");
    } catch (err) {
      toast.error(apiError(err, "Could not delete product"));
    }
  }

  async function onToggleActive() {
    if (productId == null) return;
    try {
      if (isActive) {
        await deactivate.mutateAsync(productId);
        toast.success("Product deactivated");
      } else {
        await activate.mutateAsync(productId);
        toast.success("Product activated");
      }
    } catch (err) {
      toast.error(apiError(err, "Could not update product status"));
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

  if (!product) {
    return (
      <div className="mx-auto max-w-5xl py-20 text-center text-muted-foreground">
        Product not found.
        <div className="mt-4">
          <Button variant="secondary" asChild>
            <Link to="/products">
              <ArrowLeft /> Back to products
            </Link>
          </Button>
        </div>
      </div>
    );
  }

  const togglePending = activate.isPending || deactivate.isPending;

  return (
    <div ref={ref} className="mx-auto max-w-5xl">
      <Link
        to="/products"
        className="mb-4 inline-flex items-center gap-1.5 text-sm text-muted-foreground transition-colors hover:text-foreground"
      >
        <ArrowLeft className="size-4" /> Products
      </Link>

      <div data-reveal className="mb-6 flex flex-wrap items-start justify-between gap-4">
        <div>
          <div className="flex items-center gap-2">
            {product.code && (
              <span className="font-mono text-xs text-primary/80">
                {product.code}
              </span>
            )}
            <Badge tone={isActive ? "green" : "slate"}>
              {product.status || "—"}
            </Badge>
            {product.is_managed && <Badge tone="blue">managed</Badge>}
          </div>
          <h1 className="mt-1 text-2xl">{product.name}</h1>
        </div>
        <div className="flex gap-2">
          <ProductFormDialog
            product={product}
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
              {product.description || "No description provided."}
            </p>
          </Card>

          {product.documentation && (
            <Card data-reveal className="p-5">
              <Label>Documentation</Label>
              <p className="mt-2 whitespace-pre-wrap text-sm leading-relaxed text-foreground/90">
                {product.documentation}
              </p>
            </Card>
          )}

          <div data-reveal>
            <div className="mb-3 flex items-center justify-between">
              <h2 className="flex items-center gap-2 text-sm font-semibold">
                <Layers className="size-4 text-muted-foreground" />
                Services{" "}
                <span className="font-mono text-xs text-muted-foreground">
                  ({services.length})
                </span>
              </h2>
              <ServiceFormDialog
                productId={product.id}
                trigger={
                  <Button variant="secondary" size="sm">
                    Add service
                  </Button>
                }
              />
            </div>
            <Card className="overflow-hidden">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border text-left font-mono text-[11px] uppercase tracking-wider text-muted-foreground">
                    <th className="px-4 py-3 font-medium">Name</th>
                    <th className="px-4 py-3 font-medium">Code</th>
                    <th className="px-4 py-3 font-medium">Type</th>
                    <th className="px-4 py-3 font-medium">Status</th>
                  </tr>
                </thead>
                <tbody>
                  {services.length > 0 ? (
                    services.map((s) => (
                      <tr
                        key={s.id}
                        onClick={() => navigate(`/services/${s.id}`)}
                        className="group cursor-pointer border-b border-border/60 transition-colors last:border-0 hover:bg-accent/50"
                      >
                        <td className="px-4 py-3 font-medium group-hover:text-primary">
                          {s.name}
                        </td>
                        <td className="px-4 py-3 font-mono text-xs text-muted-foreground">
                          {s.code || "—"}
                        </td>
                        <td className="px-4 py-3 text-muted-foreground">
                          {s.type || "—"}
                        </td>
                        <td className="px-4 py-3">
                          <Badge tone={s.status === "active" ? "green" : "slate"}>
                            {s.status || "—"}
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
                        No services defined for this product.
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
                product.code ? (
                  <span className="font-mono text-xs">{product.code}</span>
                ) : (
                  "—"
                )
              }
            />
            <Separator />
            <MetaRow label="Category" value={product.category || "—"} />
            <Separator />
            <MetaRow
              label="Version"
              value={
                <span className="font-mono text-xs">{product.version || "—"}</span>
              }
            />
            <Separator />
            <MetaRow label="Support level" value={product.support_level || "—"} />
            <Separator />
            <MetaRow
              label="Managed"
              value={
                <Badge tone={product.is_managed ? "blue" : "slate"}>
                  {product.is_managed ? "managed" : "unmanaged"}
                </Badge>
              }
            />
            <Separator />
            <MetaRow
              label="Status"
              value={
                <Badge tone={isActive ? "green" : "slate"}>
                  {product.status || "—"}
                </Badge>
              }
            />
            <MetaRow label="Created" value={relativeTime(product.created_at)} />
            <MetaRow label="Updated" value={relativeTime(product.updated_at)} />
          </Card>

          {product.tags.length > 0 && (
            <Card className="p-5">
              <Label>Tags</Label>
              <div className="mt-2 flex flex-wrap gap-1.5">
                {product.tags.map((t) => (
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
            <DialogTitle>Delete product</DialogTitle>
            <DialogDescription>
              Delete{" "}
              <span className="font-medium text-foreground">{product.name}</span>?
              This removes the product from the catalog.
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
              {del.isPending ? "Deleting…" : "Delete product"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
