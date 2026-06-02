import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Plus } from "lucide-react";
import { useTranslation } from "react-i18next";
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
  const { t } = useTranslation("services");
  const [open, setOpen] = useState(false);
  const isEdit = service != null;
  const create = useCreateService();
  const update = useUpdateService(service?.id ?? 0);
  const pending = isEdit ? update.isPending : create.isPending;
  // Parent product is fixed on edit / when supplied; otherwise pick from a list.
  const lockedProduct = service?.product_id ?? productId;
  const { data: products } = useProducts({ page: 1, page_size: 100 });

  const schema = z.object({
    product_id: z.number().int().positive(t("form.validation_product_required")),
    name: z.string().min(1, t("form.validation_name_required")).max(255),
    code: z.string().max(100).optional(),
    type: z.string().max(100).optional(),
    availability: z.string().max(100).optional(),
    description: z.string().optional(),
    support_channels: z.string().optional(),
    tags: z.string().optional(),
    status: z.string(),
  });
  type FormValues = z.infer<typeof schema>;

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
        toast.success(t("form.toast_updated"));
      } else {
        await create.mutateAsync(payload);
        toast.success(t("form.toast_created"));
      }
      setOpen(false);
    } catch (err) {
      toast.error(
        apiError(err, isEdit ? t("form.error_update") : t("form.error_create"))
      );
    }
  }

  const productIdValue = watch("product_id");

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        {trigger ?? (
          <Button>
            <Plus /> {t("form.new_service")}
          </Button>
        )}
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{isEdit ? t("form.title_edit") : t("form.title_create")}</DialogTitle>
          <DialogDescription>
            {isEdit
              ? t("form.description_edit")
              : t("form.description_create")}
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div className="space-y-1.5">
            <Label>{t("form.label_product")}</Label>
            <Select
              value={productIdValue ? String(productIdValue) : undefined}
              onValueChange={(v) => setValue("product_id", Number(v))}
              disabled={lockedProduct != null}
            >
              <SelectTrigger>
                <SelectValue placeholder={t("form.placeholder_product")} />
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
            <Label htmlFor="s-name">{t("form.label_name")}</Label>
            <Input id="s-name" placeholder={t("form.placeholder_name")} {...register("name")} />
            {errors.name && (
              <p className="text-xs text-destructive">{errors.name.message}</p>
            )}
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label htmlFor="s-code">{t("form.label_code")}</Label>
              <Input id="s-code" placeholder={t("form.placeholder_code")} {...register("code")} />
              {errors.code && (
                <p className="text-xs text-destructive">{errors.code.message}</p>
              )}
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="s-type">{t("form.label_type")}</Label>
              <Input id="s-type" placeholder={t("form.placeholder_type")} {...register("type")} />
            </div>
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="s-availability">{t("form.label_availability")}</Label>
            <Input
              id="s-availability"
              placeholder={t("form.placeholder_availability")}
              {...register("availability")}
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="s-channels">{t("form.label_support_channels")}</Label>
            <Input
              id="s-channels"
              placeholder={t("form.placeholder_channels")}
              {...register("support_channels")}
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="s-tags">{t("form.label_tags")}</Label>
            <Input
              id="s-tags"
              placeholder={t("form.placeholder_tags")}
              {...register("tags")}
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="s-desc">{t("form.label_description")}</Label>
            <Textarea
              id="s-desc"
              placeholder={t("form.placeholder_description")}
              {...register("description")}
            />
          </div>
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
          <DialogFooter>
            <DialogClose asChild>
              <Button type="button" variant="ghost">
                {t("actions.cancel", { ns: "common" })}
              </Button>
            </DialogClose>
            <Button type="submit" disabled={pending}>
              {pending
                ? isEdit
                  ? t("form.saving")
                  : t("form.creating")
                : isEdit
                  ? t("form.save_changes")
                  : t("form.create_service")}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
