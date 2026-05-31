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
  name: z.string().min(1, "Name is required").max(100),
  provider_type: z.string().min(1, "Type is required").max(100),
  api_endpoint: z.string().min(1, "Endpoint is required").max(500),
  api_key: z.string().optional(),
  model: z.string().min(1, "Model is required").max(200),
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
      toast.error("Select at least one task type (chat or embedding)");
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
      const t = Number(values.temperature);
      if (!Number.isNaN(t)) payload.temperature = t;
    }

    try {
      if (isEdit) {
        await update.mutateAsync(payload);
        toast.success("Provider updated");
      } else {
        await create.mutateAsync(payload);
        toast.success("Provider created");
      }
      setOpen(false);
    } catch (err) {
      toast.error(
        apiError(
          err,
          isEdit ? "Could not update provider" : "Could not create provider"
        )
      );
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        {trigger ?? (
          <Button>
            <Plus /> New provider
          </Button>
        )}
      </DialogTrigger>
      <DialogContent className="max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{isEdit ? "Edit provider" : "New provider"}</DialogTitle>
          <DialogDescription>
            Configure a chat or embedding provider. Chat and embedding are
            independent — create one row each (e.g. DeepSeek for chat, Aliyun
            Bailian for embedding).
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label htmlFor="p-name">Name</Label>
              <Input id="p-name" placeholder="DeepSeek chat" {...register("name")} />
              {errors.name && (
                <p className="text-xs text-destructive">{errors.name.message}</p>
              )}
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="p-type">Provider type</Label>
              <Input
                id="p-type"
                placeholder="openai-compatible"
                {...register("provider_type")}
              />
              {errors.provider_type && (
                <p className="text-xs text-destructive">
                  {errors.provider_type.message}
                </p>
              )}
            </div>
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="p-endpoint">API endpoint</Label>
            <Input
              id="p-endpoint"
              placeholder="https://api.deepseek.com/v1"
              {...register("api_endpoint")}
            />
            {errors.api_endpoint && (
              <p className="text-xs text-destructive">
                {errors.api_endpoint.message}
              </p>
            )}
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label htmlFor="p-model">Model</Label>
              <Input id="p-model" placeholder="deepseek-chat" {...register("model")} />
              {errors.model && (
                <p className="text-xs text-destructive">{errors.model.message}</p>
              )}
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="p-key">API key</Label>
              <Input
                id="p-key"
                type="password"
                placeholder={
                  isEdit ? "Leave blank to keep existing" : "sk-…"
                }
                {...register("api_key")}
              />
            </div>
          </div>

          <div className="space-y-1.5">
            <Label>Task types</Label>
            <div className="flex flex-wrap gap-4 rounded-md border border-input bg-background/60 px-3 py-2.5">
              <label className="flex cursor-pointer items-center gap-2 text-sm">
                <input
                  type="checkbox"
                  className="size-4 accent-primary"
                  checked={taskChat}
                  onChange={(e) => setValue("task_chat", e.target.checked)}
                />
                Chat
              </label>
              <label className="flex cursor-pointer items-center gap-2 text-sm">
                <input
                  type="checkbox"
                  className="size-4 accent-primary"
                  checked={taskEmbedding}
                  onChange={(e) => setValue("task_embedding", e.target.checked)}
                />
                Embedding
              </label>
            </div>
          </div>

          <div className="grid grid-cols-3 gap-4">
            {taskEmbedding && (
              <div className="space-y-1.5">
                <Label htmlFor="p-dims">Dimensions</Label>
                <Input
                  id="p-dims"
                  type="number"
                  placeholder="1024"
                  {...register("dimensions")}
                />
              </div>
            )}
            <div className="space-y-1.5">
              <Label htmlFor="p-maxtok">Max tokens</Label>
              <Input
                id="p-maxtok"
                type="number"
                placeholder="optional"
                {...register("max_tokens")}
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="p-temp">Temperature</Label>
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
              <Label>Default</Label>
              <Select
                value={watch("is_default") ? YES : NO}
                onValueChange={(v) => setValue("is_default", v === YES)}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value={YES}>Default</SelectItem>
                  <SelectItem value={NO}>Not default</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label>Status</Label>
              <Select
                value={watch("is_enabled") ? YES : NO}
                onValueChange={(v) => setValue("is_enabled", v === YES)}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value={YES}>Enabled</SelectItem>
                  <SelectItem value={NO}>Disabled</SelectItem>
                </SelectContent>
              </Select>
            </div>
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
                  : "Create provider"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

// --- Page -------------------------------------------------------------------

export function LLMProvidersPage() {
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
      toast.success("Provider deleted");
      setToDelete(null);
    } catch (err) {
      toast.error(apiError(err, "Could not delete provider"));
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
      toast.error(apiError(err, "Test failed"));
    } finally {
      setTestingId(null);
    }
  }

  return (
    <div ref={ref} className="w-full">
      <div data-reveal className="mb-6 flex flex-wrap items-end justify-between gap-4">
        <div>
          <div className="font-mono text-xs uppercase tracking-[0.25em] text-muted-foreground">
            ai integration
          </div>
          <h1 className="mt-1 text-3xl">AI Providers</h1>
        </div>
        <ProviderFormDialog />
      </div>

      <Card data-reveal className="overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border text-left font-mono text-[11px] uppercase tracking-wider text-muted-foreground">
              <th className="px-4 py-3 font-medium">Name</th>
              <th className="px-4 py-3 font-medium">Type</th>
              <th className="px-4 py-3 font-medium">Tasks</th>
              <th className="px-4 py-3 font-medium">Model</th>
              <th className="px-4 py-3 font-medium">Endpoint</th>
              <th className="px-4 py-3 font-medium">Dims</th>
              <th className="px-4 py-3 font-medium">Key</th>
              <th className="px-4 py-3 font-medium">Flags</th>
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
                          tasks.map((t) => (
                            <Badge
                              key={t}
                              tone={t === "chat" ? "blue" : "amber"}
                            >
                              {t}
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
                        {p.is_default && <Badge tone="amber">default</Badge>}
                        <Badge tone={p.is_enabled ? "green" : "slate"}>
                          {p.is_enabled ? "enabled" : "disabled"}
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
                          title="Test connectivity"
                        >
                          {isTesting ? (
                            <Loader2 className="animate-spin" />
                          ) : (
                            <FlaskConical />
                          )}
                          Test
                        </Button>
                        <ProviderFormDialog
                          provider={p}
                          trigger={
                            <Button
                              variant="ghost"
                              size="icon"
                              title="Edit provider"
                            >
                              <Pencil />
                            </Button>
                          }
                        />
                        <Button
                          variant="ghost"
                          size="icon"
                          title="Delete provider"
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
                    No AI providers configured yet.
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
            <DialogTitle>Delete provider?</DialogTitle>
            <DialogDescription>
              This permanently removes the "{toDelete?.name}" provider
              configuration.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <DialogClose asChild>
              <Button type="button" variant="ghost">
                Cancel
              </Button>
            </DialogClose>
            <Button
              type="button"
              variant="destructive"
              disabled={deleteProvider.isPending}
              onClick={confirmDelete}
            >
              {deleteProvider.isPending ? "Deleting…" : "Delete"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
