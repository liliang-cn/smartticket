import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Download } from "lucide-react";
import { toast } from "sonner";
import {
  useCreateExportJob,
  type CreateExportJobInput,
} from "@/features/data/api";
import { apiError } from "@/lib/api";
import { Button } from "@/components/ui/button";
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

const ENTITY_VALUES = [
  "tickets",
  "knowledge_articles",
  "users",
  "products",
  "services",
  "complete",
] as const;

const FORMAT_VALUES = ["csv", "json", "xml", "markdown", "sqlite"] as const;

const schema = z.object({
  type: z.enum([
    "tickets",
    "knowledge_articles",
    "users",
    "products",
    "services",
    "complete",
  ]),
  target_format: z.enum(["csv", "json", "xml", "markdown", "sqlite"]),
});
type FormValues = z.infer<typeof schema>;

interface ExportJobDialogProps {
  /** Optional custom trigger. Defaults to a "New export" button. */
  trigger?: React.ReactNode;
}

export function ExportJobDialog({ trigger }: ExportJobDialogProps) {
  const { t } = useTranslation("data");
  const [open, setOpen] = useState(false);
  const create = useCreateExportJob();

  const {
    handleSubmit,
    setValue,
    watch,
    reset,
    formState: { errors },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { type: "tickets", target_format: "csv" },
  });

  useEffect(() => {
    if (open) reset({ type: "tickets", target_format: "csv" });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  async function onSubmit(values: FormValues) {
    const payload: CreateExportJobInput = {
      type: values.type,
      target_format: values.target_format,
    };
    try {
      await create.mutateAsync(payload);
      toast.success(t("toasts.export_created"));
      setOpen(false);
    } catch (err) {
      toast.error(apiError(err, t("toasts.export_create_failed")));
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        {trigger ?? (
          <Button>
            <Download /> {t("actions.new_export")}
          </Button>
        )}
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("export_dialog.title")}</DialogTitle>
          <DialogDescription>
            {t("export_dialog.description")}
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div className="space-y-1.5">
            <Label>{t("export_dialog.dataset_label")}</Label>
            <Select
              value={watch("type")}
              onValueChange={(v) =>
                setValue("type", v as FormValues["type"])
              }
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {ENTITY_VALUES.map((value) => (
                  <SelectItem key={value} value={value}>
                    {t(`entity.${value}`)}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            {errors.type && (
              <p className="text-xs text-destructive">{errors.type.message}</p>
            )}
          </div>
          <div className="space-y-1.5">
            <Label>{t("export_dialog.format_label")}</Label>
            <Select
              value={watch("target_format")}
              onValueChange={(v) =>
                setValue("target_format", v as FormValues["target_format"])
              }
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {FORMAT_VALUES.map((value) => (
                  <SelectItem key={value} value={value}>
                    {t(`format.${value}`)}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            {errors.target_format && (
              <p className="text-xs text-destructive">
                {errors.target_format.message}
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
              {create.isPending ? t("export_dialog.submitting") : t("export_dialog.submit")}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
