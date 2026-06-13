import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Plus } from "lucide-react";
import { toast } from "sonner";
import { useTranslation } from "react-i18next";
import { useCreateUser, type CreateUserInput } from "@/features/users/api";
import { useCustomers } from "@/features/customers/api";
import { useRoles } from "@/features/rbac/api";
import { useDepartments } from "@/features/departments/api";
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

// The "customer" role is the one external/customer-facing role and requires a
// customer org; every other role is operator/team-side. The selectable roles
// themselves are NOT hardcoded — they come from the RBAC roles configuration.
const CUSTOMER_ROLE = "customer";

const ACTIVE = "active";
const INACTIVE = "inactive";

interface UserFormDialogProps {
  /** Optional custom trigger. Defaults to a "New user" button. */
  trigger?: React.ReactNode;
}

export function UserFormDialog({ trigger }: UserFormDialogProps) {
  const { t } = useTranslation("users");
  const [open, setOpen] = useState(false);
  const create = useCreateUser();
  // Pull customers to populate the linked-customer selector.
  const { data: customers } = useCustomers({ page: 1, page_size: 100 });
  // Selectable roles come from the RBAC roles configuration, not a hardcoded
  // list — any role an admin creates under /rbac is assignable here.
  const { data: roles = [] } = useRoles();
  const { data: departments = [] } = useDepartments();

  const schema = z
    .object({
      email: z
        .string()
        .min(1, t("form.validation.email_required"))
        .email(t("form.validation.email_invalid")),
      username: z
        .string()
        .min(3, t("form.validation.username_min"))
        .max(50)
        .regex(/^[a-zA-Z0-9_-]+$/, t("form.validation.username_chars")),
      first_name: z
        .string()
        .min(1, t("form.validation.first_name_required"))
        .max(100),
      last_name: z
        .string()
        .min(1, t("form.validation.last_name_required"))
        .max(100),
      password: z
        .string()
        .min(8, t("form.validation.password_min"))
        .regex(/[A-Z]/, t("form.validation.password_uppercase"))
        .regex(/[a-z]/, t("form.validation.password_lowercase"))
        .regex(/\d/, t("form.validation.password_digit"))
        .regex(
          /[!@#$%^&*()_+\-=[\]{}|;:,.<>?]/,
          t("form.validation.password_special")
        ),
      role: z.string().min(1, t("form.validation.role_required")),
      is_active: z.boolean(),
      customer_id: z.string().optional(),
      department_id: z.string().optional(),
    })
    .refine((v) => v.role !== CUSTOMER_ROLE || !!v.customer_id, {
      message: t("form.validation.customer_required"),
      path: ["customer_id"],
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
      email: "",
      username: "",
      first_name: "",
      last_name: "",
      password: "",
      role: "engineer",
      is_active: true,
      customer_id: undefined,
      department_id: undefined,
    },
  });

  const role = watch("role");

  useEffect(() => {
    if (open) {
      reset({
        email: "",
        username: "",
        first_name: "",
        last_name: "",
        password: "",
        role: "engineer",
        is_active: true,
        customer_id: undefined,
        department_id: undefined,
      });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  async function onSubmit(values: FormValues) {
    const payload: CreateUserInput = {
      email: values.email,
      username: values.username,
      first_name: values.first_name,
      last_name: values.last_name,
      password: values.password,
      role: values.role,
      is_active: values.is_active,
      // Only the customer role carries a customer_id; team roles must not.
      customer_id:
        values.role === CUSTOMER_ROLE && values.customer_id
          ? Number(values.customer_id)
          : undefined,
      department_id:
        values.department_id && values.department_id !== "__none__"
          ? Number(values.department_id)
          : null,
    };
    try {
      await create.mutateAsync(payload);
      toast.success(t("toasts.created"));
      setOpen(false);
    } catch (err) {
      toast.error(apiError(err, t("toasts.create_error")));
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        {trigger ?? (
          <Button>
            <Plus /> {t("actions.new_user")}
          </Button>
        )}
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("form.title")}</DialogTitle>
          <DialogDescription>
            {t("form.description")}
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label htmlFor="u-first">{t("form.fields.first_name")}</Label>
              <Input id="u-first" placeholder="John" {...register("first_name")} />
              {errors.first_name && (
                <p className="text-xs text-destructive">
                  {errors.first_name.message}
                </p>
              )}
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="u-last">{t("form.fields.last_name")}</Label>
              <Input id="u-last" placeholder="Doe" {...register("last_name")} />
              {errors.last_name && (
                <p className="text-xs text-destructive">
                  {errors.last_name.message}
                </p>
              )}
            </div>
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label htmlFor="u-email">{t("form.fields.email")}</Label>
              <Input
                id="u-email"
                placeholder="john@acme.com"
                {...register("email")}
              />
              {errors.email && (
                <p className="text-xs text-destructive">{errors.email.message}</p>
              )}
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="u-username">{t("form.fields.username")}</Label>
              <Input id="u-username" placeholder="johndoe" {...register("username")} />
              {errors.username && (
                <p className="text-xs text-destructive">
                  {errors.username.message}
                </p>
              )}
            </div>
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="u-password">{t("form.fields.password")}</Label>
            <Input
              id="u-password"
              type="password"
              placeholder="••••••••"
              {...register("password")}
            />
            {errors.password ? (
              <p className="text-xs text-destructive">{errors.password.message}</p>
            ) : (
              <p className="text-xs text-muted-foreground">
                {t("form.password_hint")}
              </p>
            )}
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label>{t("form.fields.role")}</Label>
              <Select
                value={role}
                onValueChange={(v) => {
                  setValue("role", v);
                  if (v !== CUSTOMER_ROLE) setValue("customer_id", undefined);
                }}
              >
                <SelectTrigger>
                  <SelectValue placeholder={t("form.placeholders.role")} />
                </SelectTrigger>
                <SelectContent>
                  {roles.map((r) => (
                    <SelectItem key={r.name} value={r.name}>
                      {t(`roles.${r.name}`, r.name.charAt(0).toUpperCase() + r.name.slice(1))}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label>{t("form.fields.status")}</Label>
              <Select
                value={watch("is_active") ? ACTIVE : INACTIVE}
                onValueChange={(v) => setValue("is_active", v === ACTIVE)}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value={ACTIVE}>{t("form.status_options.active")}</SelectItem>
                  <SelectItem value={INACTIVE}>{t("form.status_options.inactive")}</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
          {role === CUSTOMER_ROLE && (
            <div className="space-y-1.5">
              <Label>{t("form.fields.customer")}</Label>
              <Select
                value={watch("customer_id") ?? ""}
                onValueChange={(v) => setValue("customer_id", v)}
              >
                <SelectTrigger>
                  <SelectValue placeholder={t("form.placeholders.customer")} />
                </SelectTrigger>
                <SelectContent>
                  {(customers?.items ?? []).map((c) => (
                    <SelectItem key={c.id} value={String(c.id)}>
                      {c.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              {errors.customer_id && (
                <p className="text-xs text-destructive">
                  {errors.customer_id.message}
                </p>
              )}
            </div>
          )}
          <div className="space-y-1.5">
            <Label>{t("form.fields.department")}</Label>
            <Select
              value={watch("department_id") ?? "__none__"}
              onValueChange={(v) => setValue("department_id", v)}
            >
              <SelectTrigger>
                <SelectValue placeholder={t("form.placeholders.department")} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="__none__">
                  {t("form.placeholders.department")}
                </SelectItem>
                {departments.map((d) => (
                  <SelectItem key={d.id} value={String(d.id)}>
                    {d.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <DialogFooter>
            <DialogClose asChild>
              <Button type="button" variant="ghost">
                {t("actions.cancel", { ns: "common" })}
              </Button>
            </DialogClose>
            <Button type="submit" disabled={create.isPending}>
              {create.isPending ? t("form.submit_pending") : t("form.submit")}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
