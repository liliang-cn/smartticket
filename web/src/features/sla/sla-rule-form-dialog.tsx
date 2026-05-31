import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Plus } from "lucide-react";
import { toast } from "sonner";
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

const schema = z.object({
  template_id: z
    .string()
    .min(1, "Template ID is required")
    .refine((v) => Number.isInteger(Number(v)) && Number(v) > 0, {
      message: "Template ID must be a positive number",
    }),
  priority: z.string().min(1, "Priority is required"),
  severity: z.string().min(1, "Severity is required"),
  response_time: z
    .string()
    .refine((v) => v !== "" && Number(v) >= 0, { message: "Must be ≥ 0" }),
  resolution_time: z
    .string()
    .refine((v) => v !== "" && Number(v) >= 0, { message: "Must be ≥ 0" }),
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
        toast.success("SLA rule updated");
      } else {
        await create.mutateAsync(payload);
        toast.success("SLA rule created");
      }
      setOpen(false);
    } catch (err) {
      toast.error(
        apiError(
          err,
          isEdit ? "Could not update rule" : "Could not create rule"
        )
      );
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        {trigger ?? (
          <Button>
            <Plus /> New rule
          </Button>
        )}
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{isEdit ? "Edit SLA rule" : "New SLA rule"}</DialogTitle>
          <DialogDescription>
            {isEdit
              ? "Update this SLA rule's targets."
              : "Map priority/severity to response & resolution targets."}
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div className="space-y-1.5">
            <Label htmlFor="r-template">Template ID</Label>
            <Input
              id="r-template"
              type="number"
              placeholder="1"
              {...register("template_id")}
            />
            {errors.template_id && (
              <p className="text-xs text-destructive">
                {errors.template_id.message}
              </p>
            )}
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label>Priority</Label>
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
                  {errors.priority.message}
                </p>
              )}
            </div>
            <div className="space-y-1.5">
              <Label>Severity</Label>
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
                  {errors.severity.message}
                </p>
              )}
            </div>
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label htmlFor="r-response">Response time (min)</Label>
              <Input
                id="r-response"
                type="number"
                placeholder="60"
                {...register("response_time")}
              />
              {errors.response_time && (
                <p className="text-xs text-destructive">
                  {errors.response_time.message}
                </p>
              )}
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="r-resolution">Resolution time (min)</Label>
              <Input
                id="r-resolution"
                type="number"
                placeholder="480"
                {...register("resolution_time")}
              />
              {errors.resolution_time && (
                <p className="text-xs text-destructive">
                  {errors.resolution_time.message}
                </p>
              )}
            </div>
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label>Business hours only</Label>
              <Select
                value={watch("business_only") ? "yes" : "no"}
                onValueChange={(v) => setValue("business_only", v === "yes")}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="no">No</SelectItem>
                  <SelectItem value="yes">Yes</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label>Status</Label>
              <Select
                value={watch("is_active") ? ACTIVE : INACTIVE}
                onValueChange={(v) => setValue("is_active", v === ACTIVE)}
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
                  : "Create rule"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
