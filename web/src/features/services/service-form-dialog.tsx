import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Plus } from "lucide-react";
import { toast } from "sonner";
import {
  useCreateService,
  useUpdateService,
  type CreateServiceInput,
  type Service,
} from "@/features/services/api";
import { useProducts } from "@/features/products/api";
import { apiError } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input, Textarea } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
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

const schema = z.object({
  product_id: z.number().int().positive("Product is required"),
  name: z.string().min(1, "Name is required").max(255),
  code: z.string().max(100).optional(),
  type: z.string().max(100).optional(),
  availability: z.string().max(100).optional(),
  description: z.string().optional(),
  support_channels: z.string().optional(),
  tags: z.string().optional(),
  status: z.string(),
});
type FormValues = z.infer<typeof schema>;

interface ServiceFormDialogProps {
  /** When provided, the dialog edits this service instead of creating one. */
  service?: Service;
  /** Pre-selects the parent product (e.g. when creating from a product page). */
  productId?: number;
  /** Optional custom trigger. Defaults to a "New service" button. */
  trigger?: React.ReactNode;
}

const ACTIVE = "active";
const INACTIVE = "inactive";

function listToString(values?: string[]): string {
  return (values ?? []).join(", ");
}

function stringToList(value?: string): string[] {
  return (value ?? "")
    .split(",")
    .map((t) => t.trim())
    .filter(Boolean);
}

export function ServiceFormDialog({
  service,
  productId,
  trigger,
}: ServiceFormDialogProps) {
  const [open, setOpen] = useState(false);
  const isEdit = service != null;
  const create = useCreateService();
  const update = useUpdateService(service?.id ?? 0);
  const pending = isEdit ? update.isPending : create.isPending;
  // Parent product is fixed on edit / when supplied; otherwise pick from a list.
  const lockedProduct = service?.product_id ?? productId;
  const { data: products } = useProducts({ page: 1, page_size: 100 });

  const defaults = (): FormValues => ({
    product_id: service?.product_id ?? productId ?? 0,
    name: service?.name ?? "",
    code: service?.code ?? "",
    type: service?.type ?? "",
    availability: service?.availability ?? "",
    description: service?.description ?? "",
    support_channels: listToString(service?.support_channels),
    tags: listToString(service?.tags),
    status: service?.status ?? ACTIVE,
  });

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

  // Re-seed the form when the dialog opens (so edit reflects latest data).
  useEffect(() => {
    if (open) reset(defaults());
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  async function onSubmit(values: FormValues) {
    const payload: CreateServiceInput = {
      product_id: values.product_id,
      name: values.name,
      code: values.code || undefined,
      type: values.type || undefined,
      availability: values.availability || undefined,
      description: values.description || undefined,
      support_channels: stringToList(values.support_channels),
      tags: stringToList(values.tags),
      status: values.status,
    };
    try {
      if (isEdit) {
        await update.mutateAsync(payload);
        toast.success("Service updated");
      } else {
        await create.mutateAsync(payload);
        toast.success("Service created");
      }
      setOpen(false);
    } catch (err) {
      toast.error(
        apiError(err, isEdit ? "Could not update service" : "Could not create service")
      );
    }
  }

  const productIdValue = watch("product_id");

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        {trigger ?? (
          <Button>
            <Plus /> New service
          </Button>
        )}
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{isEdit ? "Edit service" : "New service"}</DialogTitle>
          <DialogDescription>
            {isEdit
              ? "Update this service's catalog details."
              : "Register a service offered under a product."}
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div className="space-y-1.5">
            <Label>Product</Label>
            <Select
              value={productIdValue ? String(productIdValue) : undefined}
              onValueChange={(v) => setValue("product_id", Number(v))}
              disabled={lockedProduct != null}
            >
              <SelectTrigger>
                <SelectValue placeholder="Select a product" />
              </SelectTrigger>
              <SelectContent>
                {(products?.items ?? []).map((p) => (
                  <SelectItem key={p.id} value={String(p.id)}>
                    {p.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            {errors.product_id && (
              <p className="text-xs text-destructive">{errors.product_id.message}</p>
            )}
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="s-name">Name</Label>
            <Input id="s-name" placeholder="24/7 Monitoring" {...register("name")} />
            {errors.name && (
              <p className="text-xs text-destructive">{errors.name.message}</p>
            )}
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label htmlFor="s-code">Code</Label>
              <Input id="s-code" placeholder="MON-247" {...register("code")} />
              {errors.code && (
                <p className="text-xs text-destructive">{errors.code.message}</p>
              )}
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="s-type">Type</Label>
              <Input id="s-type" placeholder="monitoring" {...register("type")} />
            </div>
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="s-availability">Availability</Label>
            <Input
              id="s-availability"
              placeholder="99.9%"
              {...register("availability")}
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="s-channels">Support channels</Label>
            <Input
              id="s-channels"
              placeholder="email, phone, chat"
              {...register("support_channels")}
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="s-tags">Tags</Label>
            <Input
              id="s-tags"
              placeholder="comma, separated, tags"
              {...register("tags")}
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="s-desc">Description</Label>
            <Textarea
              id="s-desc"
              placeholder="Notes about this service…"
              {...register("description")}
            />
          </div>
          <div className="space-y-1.5">
            <Label>Status</Label>
            <Select
              value={watch("status")}
              onValueChange={(v) => setValue("status", v)}
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={ACTIVE}>Active</SelectItem>
                <SelectItem value={INACTIVE}>Inactive</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <DialogFooter>
            <DialogClose asChild>
              <Button type="button" variant="ghost">
                Cancel
              </Button>
            </DialogClose>
            <Button type="submit" disabled={pending}>
              {pending
                ? isEdit
                  ? "Saving…"
                  : "Creating…"
                : isEdit
                  ? "Save changes"
                  : "Create service"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
