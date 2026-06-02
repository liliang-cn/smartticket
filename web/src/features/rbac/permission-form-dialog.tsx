import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Plus } from "lucide-react";
import { toast } from "sonner";
import {
  useCreatePermission,
  useUpdatePermission,
  type PermissionInput,
  type RbacPermissionFull,
} from "@/features/rbac/api";
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

function buildSchema(t: (key: string) => string) {
  return z.object({
    code: z
      .string()
      .min(1, t("permission_form.validation.code_required"))
      .max(100)
      .regex(/^[a-z0-9_]+:[a-z0-9_]+$/i, t("permission_form.validation.code_format")),
    name: z.string().min(1, t("permission_form.validation.name_required")).max(255),
    description: z.string().max(500).optional(),
    category: z.string().max(100).optional(),
  });
}
type FormValues = {
  code: string;
  name: string;
  description?: string;
  category?: string;
};

interface PermissionFormDialogProps {
  /** When provided, the dialog edits this permission instead of creating one. */
  permission?: RbacPermissionFull;
  /** Optional custom trigger. Defaults to a "New permission" button. */
  trigger?: React.ReactNode;
}

export function PermissionFormDialog({
  permission,
  trigger,
}: PermissionFormDialogProps) {
  const { t } = useTranslation("rbac");
  const [open, setOpen] = useState(false);
  const isEdit = permission != null;
  // System permission codes are immutable.
  const codeLocked = isEdit && permission.is_system;
  const create = useCreatePermission();
  const update = useUpdatePermission(permission?.id ?? 0);
  const pending = isEdit ? update.isPending : create.isPending;

  const schema = buildSchema(t);

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      code: permission?.code ?? "",
      name: permission?.name ?? "",
      description: permission?.description ?? "",
      category: permission?.category ?? "",
    },
  });

  useEffect(() => {
    if (open) {
      reset({
        code: permission?.code ?? "",
        name: permission?.name ?? "",
        description: permission?.description ?? "",
        category: permission?.category ?? "",
      });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  async function onSubmit(values: FormValues) {
    const payload: PermissionInput = {
      code: codeLocked ? permission.code : values.code,
      name: values.name,
      description: values.description || undefined,
      category: values.category || undefined,
    };
    try {
      if (isEdit) {
        await update.mutateAsync(payload);
        toast.success(t("permission_form.toast.updated"));
      } else {
        await create.mutateAsync(payload);
        toast.success(t("permission_form.toast.created"));
      }
      setOpen(false);
    } catch (err) {
      toast.error(
        apiError(
          err,
          isEdit
            ? t("permission_form.toast.update_error")
            : t("permission_form.toast.create_error")
        )
      );
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        {trigger ?? (
          <Button>
            <Plus /> {t("permission_form.trigger_new")}
          </Button>
        )}
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>
            {isEdit ? t("permission_form.title_edit") : t("permission_form.title_new")}
          </DialogTitle>
          <DialogDescription>
            {isEdit
              ? t("permission_form.description_edit")
              : t("permission_form.description_new")}
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label htmlFor="p-code">{t("permission_form.field_code")}</Label>
              <Input
                id="p-code"
                placeholder={t("permission_form.placeholder_code")}
                disabled={codeLocked}
                {...register("code")}
              />
              {codeLocked && (
                <p className="text-xs text-muted-foreground">
                  {t("permission_form.code_locked_hint")}
                </p>
              )}
              {errors.code && (
                <p className="text-xs text-destructive">
                  {errors.code.message}
                </p>
              )}
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="p-category">{t("permission_form.field_category")}</Label>
              <Input
                id="p-category"
                placeholder={t("permission_form.placeholder_category")}
                {...register("category")}
              />
              {errors.category && (
                <p className="text-xs text-destructive">
                  {errors.category.message}
                </p>
              )}
            </div>
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="p-name">{t("permission_form.field_name")}</Label>
            <Input
              id="p-name"
              placeholder={t("permission_form.placeholder_name")}
              {...register("name")}
            />
            {errors.name && (
              <p className="text-xs text-destructive">{errors.name.message}</p>
            )}
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="p-desc">{t("permission_form.field_description")}</Label>
            <Textarea
              id="p-desc"
              placeholder={t("permission_form.placeholder_description")}
              {...register("description")}
            />
            {errors.description && (
              <p className="text-xs text-destructive">
                {errors.description.message}
              </p>
            )}
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
                  ? t("permission_form.button_saving")
                  : t("permission_form.button_creating")
                : isEdit
                  ? t("permission_form.button_save")
                  : t("permission_form.button_create")}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
