import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Plus } from "lucide-react";
import { toast } from "sonner";
import { useTranslation } from "react-i18next";
import {
  useCreateCustomer,
  useUpdateCustomer,
  type CreateCustomerInput,
} from "@/features/customers/api";
import { apiError } from "@/lib/api";
import type { Customer } from "@/lib/types";
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

interface CustomerFormDialogProps {
  /** When provided, the dialog edits this customer instead of creating one. */
  customer?: Customer;
  /** Optional custom trigger. Defaults to a "New customer" button. */
  trigger?: React.ReactNode;
}

const ACTIVE = "active";
const INACTIVE = "inactive";

export function CustomerFormDialog({ customer, trigger }: CustomerFormDialogProps) {
  const { t } = useTranslation("customers");
  const [open, setOpen] = useState(false);
  const isEdit = customer != null;
  const create = useCreateCustomer();
  const update = useUpdateCustomer(customer?.id ?? 0);
  const pending = isEdit ? update.isPending : create.isPending;

  const schema = z.object({
    name: z.string().min(1, t("form.validation.name_required")).max(255),
    code: z.string().max(100).optional(),
    domain: z.string().max(255).optional(),
    description: z.string().optional(),
    is_active: z.boolean(),
  });
  type FormValues = z.infer<typeof schema>;

  const {
    register,
    handleSubmit,
    setValue,
    watch,
    reset,
    formState: { errors },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      name: customer?.name ?? "",
      code: customer?.code ?? "",
      domain: customer?.domain ?? "",
      description: customer?.description ?? "",
      is_active: customer?.is_active ?? true,
    },
  });

  // Re-seed the form when the dialog opens (so edit reflects latest data).
  useEffect(() => {
    if (open) {
      reset({
        name: customer?.name ?? "",
        code: customer?.code ?? "",
        domain: customer?.domain ?? "",
        description: customer?.description ?? "",
        is_active: customer?.is_active ?? true,
      });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  async function onSubmit(values: FormValues) {
    const payload: CreateCustomerInput = {
      name: values.name,
      code: values.code || undefined,
      domain: values.domain || undefined,
      description: values.description || undefined,
      is_active: values.is_active,
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
        apiError(err, isEdit ? t("form.toast.update_error") : t("form.toast.create_error")),
      );
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        {trigger ?? (
          <Button>
            <Plus /> {t("form.trigger_new")}
          </Button>
        )}
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>
            {isEdit ? t("form.title_edit") : t("form.title_new")}
          </DialogTitle>
          <DialogDescription>
            {isEdit
              ? t("form.description_edit")
              : t("form.description_new")}
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div className="space-y-1.5">
            <Label htmlFor="c-name">{t("form.field_name")}</Label>
            <Input id="c-name" placeholder="Acme Inc." {...register("name")} />
            {errors.name && (
              <p className="text-xs text-destructive">{errors.name.message}</p>
            )}
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label htmlFor="c-code">{t("form.field_code")}</Label>
              <Input id="c-code" placeholder="ACME" {...register("code")} />
              {errors.code && (
                <p className="text-xs text-destructive">{errors.code.message}</p>
              )}
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="c-domain">{t("form.field_domain")}</Label>
              <Input id="c-domain" placeholder="acme.com" {...register("domain")} />
              {errors.domain && (
                <p className="text-xs text-destructive">{errors.domain.message}</p>
              )}
            </div>
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="c-desc">{t("form.field_description")}</Label>
            <Textarea
              id="c-desc"
              placeholder={t("form.description_placeholder")}
              {...register("description")}
            />
          </div>
          <div className="space-y-1.5">
            <Label>{t("form.field_status")}</Label>
            <Select
              value={watch("is_active") ? ACTIVE : INACTIVE}
              onValueChange={(v) => setValue("is_active", v === ACTIVE)}
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={ACTIVE}>{t("list.status_active")}</SelectItem>
                <SelectItem value={INACTIVE}>{t("list.status_inactive")}</SelectItem>
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
                  : t("form.create")}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
