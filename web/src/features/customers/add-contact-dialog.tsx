import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { UserPlus } from "lucide-react";
import { toast } from "sonner";
import { useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { useCreateUser } from "@/features/users/api";
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

interface AddContactDialogProps {
  customerId: number;
  customerName?: string;
}

/**
 * Adds a contact (a customer-role user) bound to a specific customer org. The
 * role and customer_id are fixed here — the operator only fills in identity.
 */
export function AddContactDialog({ customerId, customerName }: AddContactDialogProps) {
  const { t } = useTranslation("customers");
  const [open, setOpen] = useState(false);
  const create = useCreateUser();
  const qc = useQueryClient();

  const schema = z.object({
    first_name: z.string().min(1, t("add_contact.validation.first_name_required")).max(100),
    last_name: z.string().min(1, t("add_contact.validation.last_name_required")).max(100),
    email: z
      .string()
      .min(1, t("add_contact.validation.email_required"))
      .email(t("add_contact.validation.email_invalid")),
    username: z
      .string()
      .min(3, t("add_contact.validation.username_min"))
      .max(50)
      .regex(/^[a-zA-Z0-9_-]+$/, t("add_contact.validation.username_pattern")),
    password: z
      .string()
      .min(8, t("add_contact.validation.password_min"))
      .regex(/[A-Z]/, t("add_contact.validation.password_uppercase"))
      .regex(/[a-z]/, t("add_contact.validation.password_lowercase"))
      .regex(/\d/, t("add_contact.validation.password_digit"))
      .regex(/[!@#$%^&*()_+\-=[\]{}|;:,.<>?]/, t("add_contact.validation.password_special")),
  });
  type FormValues = z.infer<typeof schema>;

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { first_name: "", last_name: "", email: "", username: "", password: "" },
  });

  useEffect(() => {
    if (open) reset();
  }, [open, reset]);

  async function onSubmit(values: FormValues) {
    try {
      await create.mutateAsync({
        ...values,
        role: "customer",
        is_active: true,
        customer_id: customerId,
      });
      qc.invalidateQueries({ queryKey: ["customer-users", customerId] });
      toast.success(t("add_contact.toast.added"));
      setOpen(false);
    } catch (err) {
      toast.error(apiError(err, t("add_contact.toast.add_error")));
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button size="sm" variant="secondary">
          <UserPlus /> {t("add_contact.trigger")}
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("add_contact.title")}</DialogTitle>
          <DialogDescription>
            {customerName
              ? t("add_contact.description_with_customer", { customerName })
              : t("add_contact.description_no_customer")}
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label htmlFor="c-first">{t("add_contact.field_first_name")}</Label>
              <Input id="c-first" placeholder="Jane" {...register("first_name")} />
              {errors.first_name && (
                <p className="text-xs text-destructive">{errors.first_name.message}</p>
              )}
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="c-last">{t("add_contact.field_last_name")}</Label>
              <Input id="c-last" placeholder="Roe" {...register("last_name")} />
              {errors.last_name && (
                <p className="text-xs text-destructive">{errors.last_name.message}</p>
              )}
            </div>
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label htmlFor="c-email">{t("add_contact.field_email")}</Label>
              <Input id="c-email" placeholder="jane@acme.com" {...register("email")} />
              {errors.email && (
                <p className="text-xs text-destructive">{errors.email.message}</p>
              )}
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="c-username">{t("add_contact.field_username")}</Label>
              <Input id="c-username" placeholder="janeroe" {...register("username")} />
              {errors.username && (
                <p className="text-xs text-destructive">{errors.username.message}</p>
              )}
            </div>
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="c-password">{t("add_contact.field_password")}</Label>
            <Input id="c-password" type="password" placeholder="••••••••" {...register("password")} />
            {errors.password ? (
              <p className="text-xs text-destructive">{errors.password.message}</p>
            ) : (
              <p className="text-xs text-muted-foreground">
                {t("add_contact.password_hint")}
              </p>
            )}
          </div>
          <DialogFooter>
            <DialogClose asChild>
              <Button type="button" variant="ghost">
                {t("actions.cancel", { ns: "common" })}
              </Button>
            </DialogClose>
            <Button type="submit" disabled={create.isPending}>
              {create.isPending ? t("add_contact.adding") : t("add_contact.submit")}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
