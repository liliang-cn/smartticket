import { useEffect, useState } from "react";
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

const schema = z.object({
  code: z
    .string()
    .min(1, "Code is required")
    .max(100)
    .regex(/^[a-z0-9_]+:[a-z0-9_]+$/i, "Use the form resource:action, e.g. ticket:write"),
  name: z.string().min(1, "Name is required").max(255),
  description: z.string().max(500).optional(),
  category: z.string().max(100).optional(),
});
type FormValues = z.infer<typeof schema>;

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
  const [open, setOpen] = useState(false);
  const isEdit = permission != null;
  // System permission codes are immutable.
  const codeLocked = isEdit && permission.is_system;
  const create = useCreatePermission();
  const update = useUpdatePermission(permission?.id ?? 0);
  const pending = isEdit ? update.isPending : create.isPending;

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
        toast.success("Permission updated");
      } else {
        await create.mutateAsync(payload);
        toast.success("Permission created");
      }
      setOpen(false);
    } catch (err) {
      toast.error(
        apiError(
          err,
          isEdit ? "Could not update permission" : "Could not create permission"
        )
      );
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        {trigger ?? (
          <Button>
            <Plus /> New permission
          </Button>
        )}
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>
            {isEdit ? "Edit permission" : "New permission"}
          </DialogTitle>
          <DialogDescription>
            {isEdit
              ? "Update this permission's details."
              : "Define a new permission that can be granted to roles."}
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label htmlFor="p-code">Code</Label>
              <Input
                id="p-code"
                placeholder="ticket:write"
                disabled={codeLocked}
                {...register("code")}
              />
              {codeLocked && (
                <p className="text-xs text-muted-foreground">
                  System permission codes cannot be changed.
                </p>
              )}
              {errors.code && (
                <p className="text-xs text-destructive">
                  {errors.code.message}
                </p>
              )}
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="p-category">Category</Label>
              <Input
                id="p-category"
                placeholder="ticket"
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
            <Label htmlFor="p-name">Name</Label>
            <Input
              id="p-name"
              placeholder="Write tickets"
              {...register("name")}
            />
            {errors.name && (
              <p className="text-xs text-destructive">{errors.name.message}</p>
            )}
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="p-desc">Description</Label>
            <Textarea
              id="p-desc"
              placeholder="What this permission grants…"
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
                  : "Create permission"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
