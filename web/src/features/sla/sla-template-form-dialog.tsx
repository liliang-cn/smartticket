import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Plus } from "lucide-react";
import { toast } from "sonner";
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

const schema = z.object({
  name: z.string().min(1, "Name is required").max(255),
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
        toast.success("SLA template updated");
      } else {
        await create.mutateAsync(payload);
        toast.success("SLA template created");
      }
      setOpen(false);
    } catch (err) {
      toast.error(
        apiError(
          err,
          isEdit ? "Could not update template" : "Could not create template"
        )
      );
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        {trigger ?? (
          <Button>
            <Plus /> New template
          </Button>
        )}
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>
            {isEdit ? "Edit SLA template" : "New SLA template"}
          </DialogTitle>
          <DialogDescription>
            {isEdit
              ? "Update this SLA template's basic details."
              : "Define a reusable service-level agreement template."}
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div className="space-y-1.5">
            <Label htmlFor="t-name">Name</Label>
            <Input
              id="t-name"
              placeholder="Standard support"
              {...register("name")}
            />
            {errors.name && (
              <p className="text-xs text-destructive">{errors.name.message}</p>
            )}
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="t-desc">Description</Label>
            <Textarea
              id="t-desc"
              placeholder="Notes about this SLA template…"
              {...register("description")}
            />
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label>Default</Label>
              <Select
                value={watch("is_default") ? "yes" : "no"}
                onValueChange={(v) => setValue("is_default", v === "yes")}
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
                  : "Create template"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
