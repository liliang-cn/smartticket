import { useEffect, useState } from "react";
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

const schema = z.object({
  name: z.string().min(1, "Name is required").max(100),
  description: z.string().max(500).optional(),
});
type FormValues = z.infer<typeof schema>;

interface RoleFormDialogProps {
  /** When provided, the dialog edits this role instead of creating one. */
  role?: RbacRoleFull;
  /** Optional custom trigger. Defaults to a "New role" button. */
  trigger?: React.ReactNode;
}

export function RoleFormDialog({ role, trigger }: RoleFormDialogProps) {
  const [open, setOpen] = useState(false);
  const isEdit = role != null;
  // System roles may not be renamed; only the description is editable.
  const nameLocked = isEdit && role.is_system;
  const create = useCreateRole();
  const update = useUpdateRole(role?.id ?? 0);
  const pending = isEdit ? update.isPending : create.isPending;

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
        toast.success("Role updated");
      } else {
        await create.mutateAsync(payload);
        toast.success("Role created");
      }
      setOpen(false);
    } catch (err) {
      toast.error(
        apiError(err, isEdit ? "Could not update role" : "Could not create role")
      );
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        {trigger ?? (
          <Button>
            <Plus /> New role
          </Button>
        )}
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{isEdit ? "Edit role" : "New role"}</DialogTitle>
          <DialogDescription>
            {isEdit
              ? "Update this role's details."
              : "Define a new role that can be assigned to users."}
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div className="space-y-1.5">
            <Label htmlFor="r-name">Name</Label>
            <Input
              id="r-name"
              placeholder="support-lead"
              disabled={nameLocked}
              {...register("name")}
            />
            {nameLocked && (
              <p className="text-xs text-muted-foreground">
                System role names cannot be changed.
              </p>
            )}
            {errors.name && (
              <p className="text-xs text-destructive">{errors.name.message}</p>
            )}
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="r-desc">Description</Label>
            <Textarea
              id="r-desc"
              placeholder="What this role is allowed to do…"
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
                  : "Create role"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
