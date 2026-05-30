import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Plus } from "lucide-react";
import { toast } from "sonner";
import { useCreateUser, type CreateUserInput } from "@/features/users/api";
import { useCustomers } from "@/features/customers/api";
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

const ROLES = ["admin", "engineer", "customer"] as const;

const schema = z
  .object({
    email: z.string().min(1, "Email is required").email("Invalid email"),
    username: z
      .string()
      .min(3, "At least 3 characters")
      .max(50)
      .regex(/^[a-zA-Z0-9_-]+$/, "Letters, numbers, _ and - only"),
    first_name: z.string().min(1, "First name is required").max(100),
    last_name: z.string().min(1, "Last name is required").max(100),
    password: z
      .string()
      .min(8, "At least 8 characters")
      .regex(/[A-Z]/, "Needs an uppercase letter")
      .regex(/[a-z]/, "Needs a lowercase letter")
      .regex(/\d/, "Needs a digit")
      .regex(/[!@#$%^&*()_+\-=[\]{}|;:,.<>?]/, "Needs a special character"),
    role: z.enum(ROLES),
    is_active: z.boolean(),
    customer_id: z.string().optional(),
  })
  .refine((v) => v.role !== "customer" || !!v.customer_id, {
    message: "Select a customer for the customer role",
    path: ["customer_id"],
  });
type FormValues = z.infer<typeof schema>;

interface UserFormDialogProps {
  /** Optional custom trigger. Defaults to a "New user" button. */
  trigger?: React.ReactNode;
}

const ACTIVE = "active";
const INACTIVE = "inactive";

export function UserFormDialog({ trigger }: UserFormDialogProps) {
  const [open, setOpen] = useState(false);
  const create = useCreateUser();
  // Pull customers to populate the linked-customer selector.
  const { data: customers } = useCustomers({ page: 1, page_size: 100 });

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
        values.role === "customer" && values.customer_id
          ? Number(values.customer_id)
          : undefined,
    };
    try {
      await create.mutateAsync(payload);
      toast.success("User created");
      setOpen(false);
    } catch (err) {
      toast.error(apiError(err, "Could not create user"));
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        {trigger ?? (
          <Button>
            <Plus /> New user
          </Button>
        )}
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>New user</DialogTitle>
          <DialogDescription>
            Create an operator or a customer-side account.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label htmlFor="u-first">First name</Label>
              <Input id="u-first" placeholder="John" {...register("first_name")} />
              {errors.first_name && (
                <p className="text-xs text-destructive">
                  {errors.first_name.message}
                </p>
              )}
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="u-last">Last name</Label>
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
              <Label htmlFor="u-email">Email</Label>
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
              <Label htmlFor="u-username">Username</Label>
              <Input id="u-username" placeholder="johndoe" {...register("username")} />
              {errors.username && (
                <p className="text-xs text-destructive">
                  {errors.username.message}
                </p>
              )}
            </div>
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="u-password">Password</Label>
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
                At least 8 characters, with an uppercase letter, a digit and a
                special character.
              </p>
            )}
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label>Role</Label>
              <Select
                value={role}
                onValueChange={(v) => {
                  setValue("role", v as FormValues["role"]);
                  if (v !== "customer") setValue("customer_id", undefined);
                }}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="admin">Admin</SelectItem>
                  <SelectItem value="engineer">Engineer</SelectItem>
                  <SelectItem value="customer">Customer</SelectItem>
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
          {role === "customer" && (
            <div className="space-y-1.5">
              <Label>Customer</Label>
              <Select
                value={watch("customer_id") ?? ""}
                onValueChange={(v) => setValue("customer_id", v)}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Select a customer organization" />
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
          <DialogFooter>
            <DialogClose asChild>
              <Button type="button" variant="ghost">
                Cancel
              </Button>
            </DialogClose>
            <Button type="submit" disabled={create.isPending}>
              {create.isPending ? "Creating…" : "Create user"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
