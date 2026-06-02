import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Plus } from "lucide-react";
import { toast } from "sonner";
import { useTranslation } from "react-i18next";
import {
  useCreateSLATemplate,
  useUpdateSLATemplate,
  type CreateSLATemplateInput,
  type SLATemplate,
} from "@/features/sla/api";
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

// Validation messages are resolved at submit time via t(); keep static keys here.
const schema = z.object({
  name: z.string().min(1, "name_required").max(255),
  description: z.string().optional(),
  is_default: z.boolean(),
  is_active: z.boolean(),
});
type FormValues = z.infer<typeof schema>;

interface SLATemplateFormDialogProps {
  /** When provided, the dialog edits this template instead of creating one. */
  template?: SLATemplate;
  /** Optional custom trigger. Defaults to a "New template" button. */
  trigger?: React.ReactNode;
}

const ACTIVE = "active";
const INACTIVE = "inactive";

export function SLATemplateFormDialog({
  template,
  trigger,
}: SLATemplateFormDialogProps) {
  const { t } = useTranslation("sla");
  const [open, setOpen] = useState(false);
  const isEdit = template != null;
  const create = useCreateSLATemplate();
  const update = useUpdateSLATemplate(template?.id ?? 0);
  const pending = isEdit ? update.isPending : create.isPending;

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
      name: template?.name ?? "",
      description: template?.description ?? "",
      is_default: template?.is_default ?? false,
      is_active: template?.is_active ?? true,
    },
  });

  // Re-seed the form when the dialog opens (so edit reflects latest data).
  useEffect(() => {
    if (open) {
      reset({
        name: template?.name ?? "",
        description: template?.description ?? "",
        is_default: template?.is_default ?? false,
        is_active: template?.is_active ?? true,
      });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  async function onSubmit(values: FormValues) {
    const payload: CreateSLATemplateInput = {
      name: values.name,
      description: values.description || undefined,
      is_default: values.is_default,
      is_active: values.is_active,
    };
    try {
      if (isEdit) {
        await update.mutateAsync(payload);
        toast.success(t("template_dialog.toast.updated"));
      } else {
        await create.mutateAsync(payload);
        toast.success(t("template_dialog.toast.created"));
      }
      setOpen(false);
    } catch (err) {
      toast.error(
        apiError(
          err,
          isEdit
            ? t("template_dialog.toast.update_error")
            : t("template_dialog.toast.create_error")
        )
      );
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        {trigger ?? (
          <Button>
            <Plus /> {t("template_dialog.trigger_new")}
          </Button>
        )}
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>
            {isEdit ? t("template_dialog.title_edit") : t("template_dialog.title_new")}
          </DialogTitle>
          <DialogDescription>
            {isEdit
              ? t("template_dialog.description_edit")
              : t("template_dialog.description_new")}
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div className="space-y-1.5">
            <Label htmlFor="t-name">{t("template_dialog.fields.name_label")}</Label>
            <Input
              id="t-name"
              placeholder={t("template_dialog.fields.name_placeholder")}
              {...register("name")}
            />
            {errors.name && (
              <p className="text-xs text-destructive">
                {t("template_dialog.validation.name_required")}
              </p>
            )}
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="t-desc">{t("template_dialog.fields.description_label")}</Label>
            <Textarea
              id="t-desc"
              placeholder={t("template_dialog.fields.description_placeholder")}
              {...register("description")}
            />
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label>{t("template_dialog.fields.default_label")}</Label>
              <Select
                value={watch("is_default") ? "yes" : "no"}
                onValueChange={(v) => setValue("is_default", v === "yes")}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="no">{t("template_dialog.options.no")}</SelectItem>
                  <SelectItem value="yes">{t("template_dialog.options.yes")}</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label>{t("template_dialog.fields.status_label")}</Label>
              <Select
                value={watch("is_active") ? ACTIVE : INACTIVE}
                onValueChange={(v) => setValue("is_active", v === ACTIVE)}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value={ACTIVE}>{t("template_dialog.options.active")}</SelectItem>
                  <SelectItem value={INACTIVE}>{t("template_dialog.options.inactive")}</SelectItem>
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
                  ? t("template_dialog.submitting_edit")
                  : t("template_dialog.submitting_create")
                : isEdit
                  ? t("template_dialog.submit_edit")
                  : t("template_dialog.submit_create")}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
