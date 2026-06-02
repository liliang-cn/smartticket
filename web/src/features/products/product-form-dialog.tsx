import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Plus } from "lucide-react";
import { toast } from "sonner";
import { useTranslation } from "react-i18next";
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
  name: z.string().min(1, "form.validation_name_required").max(255),
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
  const { t } = useTranslation("products");
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
        toast.success(t("form.toast.updated"));
      } else {
        await create.mutateAsync(payload);
        toast.success(t("form.toast.created"));
      }
      setOpen(false);
    } catch (err) {
      toast.error(
        apiError(
          err,
          isEdit ? t("form.toast.error_update") : t("form.toast.error_create"),
        ),
      );
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        {trigger ?? (
          <Button>
            <Plus /> {t("form.new_trigger")}
          </Button>
        )}
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>
            {isEdit ? t("form.title_edit") : t("form.title_create")}
          </DialogTitle>
          <DialogDescription>
            {isEdit ? t("form.desc_edit") : t("form.desc_create")}
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div className="space-y-1.5">
            <Label htmlFor="p-name">{t("form.label_name")}</Label>
            <Input
              id="p-name"
              placeholder={t("form.placeholder_name")}
              {...register("name")}
            />
            {errors.name && (
              <p className="text-xs text-destructive">
                {t(errors.name.message ?? "form.validation_name_required")}
              </p>
            )}
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label htmlFor="p-code">{t("form.label_code")}</Label>
              <Input
                id="p-code"
                placeholder={t("form.placeholder_code")}
                {...register("code")}
              />
              {errors.code && (
                <p className="text-xs text-destructive">{errors.code.message}</p>
              )}
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="p-category">{t("form.label_category")}</Label>
              <Input
                id="p-category"
                placeholder={t("form.placeholder_category")}
                {...register("category")}
              />
            </div>
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label htmlFor="p-version">{t("form.label_version")}</Label>
              <Input
                id="p-version"
                placeholder={t("form.placeholder_version")}
                {...register("version")}
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="p-support">{t("form.label_support_level")}</Label>
              <Input
                id="p-support"
                placeholder={t("form.placeholder_support_level")}
                {...register("support_level")}
              />
            </div>
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="p-tags">{t("form.label_tags")}</Label>
            <Input
              id="p-tags"
              placeholder={t("form.placeholder_tags")}
              {...register("tags")}
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="p-desc">{t("form.label_description")}</Label>
            <Textarea
              id="p-desc"
              placeholder={t("form.placeholder_description")}
              {...register("description")}
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="p-doc">{t("form.label_documentation")}</Label>
            <Textarea
              id="p-doc"
              placeholder={t("form.placeholder_documentation")}
              {...register("documentation")}
            />
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label>{t("form.label_status")}</Label>
              <Select
                value={watch("status")}
                onValueChange={(v) => setValue("status", v)}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value={ACTIVE}>{t("form.status_active")}</SelectItem>
                  <SelectItem value={INACTIVE}>{t("form.status_inactive")}</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label>{t("form.label_managed")}</Label>
              <Select
                value={watch("is_managed") ? "yes" : "no"}
                onValueChange={(v) => setValue("is_managed", v === "yes")}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="yes">{t("form.managed_yes")}</SelectItem>
                  <SelectItem value="no">{t("form.managed_no")}</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
          <DialogFooter>
            <DialogClose asChild>
              <Button type="button" variant="ghost">
                {t("actions.cancel", { ns: "common" })}
              </Button>
            </DialogClose>
            <Button type="submit" disabled={pending}>
              {pending
                ? isEdit
                  ? t("form.submit_saving")
                  : t("form.submit_creating")
                : isEdit
                  ? t("form.submit_save")
                  : t("form.submit_create")}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
