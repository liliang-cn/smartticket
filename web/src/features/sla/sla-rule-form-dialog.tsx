import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Plus } from "lucide-react";
import { toast } from "sonner";
import { useTranslation } from "react-i18next";
import {
  useCreateSLARule,
  useUpdateSLARule,
  type CreateSLARuleInput,
  type SLARule,
} from "@/features/sla/api";
import { apiError } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
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

const PRIORITY_OPTIONS = ["low", "medium", "high", "critical"];
const SEVERITY_OPTIONS = ["trivial", "minor", "major", "critical"];

// Validation message keys are resolved via t() at render time.
const schema = z.object({
  template_id: z
    .string()
    .min(1, "template_id_required")
    .refine((v) => Number.isInteger(Number(v)) && Number(v) > 0, {
      message: "template_id_positive",
    }),
  priority: z.string().min(1, "priority_required"),
  severity: z.string().min(1, "severity_required"),
  response_time: z
    .string()
    .refine((v) => v !== "" && Number(v) >= 0, { message: "time_non_negative" }),
  resolution_time: z
    .string()
    .refine((v) => v !== "" && Number(v) >= 0, { message: "time_non_negative" }),
  business_only: z.boolean(),
  is_active: z.boolean(),
});
type FormValues = z.infer<typeof schema>;

interface SLARuleFormDialogProps {
  /** When provided, the dialog edits this rule instead of creating one. */
  rule?: SLARule;
  /** Optional custom trigger. Defaults to a "New rule" button. */
  trigger?: React.ReactNode;
}

const ACTIVE = "active";
const INACTIVE = "inactive";

export function SLARuleFormDialog({ rule, trigger }: SLARuleFormDialogProps) {
  const { t } = useTranslation("sla");
  const [open, setOpen] = useState(false);
  const isEdit = rule != null;
  const create = useCreateSLARule();
  const update = useUpdateSLARule(rule?.id ?? 0);
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
      template_id: rule?.template_id ? String(rule.template_id) : "",
      priority: rule?.priority ?? PRIORITY_OPTIONS[1],
      severity: rule?.severity ?? SEVERITY_OPTIONS[1],
      response_time: String(rule?.response_time ?? 60),
      resolution_time: String(rule?.resolution_time ?? 480),
      business_only: rule?.business_only ?? false,
      is_active: rule?.is_active ?? true,
    },
  });

  // Re-seed the form when the dialog opens (so edit reflects latest data).
  useEffect(() => {
    if (open) {
      reset({
        template_id: rule?.template_id ? String(rule.template_id) : "",
        priority: rule?.priority ?? PRIORITY_OPTIONS[1],
        severity: rule?.severity ?? SEVERITY_OPTIONS[1],
        response_time: String(rule?.response_time ?? 60),
        resolution_time: String(rule?.resolution_time ?? 480),
        business_only: rule?.business_only ?? false,
        is_active: rule?.is_active ?? true,
      });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  async function onSubmit(values: FormValues) {
    const payload: CreateSLARuleInput = {
      template_id: Number(values.template_id),
      priority: values.priority,
      severity: values.severity,
      response_time: Number(values.response_time),
      resolution_time: Number(values.resolution_time),
      business_only: values.business_only,
      is_active: values.is_active,
    };
    try {
      if (isEdit) {
        await update.mutateAsync(payload);
        toast.success(t("rule_dialog.toast.updated"));
      } else {
        await create.mutateAsync(payload);
        toast.success(t("rule_dialog.toast.created"));
      }
      setOpen(false);
    } catch (err) {
      toast.error(
        apiError(
          err,
          isEdit
            ? t("rule_dialog.toast.update_error")
            : t("rule_dialog.toast.create_error")
        )
      );
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        {trigger ?? (
          <Button>
            <Plus /> {t("rule_dialog.trigger_new")}
          </Button>
        )}
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{isEdit ? t("rule_dialog.title_edit") : t("rule_dialog.title_new")}</DialogTitle>
          <DialogDescription>
            {isEdit
              ? t("rule_dialog.description_edit")
              : t("rule_dialog.description_new")}
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div className="space-y-1.5">
            <Label htmlFor="r-template">{t("rule_dialog.fields.template_id_label")}</Label>
            <Input
              id="r-template"
              type="number"
              placeholder="1"
              {...register("template_id")}
            />
            {errors.template_id && (
              <p className="text-xs text-destructive">
                {t(`rule_dialog.validation.${errors.template_id.message}`)}
              </p>
            )}
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label>{t("rule_dialog.fields.priority_label")}</Label>
              <Select
                value={watch("priority")}
                onValueChange={(v) => setValue("priority", v)}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {PRIORITY_OPTIONS.map((p) => (
                    <SelectItem key={p} value={p} className="capitalize">
                      {p}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              {errors.priority && (
                <p className="text-xs text-destructive">
                  {t("rule_dialog.validation.priority_required")}
                </p>
              )}
            </div>
            <div className="space-y-1.5">
              <Label>{t("rule_dialog.fields.severity_label")}</Label>
              <Select
                value={watch("severity")}
                onValueChange={(v) => setValue("severity", v)}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {SEVERITY_OPTIONS.map((s) => (
                    <SelectItem key={s} value={s} className="capitalize">
                      {s}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              {errors.severity && (
                <p className="text-xs text-destructive">
                  {t("rule_dialog.validation.severity_required")}
                </p>
              )}
            </div>
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label htmlFor="r-response">{t("rule_dialog.fields.response_time_label")}</Label>
              <Input
                id="r-response"
                type="number"
                placeholder="60"
                {...register("response_time")}
              />
              {errors.response_time && (
                <p className="text-xs text-destructive">
                  {t("rule_dialog.validation.time_non_negative")}
                </p>
              )}
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="r-resolution">{t("rule_dialog.fields.resolution_time_label")}</Label>
              <Input
                id="r-resolution"
                type="number"
                placeholder="480"
                {...register("resolution_time")}
              />
              {errors.resolution_time && (
                <p className="text-xs text-destructive">
                  {t("rule_dialog.validation.time_non_negative")}
                </p>
              )}
            </div>
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label>{t("rule_dialog.fields.business_hours_label")}</Label>
              <Select
                value={watch("business_only") ? "yes" : "no"}
                onValueChange={(v) => setValue("business_only", v === "yes")}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="no">{t("rule_dialog.options.no")}</SelectItem>
                  <SelectItem value="yes">{t("rule_dialog.options.yes")}</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label>{t("rule_dialog.fields.status_label")}</Label>
              <Select
                value={watch("is_active") ? ACTIVE : INACTIVE}
                onValueChange={(v) => setValue("is_active", v === ACTIVE)}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value={ACTIVE}>{t("rule_dialog.options.active")}</SelectItem>
                  <SelectItem value={INACTIVE}>{t("rule_dialog.options.inactive")}</SelectItem>
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
                  ? t("rule_dialog.submitting_edit")
                  : t("rule_dialog.submitting_create")
                : isEdit
                  ? t("rule_dialog.submit_edit")
                  : t("rule_dialog.submit_create")}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
