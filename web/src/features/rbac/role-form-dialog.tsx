import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Plus } from "lucide-react";
import { toast } from "sonner";
import {
  useCreateRole,
  useUpdateRole,
  type RbacRoleFull,
  type RoleInput,
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
    name: z.string().min(1, t("role_form.validation.name_required")).max(100),
    description: z.string().max(500).optional(),
  });
}
type FormValues = {
  name: string;
  description?: string;
};

interface RoleFormDialogProps {
  /** When provided, the dialog edits this role instead of creating one. */
  role?: RbacRoleFull;
  /** Optional custom trigger. Defaults to a "New role" button. */
  trigger?: React.ReactNode;
}

export function RoleFormDialog({ role, trigger }: RoleFormDialogProps) {
  const { t } = useTranslation("rbac");
  const [open, setOpen] = useState(false);
  const isEdit = role != null;
  // System roles may not be renamed; only the description is editable.
  const nameLocked = isEdit && role.is_system;
  const create = useCreateRole();
  const update = useUpdateRole(role?.id ?? 0);
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
      name: role?.name ?? "",
      description: role?.description ?? "",
    },
  });

  useEffect(() => {
    if (open) {
      reset({
        name: role?.name ?? "",
        description: role?.description ?? "",
      });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  async function onSubmit(values: FormValues) {
    const payload: RoleInput = {
      name: values.name,
      description: values.description || undefined,
    };
    // Never send a renamed `name` for a system role.
    if (nameLocked) payload.name = role.name;
    try {
      if (isEdit) {
        await update.mutateAsync(payload);
        toast.success(t("role_form.toast.updated"));
      } else {
        await create.mutateAsync(payload);
        toast.success(t("role_form.toast.created"));
      }
      setOpen(false);
    } catch (err) {
      toast.error(
        apiError(
          err,
          isEdit ? t("role_form.toast.update_error") : t("role_form.toast.create_error")
        )
      );
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        {trigger ?? (
          <Button>
            <Plus /> {t("role_form.trigger_new")}
          </Button>
        )}
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{isEdit ? t("role_form.title_edit") : t("role_form.title_new")}</DialogTitle>
          <DialogDescription>
            {isEdit
              ? t("role_form.description_edit")
              : t("role_form.description_new")}
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div className="space-y-1.5">
            <Label htmlFor="r-name">{t("role_form.field_name")}</Label>
            <Input
              id="r-name"
              placeholder={t("role_form.placeholder_name")}
              disabled={nameLocked}
              {...register("name")}
            />
            {nameLocked && (
              <p className="text-xs text-muted-foreground">
                {t("role_form.name_locked_hint")}
              </p>
            )}
            {errors.name && (
              <p className="text-xs text-destructive">{errors.name.message}</p>
            )}
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="r-desc">{t("role_form.field_description")}</Label>
            <Textarea
              id="r-desc"
              placeholder={t("role_form.placeholder_description")}
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
                  ? t("role_form.button_saving")
                  : t("role_form.button_creating")
                : isEdit
                  ? t("role_form.button_save")
                  : t("role_form.button_create")}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
