import { useEffect, useState } from "react";
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

const ENTITIES = [
  { value: "tickets", label: "Tickets" },
  { value: "knowledge_articles", label: "Knowledge articles" },
  { value: "users", label: "Users" },
  { value: "products", label: "Products" },
  { value: "services", label: "Services" },
  { value: "complete", label: "Complete (full database)" },
] as const;

const FORMATS = [
  { value: "csv", label: "CSV" },
  { value: "json", label: "JSON" },
  { value: "xml", label: "XML" },
  { value: "markdown", label: "Markdown" },
  { value: "sqlite", label: "SQLite" },
] as const;

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
      toast.success("Export job created");
      setOpen(false);
    } catch (err) {
      toast.error(apiError(err, "Could not create export job"));
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        {trigger ?? (
          <Button>
            <Download /> New export
          </Button>
        )}
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>New export job</DialogTitle>
          <DialogDescription>
            Queue an export of a data set. The job runs in the background; you
            can download the result once it completes.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div className="space-y-1.5">
            <Label>Data set</Label>
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
                {ENTITIES.map((e) => (
                  <SelectItem key={e.value} value={e.value}>
                    {e.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            {errors.type && (
              <p className="text-xs text-destructive">{errors.type.message}</p>
            )}
          </div>
          <div className="space-y-1.5">
            <Label>Format</Label>
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
                {FORMATS.map((f) => (
                  <SelectItem key={f.value} value={f.value}>
                    {f.label}
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
                Cancel
              </Button>
            </DialogClose>
            <Button type="submit" disabled={create.isPending}>
              {create.isPending ? "Creating…" : "Create export"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
