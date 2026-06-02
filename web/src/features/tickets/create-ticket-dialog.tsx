import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Plus } from "lucide-react";
import { toast } from "sonner";
import { useTranslation } from "react-i18next";
import { useCreateTicket } from "@/features/tickets/api";
import { apiError } from "@/lib/api";
import {
  PRIORITY_OPTIONS,
} from "@/components/ticket-meta";
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

type FormValues = {
  title: string;
  description: string;
  priority: "low" | "medium" | "high" | "critical";
  requester_name: string;
  requester_email: string;
};

export function CreateTicketDialog() {
  const { t } = useTranslation("tickets");
  const { t: tCommon } = useTranslation("common");
  const [open, setOpen] = useState(false);
  const create = useCreateTicket();

  const schema = z.object({
    title: z.string().min(1, t("create.validation.title_required")).max(255),
    description: z.string().min(1, t("create.validation.description_required")),
    priority: z.enum(["low", "medium", "high", "critical"]),
    requester_name: z.string().min(1, t("create.validation.requester_name_required")),
    requester_email: z.string().email(t("create.validation.email_required")),
  });

  const {
    register,
    handleSubmit,
    setValue,
    watch,
    reset,
    formState: { errors },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { priority: "medium" },
  });

  async function onSubmit(values: FormValues) {
    try {
      await create.mutateAsync({ ...values, severity: "minor" });
      toast.success(t("create.toast_created"));
      reset({ priority: "medium" });
      setOpen(false);
    } catch (err) {
      toast.error(apiError(err, t("create.toast_create_error")));
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button>
          <Plus /> {t("create.btn_new_ticket")}
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("create.dialog_title")}</DialogTitle>
          <DialogDescription>
            {t("create.dialog_description")}
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div className="space-y-1.5">
            <Label htmlFor="t-title">{t("create.label_title")}</Label>
            <Input id="t-title" placeholder={t("create.placeholder_title")} {...register("title")} />
            {errors.title && (
              <p className="text-xs text-destructive">{errors.title.message}</p>
            )}
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="t-desc">{t("create.label_description")}</Label>
            <Textarea
              id="t-desc"
              placeholder={t("create.placeholder_description")}
              {...register("description")}
            />
            {errors.description && (
              <p className="text-xs text-destructive">
                {errors.description.message}
              </p>
            )}
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label>{t("create.label_priority")}</Label>
              <Select
                value={watch("priority")}
                onValueChange={(v) =>
                  setValue("priority", v as FormValues["priority"])
                }
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {PRIORITY_OPTIONS.map((p) => (
                    <SelectItem key={p} value={p}>
                      {tCommon(`enums.priority.${p}`)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="t-rname">{t("create.label_requester")}</Label>
              <Input
                id="t-rname"
                placeholder={t("create.placeholder_requester")}
                {...register("requester_name")}
              />
            </div>
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="t-remail">{t("create.label_requester_email")}</Label>
            <Input
              id="t-remail"
              type="email"
              placeholder="jane@acme.com"
              {...register("requester_email")}
            />
            {errors.requester_email && (
              <p className="text-xs text-destructive">
                {errors.requester_email.message}
              </p>
            )}
          </div>
          <DialogFooter>
            <DialogClose asChild>
              <Button type="button" variant="ghost">
                {t("create.btn_cancel")}
              </Button>
            </DialogClose>
            <Button type="submit" disabled={create.isPending}>
              {create.isPending ? t("create.btn_creating") : t("create.btn_create")}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
