import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Plus } from "lucide-react";
import { toast } from "sonner";
import {
  useCreateProduct,
  useUpdateProduct,
  type CreateProductInput,
  type Product,
} from "@/features/products/api";
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
  name: z.string().min(1, "Name is required").max(255),
  code: z.string().max(100).optional(),
  category: z.string().max(100).optional(),
  version: z.string().max(100).optional(),
  support_level: z.string().max(100).optional(),
  documentation: z.string().optional(),
  description: z.string().optional(),
  tags: z.string().optional(),
  status: z.string(),
  is_managed: z.boolean(),
});
type FormValues = z.infer<typeof schema>;

interface ProductFormDialogProps {
  /** When provided, the dialog edits this product instead of creating one. */
  product?: Product;
  /** Optional custom trigger. Defaults to a "New product" button. */
  trigger?: React.ReactNode;
}

const ACTIVE = "active";
const INACTIVE = "inactive";

function tagsToString(tags?: string[]): string {
  return (tags ?? []).join(", ");
}

function stringToTags(value?: string): string[] {
  return (value ?? "")
    .split(",")
    .map((t) => t.trim())
    .filter(Boolean);
}

export function ProductFormDialog({ product, trigger }: ProductFormDialogProps) {
  const [open, setOpen] = useState(false);
  const isEdit = product != null;
  const create = useCreateProduct();
  const update = useUpdateProduct(product?.id ?? 0);
  const pending = isEdit ? update.isPending : create.isPending;

  const defaults = (): FormValues => ({
    name: product?.name ?? "",
    code: product?.code ?? "",
    category: product?.category ?? "",
    version: product?.version ?? "",
    support_level: product?.support_level ?? "",
    documentation: product?.documentation ?? "",
    description: product?.description ?? "",
    tags: tagsToString(product?.tags),
    status: product?.status ?? ACTIVE,
    is_managed: product?.is_managed ?? false,
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
    const payload: CreateProductInput = {
      name: values.name,
      code: values.code || undefined,
      category: values.category || undefined,
      version: values.version || undefined,
      support_level: values.support_level || undefined,
      documentation: values.documentation || undefined,
      description: values.description || undefined,
      tags: stringToTags(values.tags),
      status: values.status,
      is_managed: values.is_managed,
    };
    try {
      if (isEdit) {
        await update.mutateAsync(payload);
        toast.success("Product updated");
      } else {
        await create.mutateAsync(payload);
        toast.success("Product created");
      }
      setOpen(false);
    } catch (err) {
      toast.error(
        apiError(err, isEdit ? "Could not update product" : "Could not create product")
      );
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        {trigger ?? (
          <Button>
            <Plus /> New product
          </Button>
        )}
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{isEdit ? "Edit product" : "New product"}</DialogTitle>
          <DialogDescription>
            {isEdit
              ? "Update this product's catalog details."
              : "Register a product in your service catalog."}
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div className="space-y-1.5">
            <Label htmlFor="p-name">Name</Label>
            <Input id="p-name" placeholder="Gateway Pro" {...register("name")} />
            {errors.name && (
              <p className="text-xs text-destructive">{errors.name.message}</p>
            )}
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label htmlFor="p-code">Code</Label>
              <Input id="p-code" placeholder="GW-PRO" {...register("code")} />
              {errors.code && (
                <p className="text-xs text-destructive">{errors.code.message}</p>
              )}
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="p-category">Category</Label>
              <Input
                id="p-category"
                placeholder="Networking"
                {...register("category")}
              />
            </div>
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label htmlFor="p-version">Version</Label>
              <Input id="p-version" placeholder="2.4.0" {...register("version")} />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="p-support">Support level</Label>
              <Input
                id="p-support"
                placeholder="standard"
                {...register("support_level")}
              />
            </div>
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="p-tags">Tags</Label>
            <Input
              id="p-tags"
              placeholder="comma, separated, tags"
              {...register("tags")}
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="p-desc">Description</Label>
            <Textarea
              id="p-desc"
              placeholder="Notes about this product…"
              {...register("description")}
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="p-doc">Documentation</Label>
            <Textarea
              id="p-doc"
              placeholder="Links or documentation references…"
              {...register("documentation")}
            />
          </div>
          <div className="grid grid-cols-2 gap-4">
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
            <div className="space-y-1.5">
              <Label>Managed</Label>
              <Select
                value={watch("is_managed") ? "yes" : "no"}
                onValueChange={(v) => setValue("is_managed", v === "yes")}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="yes">Managed</SelectItem>
                  <SelectItem value="no">Unmanaged</SelectItem>
                </SelectContent>
              </Select>
            </div>
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
                  : "Create product"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
