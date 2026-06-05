import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { MessageSquareText, Plus, Pencil, Trash2 } from "lucide-react";
import { toast } from "sonner";
import {
  useMacros,
  useCreateMacro,
  useUpdateMacro,
  useDeleteMacro,
  type Macro,
} from "@/features/macros/api";
import { apiError } from "@/lib/api";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/misc";
import { useReveal } from "@/lib/use-reveal";
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

// --- Form dialog -------------------------------------------------------------

interface FormState {
  title: string;
  category: string;
  body: string;
  actions: string;
  shared: boolean;
}

function emptyForm(): FormState {
  return { title: "", category: "", body: "", actions: "", shared: true };
}

function macroToForm(macro: Macro): FormState {
  return {
    title: macro.title,
    category: macro.category,
    body: macro.body,
    actions: macro.actions ?? "",
    shared: macro.shared,
  };
}

function MacroFormDialog({
  macro,
  trigger,
}: {
  macro?: Macro;
  trigger?: React.ReactNode;
}) {
  const { t } = useTranslation("macros");
  const [open, setOpen] = useState(false);
  const isEdit = macro != null;
  const create = useCreateMacro();
  const update = useUpdateMacro(macro?.id ?? 0);
  const pending = isEdit ? update.isPending : create.isPending;

  const [form, setForm] = useState<FormState>(emptyForm);

  useEffect(() => {
    if (open) {
      setForm(isEdit && macro ? macroToForm(macro) : emptyForm());
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  function updateForm(patch: Partial<FormState>) {
    setForm((f) => ({ ...f, ...patch }));
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!form.title.trim()) {
      toast.error(t("validation.titleRequired"));
      return;
    }
    if (!form.body.trim()) {
      toast.error(t("validation.bodyRequired"));
      return;
    }
    const payload = {
      title: form.title.trim(),
      category: form.category.trim(),
      body: form.body.trim(),
      shared: form.shared,
      ...(form.actions.trim() ? { actions: form.actions.trim() } : {}),
    };
    try {
      if (isEdit) {
        await update.mutateAsync(payload);
        toast.success(t("toast.updated"));
      } else {
        await create.mutateAsync(payload);
        toast.success(t("toast.created"));
      }
      setOpen(false);
    } catch (err) {
      toast.error(apiError(err, isEdit ? t("toast.updateFailed") : t("toast.createFailed")));
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        {trigger ?? (
          <Button>
            <Plus /> {t("newMacro")}
          </Button>
        )}
      </DialogTrigger>
      <DialogContent className="max-h-[85vh] max-w-xl overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{isEdit ? t("form.titleEdit") : t("form.titleCreate")}</DialogTitle>
          <DialogDescription>{t("form.description")}</DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label htmlFor="m-title">{t("form.title")}</Label>
              <Input
                id="m-title"
                placeholder={t("form.titlePlaceholder")}
                value={form.title}
                onChange={(e) => updateForm({ title: e.target.value })}
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="m-category">{t("form.category")}</Label>
              <Input
                id="m-category"
                placeholder={t("form.categoryPlaceholder")}
                value={form.category}
                onChange={(e) => updateForm({ category: e.target.value })}
              />
            </div>
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="m-body">{t("form.body")}</Label>
            <textarea
              id="m-body"
              rows={5}
              placeholder={t("form.bodyPlaceholder")}
              value={form.body}
              onChange={(e) => updateForm({ body: e.target.value })}
              className="w-full resize-y rounded-md border border-input bg-background px-3 py-2 text-sm outline-none ring-offset-background placeholder:text-muted-foreground focus:ring-2 focus:ring-ring focus:ring-offset-2"
            />
            <p className="text-xs text-muted-foreground">{t("form.bodyHint")}</p>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label>{t("form.shared")}</Label>
              <Select
                value={form.shared ? "yes" : "no"}
                onValueChange={(v) => updateForm({ shared: v === "yes" })}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="yes">{t("visibility.shared")}</SelectItem>
                  <SelectItem value="no">{t("visibility.private")}</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="m-actions">{t("form.actions_field")}</Label>
              <Input
                id="m-actions"
                placeholder={t("form.actionsPlaceholder")}
                value={form.actions}
                onChange={(e) => updateForm({ actions: e.target.value })}
              />
            </div>
          </div>

          <DialogFooter>
            <DialogClose asChild>
              <Button type="button" variant="ghost">
                {t("actions.cancel", { ns: "common" })}
              </Button>
            </DialogClose>
            <Button type="submit" disabled={pending}>
              {pending
                ? isEdit ? t("form.savingPending") : t("form.creatingPending")
                : isEdit ? t("form.submitEdit") : t("form.submitCreate")}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

// --- Page --------------------------------------------------------------------

export function MacrosPage() {
  const { t } = useTranslation("macros");
  const { data: macros, isLoading } = useMacros();
  const deleteMacro = useDeleteMacro();
  const ref = useReveal<HTMLDivElement>();

  const [toDelete, setToDelete] = useState<{ id: number; title: string } | null>(null);

  async function confirmDelete() {
    if (!toDelete) return;
    try {
      await deleteMacro.mutateAsync(toDelete.id);
      toast.success(t("toast.deleted"));
      setToDelete(null);
    } catch (err) {
      toast.error(apiError(err, t("toast.deleteFailed")));
    }
  }

  return (
    <div ref={ref} className="w-full">
      <div data-reveal className="mb-6 flex flex-wrap items-end justify-between gap-4">
        <div>
          <div className="font-mono text-xs uppercase tracking-[0.25em] text-muted-foreground">
            {t("page.eyebrow")}
          </div>
          <h1 className="mt-1 text-3xl">{t("page.title")}</h1>
        </div>
        <MacroFormDialog />
      </div>

      <Card data-reveal className="overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border text-left font-mono text-[11px] uppercase tracking-wider text-muted-foreground">
              <th className="px-4 py-3 font-medium">{t("table.title")}</th>
              <th className="px-4 py-3 font-medium">{t("table.category")}</th>
              <th className="px-4 py-3 font-medium">{t("table.visibility")}</th>
              <th className="px-4 py-3 font-medium">{t("table.usage")}</th>
              <th className="px-4 py-3" />
            </tr>
          </thead>
          <tbody>
            {isLoading ? (
              Array.from({ length: 5 }).map((_, i) => (
                <tr key={i} className="border-b border-border/60">
                  {Array.from({ length: 5 }).map((__, j) => (
                    <td key={j} className="px-4 py-3.5">
                      <Skeleton className="h-4 w-full" />
                    </td>
                  ))}
                </tr>
              ))
            ) : macros && macros.length > 0 ? (
              macros.map((macro) => (
                <tr
                  key={macro.id}
                  className="border-b border-border/60 transition-colors last:border-0 hover:bg-accent/50"
                >
                  <td className="px-4 py-3.5 font-medium text-foreground">{macro.title}</td>
                  <td className="px-4 py-3.5 text-muted-foreground">
                    {macro.category || (
                      <span className="text-muted-foreground/40">—</span>
                    )}
                  </td>
                  <td className="px-4 py-3.5">
                    <Badge tone={macro.shared ? "blue" : "amber"}>
                      {macro.shared ? t("visibility.shared") : t("visibility.private")}
                    </Badge>
                  </td>
                  <td className="px-4 py-3.5 font-mono text-xs text-muted-foreground">
                    {macro.usage_count}
                  </td>
                  <td className="px-2 py-3.5">
                    <div className="flex items-center justify-end gap-1">
                      <MacroFormDialog
                        macro={macro}
                        trigger={
                          <Button variant="ghost" size="icon" title={t("form.titleEdit")}>
                            <Pencil />
                          </Button>
                        }
                      />
                      <Button
                        variant="ghost"
                        size="icon"
                        title={t("deleteDialog.title")}
                        onClick={() => setToDelete({ id: macro.id, title: macro.title })}
                      >
                        <Trash2 />
                      </Button>
                    </div>
                  </td>
                </tr>
              ))
            ) : (
              <tr>
                <td colSpan={5} className="px-4 py-16 text-center">
                  <MessageSquareText className="mx-auto size-8 text-muted-foreground/40" />
                  <p className="mt-3 text-sm text-muted-foreground">{t("empty")}</p>
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </Card>

      <Dialog open={toDelete != null} onOpenChange={(v) => !v && setToDelete(null)}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>{t("deleteDialog.title")}</DialogTitle>
            <DialogDescription>
              {t("deleteDialog.description", { title: toDelete?.title })}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <DialogClose asChild>
              <Button type="button" variant="ghost">
                {t("actions.cancel", { ns: "common" })}
              </Button>
            </DialogClose>
            <Button
              type="button"
              variant="destructive"
              disabled={deleteMacro.isPending}
              onClick={confirmDelete}
            >
              {deleteMacro.isPending ? t("deleteDialog.confirmPending") : t("deleteDialog.confirm")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
