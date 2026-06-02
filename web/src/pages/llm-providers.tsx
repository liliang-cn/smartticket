import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import {
  Sparkles,
  Plus,
  Pencil,
  Trash2,
  FlaskConical,
  Loader2,
} from "lucide-react";
import { toast } from "sonner";
import { useTranslation } from "react-i18next";
import {
  useProviders,
  useCreateProvider,
  useUpdateProvider,
  useDeleteProvider,
  useTestProvider,
} from "@/features/llm/api";
import type { LLMProvider, LLMTaskType, ProviderInput } from "@/features/llm/types";
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

/** Parse the JSON-encoded task_types string the backend returns. */
function parseTaskTypes(raw: string | undefined): LLMTaskType[] {
  if (!raw) return [];
  try {
    const parsed = JSON.parse(raw);
    if (Array.isArray(parsed)) {
      return parsed.filter(
        (t): t is LLMTaskType => t === "chat" || t === "embedding"
      );
    }
    return [];
  } catch {
    return [];
  }
}

// --- Provider form ----------------------------------------------------------

const schema = z.object({
  name: z.string().min(1).max(100),
  provider_type: z.string().min(1).max(100),
  api_endpoint: z.string().min(1).max(500),
  api_key: z.string().optional(),
  model: z.string().min(1).max(200),
  task_chat: z.boolean(),
  task_embedding: z.boolean(),
  dimensions: z.string().optional(),
  max_tokens: z.string().optional(),
  temperature: z.string().optional(),
  is_default: z.boolean(),
  is_enabled: z.boolean(),
});
type FormValues = z.infer<typeof schema>;

const YES = "yes";
const NO = "no";

function defaultsFor(provider?: LLMProvider): FormValues {
  const tasks = parseTaskTypes(provider?.task_types);
  return {
    name: provider?.name ?? "",
    provider_type: provider?.provider_type ?? "openai-compatible",
    api_endpoint: provider?.api_endpoint ?? "",
    api_key: "",
    model: provider?.model ?? "",
    task_chat: tasks.includes("chat"),
    task_embedding: tasks.includes("embedding"),
    dimensions:
      provider?.dimensions != null ? String(provider.dimensions) : "1024",
    max_tokens: provider?.max_tokens != null ? String(provider.max_tokens) : "",
    temperature:
      provider?.temperature != null ? String(provider.temperature) : "",
    is_default: provider?.is_default ?? false,
    is_enabled: provider?.is_enabled ?? true,
  };
}

function ProviderFormDialog({
  provider,
  trigger,
}: {
  provider?: LLMProvider;
  trigger?: React.ReactNode;
}) {
  const { t } = useTranslation("llm");
  const [open, setOpen] = useState(false);
  const isEdit = provider != null;
  const create = useCreateProvider();
  const update = useUpdateProvider(provider?.id ?? 0);
  const pending = isEdit ? update.isPending : create.isPending;

  const {
    register,
    handleSubmit,
    setValue,
    watch,
    reset,
    formState: { errors },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: defaultsFor(provider),
  });

  useEffect(() => {
    if (open) reset(defaultsFor(provider));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  const taskChat = watch("task_chat");
  const taskEmbedding = watch("task_embedding");

  async function onSubmit(values: FormValues) {
    const task_types: LLMTaskType[] = [];
    if (values.task_chat) task_types.push("chat");
    if (values.task_embedding) task_types.push("embedding");
    if (task_types.length === 0) {
      toast.error(t("validation.atLeastOneTask"));
      return;
    }

    const payload: ProviderInput = {
      name: values.name,
      provider_type: values.provider_type,
      api_endpoint: values.api_endpoint,
      model: values.model,
      task_types,
      is_default: values.is_default,
      is_enabled: values.is_enabled,
    };
    // Leave api_key out when blank so the backend keeps the existing key.
    if (values.api_key && values.api_key.trim()) {
      payload.api_key = values.api_key.trim();
    }
    if (values.task_embedding && values.dimensions) {
      const d = Number(values.dimensions);
      if (!Number.isNaN(d)) payload.dimensions = d;
    }
    if (values.max_tokens) {
      const m = Number(values.max_tokens);
      if (!Number.isNaN(m)) payload.max_tokens = m;
    }
    if (values.temperature) {
      const temp = Number(values.temperature);
      if (!Number.isNaN(temp)) payload.temperature = temp;
    }

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
      toast.error(
        apiError(
          err,
          isEdit ? t("toast.updateFailed") : t("toast.createFailed")
        )
      );
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        {trigger ?? (
          <Button>
            <Plus /> {t("newProvider")}
          </Button>
        )}
      </DialogTrigger>
      <DialogContent className="max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>
            {isEdit ? t("form.titleEdit") : t("form.titleCreate")}
          </DialogTitle>
          <DialogDescription>{t("form.description")}</DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label htmlFor="p-name">{t("form.name")}</Label>
              <Input id="p-name" placeholder="DeepSeek chat" {...register("name")} />
              {errors.name && (
                <p className="text-xs text-destructive">
                  {t("validation.nameRequired")}
                </p>
              )}
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="p-type">{t("form.providerType")}</Label>
              <Input
                id="p-type"
                placeholder="openai-compatible"
                {...register("provider_type")}
              />
              {errors.provider_type && (
                <p className="text-xs text-destructive">
                  {t("validation.typeRequired")}
                </p>
              )}
            </div>
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="p-endpoint">{t("form.apiEndpoint")}</Label>
            <Input
              id="p-endpoint"
              placeholder="https://api.deepseek.com/v1"
              {...register("api_endpoint")}
            />
            {errors.api_endpoint && (
              <p className="text-xs text-destructive">
                {t("validation.endpointRequired")}
              </p>
            )}
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label htmlFor="p-model">{t("form.model")}</Label>
              <Input id="p-model" placeholder="deepseek-chat" {...register("model")} />
              {errors.model && (
                <p className="text-xs text-destructive">
                  {t("validation.modelRequired")}
                </p>
              )}
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="p-key">{t("form.apiKey")}</Label>
              <Input
                id="p-key"
                type="password"
                placeholder={
                  isEdit ? t("form.apiKeyPlaceholderEdit") : "sk-…"
                }
                {...register("api_key")}
              />
            </div>
          </div>

          <div className="space-y-1.5">
            <Label>{t("form.taskTypes")}</Label>
            <div className="flex flex-wrap gap-4 rounded-md border border-input bg-background/60 px-3 py-2.5">
              <label className="flex cursor-pointer items-center gap-2 text-sm">
                <input
                  type="checkbox"
                  className="size-4 accent-primary"
                  checked={taskChat}
                  onChange={(e) => setValue("task_chat", e.target.checked)}
                />
                {t("taskLabels.chat")}
              </label>
              <label className="flex cursor-pointer items-center gap-2 text-sm">
                <input
                  type="checkbox"
                  className="size-4 accent-primary"
                  checked={taskEmbedding}
                  onChange={(e) => setValue("task_embedding", e.target.checked)}
                />
                {t("taskLabels.embedding")}
              </label>
            </div>
          </div>

          <div className="grid grid-cols-3 gap-4">
            {taskEmbedding && (
              <div className="space-y-1.5">
                <Label htmlFor="p-dims">{t("form.dimensions")}</Label>
                <Input
                  id="p-dims"
                  type="number"
                  placeholder="1024"
                  {...register("dimensions")}
                />
              </div>
            )}
            <div className="space-y-1.5">
              <Label htmlFor="p-maxtok">{t("form.maxTokens")}</Label>
              <Input
                id="p-maxtok"
                type="number"
                placeholder="optional"
                {...register("max_tokens")}
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="p-temp">{t("form.temperature")}</Label>
              <Input
                id="p-temp"
                type="number"
                step="0.1"
                placeholder="optional"
                {...register("temperature")}
              />
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label>{t("form.default")}</Label>
              <Select
                value={watch("is_default") ? YES : NO}
                onValueChange={(v) => setValue("is_default", v === YES)}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value={YES}>{t("form.optionDefault")}</SelectItem>
                  <SelectItem value={NO}>{t("form.optionNotDefault")}</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label>{t("form.status")}</Label>
              <Select
                value={watch("is_enabled") ? YES : NO}
                onValueChange={(v) => setValue("is_enabled", v === YES)}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value={YES}>{t("form.optionEnabled")}</SelectItem>
                  <SelectItem value={NO}>{t("form.optionDisabled")}</SelectItem>
                </SelectContent>
              </Select>
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
                ? isEdit
                  ? t("form.savingPending")
                  : t("form.creatingPending")
                : isEdit
                  ? t("form.submitEdit")
                  : t("form.submitCreate")}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

// --- Page -------------------------------------------------------------------

export function LLMProvidersPage() {
  const { t } = useTranslation("llm");
  const { data: providers, isLoading } = useProviders();
  const deleteProvider = useDeleteProvider();
  const test = useTestProvider();
  const ref = useReveal<HTMLDivElement>();

  const [toDelete, setToDelete] = useState<{ id: number; name: string } | null>(
    null
  );
  // Track which row currently has a test in flight.
  const [testingId, setTestingId] = useState<number | null>(null);

  async function confirmDelete() {
    if (!toDelete) return;
    try {
      await deleteProvider.mutateAsync(toDelete.id);
      toast.success(t("toast.deleted"));
      setToDelete(null);
    } catch (err) {
      toast.error(apiError(err, t("toast.deleteFailed")));
    }
  }

  async function runTest(id: number) {
    setTestingId(id);
    try {
      const r = await test.mutateAsync(id);
      const summary = `chat:${r.chat_ok} embed:${r.embedding_ok} cortex:${r.cortex_ok} (${r.latency_ms}ms)`;
      if (r.error || !(r.chat_ok || r.embedding_ok || r.cortex_ok)) {
        toast.error(r.error ? `${summary} — ${r.error}` : summary);
      } else {
        toast.success(summary);
      }
    } catch (err) {
      toast.error(apiError(err, t("toast.testFailed")));
    } finally {
      setTestingId(null);
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
        <ProviderFormDialog />
      </div>

      <Card data-reveal className="overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border text-left font-mono text-[11px] uppercase tracking-wider text-muted-foreground">
              <th className="px-4 py-3 font-medium">{t("table.name")}</th>
              <th className="px-4 py-3 font-medium">{t("table.type")}</th>
              <th className="px-4 py-3 font-medium">{t("table.tasks")}</th>
              <th className="px-4 py-3 font-medium">{t("table.model")}</th>
              <th className="px-4 py-3 font-medium">{t("table.endpoint")}</th>
              <th className="px-4 py-3 font-medium">{t("table.dims")}</th>
              <th className="px-4 py-3 font-medium">{t("table.key")}</th>
              <th className="px-4 py-3 font-medium">{t("table.flags")}</th>
              <th className="px-4 py-3" />
            </tr>
          </thead>
          <tbody>
            {isLoading ? (
              Array.from({ length: 4 }).map((_, i) => (
                <tr key={i} className="border-b border-border/60">
                  {Array.from({ length: 9 }).map((__, j) => (
                    <td key={j} className="px-4 py-3.5">
                      <Skeleton className="h-4 w-full" />
                    </td>
                  ))}
                </tr>
              ))
            ) : providers && providers.length > 0 ? (
              providers.map((p) => {
                const tasks = parseTaskTypes(p.task_types);
                const isTesting = testingId === p.id;
                return (
                  <tr
                    key={p.id}
                    className="border-b border-border/60 transition-colors last:border-0 hover:bg-accent/50"
                  >
                    <td className="px-4 py-3.5 font-medium text-foreground">
                      {p.name}
                    </td>
                    <td className="px-4 py-3.5 font-mono text-xs text-muted-foreground">
                      {p.provider_type}
                    </td>
                    <td className="px-4 py-3.5">
                      <div className="flex flex-wrap gap-1">
                        {tasks.length > 0 ? (
                          tasks.map((taskType) => (
                            <Badge
                              key={taskType}
                              tone={taskType === "chat" ? "blue" : "amber"}
                            >
                              {t(`taskLabels.${taskType}`)}
                            </Badge>
                          ))
                        ) : (
                          <span className="font-mono text-xs text-muted-foreground/60">
                            —
                          </span>
                        )}
                      </div>
                    </td>
                    <td className="px-4 py-3.5 font-mono text-xs text-muted-foreground">
                      {p.model}
                    </td>
                    <td className="max-w-48 truncate px-4 py-3.5 font-mono text-xs text-muted-foreground">
                      {p.api_endpoint}
                    </td>
                    <td className="px-4 py-3.5 font-mono text-xs text-muted-foreground">
                      {p.dimensions ?? "—"}
                    </td>
                    <td className="px-4 py-3.5 font-mono text-xs text-muted-foreground">
                      {p.api_key_masked || "—"}
                    </td>
                    <td className="px-4 py-3.5">
                      <div className="flex flex-wrap gap-1">
                        {p.is_default && (
                          <Badge tone="amber">{t("badges.default")}</Badge>
                        )}
                        <Badge tone={p.is_enabled ? "green" : "slate"}>
                          {p.is_enabled
                            ? t("badges.enabled")
                            : t("badges.disabled")}
                        </Badge>
                      </div>
                    </td>
                    <td className="px-2 py-3.5">
                      <div className="flex items-center justify-end gap-1">
                        <Button
                          variant="outline"
                          size="sm"
                          disabled={isTesting}
                          onClick={() => runTest(p.id)}
                          title={t("actions.testTitle")}
                        >
                          {isTesting ? (
                            <Loader2 className="animate-spin" />
                          ) : (
                            <FlaskConical />
                          )}
                          {t("actions.test")}
                        </Button>
                        <ProviderFormDialog
                          provider={p}
                          trigger={
                            <Button
                              variant="ghost"
                              size="icon"
                              title={t("actions.editTitle")}
                            >
                              <Pencil />
                            </Button>
                          }
                        />
                        <Button
                          variant="ghost"
                          size="icon"
                          title={t("actions.deleteTitle")}
                          onClick={() =>
                            setToDelete({ id: p.id, name: p.name })
                          }
                        >
                          <Trash2 />
                        </Button>
                      </div>
                    </td>
                  </tr>
                );
              })
            ) : (
              <tr>
                <td colSpan={9} className="px-4 py-16 text-center">
                  <Sparkles className="mx-auto size-8 text-muted-foreground/40" />
                  <p className="mt-3 text-sm text-muted-foreground">
                    {t("empty")}
                  </p>
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </Card>

      <Dialog
        open={toDelete != null}
        onOpenChange={(v) => !v && setToDelete(null)}
      >
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>{t("deleteDialog.title")}</DialogTitle>
            <DialogDescription>
              {t("deleteDialog.description", { name: toDelete?.name })}
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
              disabled={deleteProvider.isPending}
              onClick={confirmDelete}
            >
              {deleteProvider.isPending
                ? t("deleteDialog.confirmPending")
                : t("deleteDialog.confirm")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
