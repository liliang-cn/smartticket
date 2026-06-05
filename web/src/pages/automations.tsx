import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  Workflow,
  Plus,
  Pencil,
  Trash2,
  ChevronUp,
  ChevronDown,
  X,
} from "lucide-react";
import { toast } from "sonner";
import {
  useAutomationRules,
  useCreateAutomationRule,
  useUpdateAutomationRule,
  useDeleteAutomationRule,
  useReorderAutomationRules,
  type AutomationRule,
  type Condition,
  type RuleAction,
} from "@/features/automations/api";
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

// --- Constants ---------------------------------------------------------------

const EVENTS = [
  "ticket.created",
  "ticket.updated",
  "message.created",
  "ticket.sla_warning",
] as const;

const CONDITION_FIELDS = [
  "status",
  "priority",
  "severity",
  "channel",
  "customer_email",
  "tags",
] as const;

const CONDITION_OPS = ["eq", "neq", "contains", "in", "gt", "lt"] as const;

const ACTION_TYPES = [
  "assign",
  "add_tag",
  "set_priority",
  "set_status",
  "set_severity",
  "notify",
  "send_email",
  "escalate",
  "ai_suggest",
  "ai_auto_reply",
  "close",
] as const;

// Action types that take a single text param
const SINGLE_PARAM_ACTIONS = new Set([
  "assign",
  "add_tag",
  "set_priority",
  "set_status",
  "set_severity",
  "notify",
  "send_email",
]);

// Param label by action type (for the single-param field)
function paramLabelForAction(type: string): string {
  switch (type) {
    case "assign": return "assignee_id";
    case "add_tag": return "tag";
    case "set_priority": return "priority";
    case "set_status": return "status";
    case "set_severity": return "severity";
    case "notify":
    case "send_email": return "message";
    default: return "value";
  }
}

function safeParseConditions(raw: string): Condition[] {
  try {
    const v = JSON.parse(raw);
    if (Array.isArray(v)) return v as Condition[];
  } catch {
    // ignore
  }
  return [];
}

function safeParseActions(raw: string): RuleAction[] {
  try {
    const v = JSON.parse(raw);
    if (Array.isArray(v)) return v as RuleAction[];
  } catch {
    // ignore
  }
  return [];
}

// --- Form dialog -------------------------------------------------------------

interface FormCondition {
  field: string;
  op: string;
  value: string;
}

interface FormAction {
  type: string;
  paramValue: string; // the single param value (serialized as {paramKey: paramValue})
}

interface FormState {
  name: string;
  description: string;
  event: string;
  match: "all" | "any";
  enabled: boolean;
  conditions: FormCondition[];
  actions: FormAction[];
}

function emptyForm(): FormState {
  return {
    name: "",
    description: "",
    event: "",
    match: "all",
    enabled: true,
    conditions: [],
    actions: [],
  };
}

function ruleToForm(rule: AutomationRule): FormState {
  const conditions = safeParseConditions(rule.conditions).map((c) => ({
    field: c.field,
    op: c.op,
    value: c.value,
  }));

  const actions = safeParseActions(rule.actions).map((a) => {
    const paramKey = a.params ? Object.keys(a.params)[0] ?? "" : "";
    const paramValue = paramKey && a.params ? a.params[paramKey] : "";
    return { type: a.type, paramValue };
  });

  return {
    name: rule.name,
    description: rule.description,
    event: rule.event,
    match: rule.match,
    enabled: rule.enabled,
    conditions,
    actions,
  };
}

function formToPayload(form: FormState) {
  const conditions: Condition[] = form.conditions
    .filter((c) => c.field && c.op)
    .map((c) => ({ field: c.field, op: c.op, value: c.value }));

  const actions: RuleAction[] = form.actions
    .filter((a) => a.type)
    .map((a) => {
      const result: RuleAction = { type: a.type };
      if (SINGLE_PARAM_ACTIONS.has(a.type) && a.paramValue.trim()) {
        const key = paramLabelForAction(a.type);
        result.params = { [key]: a.paramValue.trim() };
      }
      return result;
    });

  return {
    name: form.name,
    description: form.description,
    event: form.event,
    match: form.match,
    enabled: form.enabled,
    conditions: JSON.stringify(conditions),
    actions: JSON.stringify(actions),
  };
}

function RuleFormDialog({
  rule,
  trigger,
}: {
  rule?: AutomationRule;
  trigger?: React.ReactNode;
}) {
  const { t } = useTranslation("automations");
  const [open, setOpen] = useState(false);
  const isEdit = rule != null;
  const create = useCreateAutomationRule();
  const update = useUpdateAutomationRule(rule?.id ?? 0);
  const pending = isEdit ? update.isPending : create.isPending;

  const [form, setForm] = useState<FormState>(emptyForm);

  useEffect(() => {
    if (open) {
      setForm(isEdit && rule ? ruleToForm(rule) : emptyForm());
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  function updateForm(patch: Partial<FormState>) {
    setForm((f) => ({ ...f, ...patch }));
  }

  function updateCondition(i: number, patch: Partial<FormCondition>) {
    setForm((f) => {
      const conditions = [...f.conditions];
      conditions[i] = { ...conditions[i], ...patch };
      return { ...f, conditions };
    });
  }

  function addCondition() {
    setForm((f) => ({
      ...f,
      conditions: [...f.conditions, { field: "status", op: "eq", value: "" }],
    }));
  }

  function removeCondition(i: number) {
    setForm((f) => ({ ...f, conditions: f.conditions.filter((_, idx) => idx !== i) }));
  }

  function updateAction(i: number, patch: Partial<FormAction>) {
    setForm((f) => {
      const actions = [...f.actions];
      actions[i] = { ...actions[i], ...patch };
      return { ...f, actions };
    });
  }

  function addAction() {
    setForm((f) => ({
      ...f,
      actions: [...f.actions, { type: "assign", paramValue: "" }],
    }));
  }

  function removeAction(i: number) {
    setForm((f) => ({ ...f, actions: f.actions.filter((_, idx) => idx !== i) }));
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!form.name.trim()) {
      toast.error(t("validation.nameRequired"));
      return;
    }
    if (!form.event) {
      toast.error(t("validation.eventRequired"));
      return;
    }
    const payload = formToPayload(form);
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
            <Plus /> {t("newRule")}
          </Button>
        )}
      </DialogTrigger>
      <DialogContent className="max-h-[90vh] max-w-2xl overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{isEdit ? t("form.titleEdit") : t("form.titleCreate")}</DialogTitle>
          <DialogDescription>{t("form.description")}</DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-5">
          {/* Name + description */}
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label htmlFor="ar-name">{t("form.name")}</Label>
              <Input
                id="ar-name"
                placeholder={t("form.namePlaceholder")}
                value={form.name}
                onChange={(e) => updateForm({ name: e.target.value })}
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="ar-desc">{t("form.description_field")}</Label>
              <Input
                id="ar-desc"
                placeholder={t("form.descriptionPlaceholder")}
                value={form.description}
                onChange={(e) => updateForm({ description: e.target.value })}
              />
            </div>
          </div>

          {/* Event + match + enabled */}
          <div className="grid grid-cols-3 gap-4">
            <div className="space-y-1.5">
              <Label>{t("form.event")}</Label>
              <Select value={form.event} onValueChange={(v) => updateForm({ event: v })}>
                <SelectTrigger>
                  <SelectValue placeholder="—" />
                </SelectTrigger>
                <SelectContent>
                  {EVENTS.map((ev) => (
                    <SelectItem key={ev} value={ev}>
                      {t(`events.${ev}`)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label>{t("form.match")}</Label>
              <Select
                value={form.match}
                onValueChange={(v) => updateForm({ match: v as "all" | "any" })}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">{t("match.all")}</SelectItem>
                  <SelectItem value="any">{t("match.any")}</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label>{t("form.enabled")}</Label>
              <Select
                value={form.enabled ? "yes" : "no"}
                onValueChange={(v) => updateForm({ enabled: v === "yes" })}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="yes">Yes</SelectItem>
                  <SelectItem value="no">No</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>

          {/* Conditions */}
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <Label>{t("form.conditions")}</Label>
              <Button type="button" variant="ghost" size="sm" onClick={addCondition}>
                <Plus className="size-3.5" />
                {t("form.addCondition")}
              </Button>
            </div>
            {form.conditions.map((cond, i) => (
              <div key={i} className="flex items-center gap-2 rounded-md border border-border bg-background/40 px-3 py-2">
                <Select value={cond.field} onValueChange={(v) => updateCondition(i, { field: v })}>
                  <SelectTrigger className="w-36">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {CONDITION_FIELDS.map((f) => (
                      <SelectItem key={f} value={f}>
                        {t(`condition.fields.${f}`)}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <Select value={cond.op} onValueChange={(v) => updateCondition(i, { op: v })}>
                  <SelectTrigger className="w-28">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {CONDITION_OPS.map((op) => (
                      <SelectItem key={op} value={op}>
                        {t(`condition.ops.${op}`)}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <Input
                  className="flex-1"
                  placeholder={t("condition.valuePlaceholder")}
                  value={cond.value}
                  onChange={(e) => updateCondition(i, { value: e.target.value })}
                />
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  className="shrink-0"
                  onClick={() => removeCondition(i)}
                >
                  <X className="size-4" />
                </Button>
              </div>
            ))}
          </div>

          {/* Actions */}
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <Label>{t("form.actions_field")}</Label>
              <Button type="button" variant="ghost" size="sm" onClick={addAction}>
                <Plus className="size-3.5" />
                {t("form.addAction")}
              </Button>
            </div>
            {form.actions.map((action, i) => (
              <div key={i} className="flex items-center gap-2 rounded-md border border-border bg-background/40 px-3 py-2">
                <Select
                  value={action.type}
                  onValueChange={(v) => updateAction(i, { type: v, paramValue: "" })}
                >
                  <SelectTrigger className="w-40">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {ACTION_TYPES.map((at) => (
                      <SelectItem key={at} value={at}>
                        {t(`action.types.${at}`)}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                {SINGLE_PARAM_ACTIONS.has(action.type) && (
                  <Input
                    className="flex-1"
                    placeholder={t(`action.params.${paramLabelForAction(action.type)}`)}
                    value={action.paramValue}
                    onChange={(e) => updateAction(i, { paramValue: e.target.value })}
                  />
                )}
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  className="ml-auto shrink-0"
                  onClick={() => removeAction(i)}
                >
                  <X className="size-4" />
                </Button>
              </div>
            ))}
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

export function AutomationsPage() {
  const { t } = useTranslation("automations");
  const { data: rules, isLoading } = useAutomationRules();
  const deleteRule = useDeleteAutomationRule();
  const reorder = useReorderAutomationRules();
  const ref = useReveal<HTMLDivElement>();

  const [toDelete, setToDelete] = useState<{ id: number; name: string } | null>(null);

  async function confirmDelete() {
    if (!toDelete) return;
    try {
      await deleteRule.mutateAsync(toDelete.id);
      toast.success(t("toast.deleted"));
      setToDelete(null);
    } catch (err) {
      toast.error(apiError(err, t("toast.deleteFailed")));
    }
  }

  async function moveRule(rules: AutomationRule[], idx: number, dir: "up" | "down") {
    const reordered = [...rules];
    const swapIdx = dir === "up" ? idx - 1 : idx + 1;
    if (swapIdx < 0 || swapIdx >= reordered.length) return;
    [reordered[idx], reordered[swapIdx]] = [reordered[swapIdx], reordered[idx]];
    try {
      await reorder.mutateAsync(reordered.map((r) => r.id));
    } catch (err) {
      toast.error(apiError(err, t("toast.reorderFailed")));
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
        <RuleFormDialog />
      </div>

      <Card data-reveal className="overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border text-left font-mono text-[11px] uppercase tracking-wider text-muted-foreground">
              <th className="w-10 px-4 py-3 font-medium">{t("table.position")}</th>
              <th className="px-4 py-3 font-medium">{t("table.name")}</th>
              <th className="px-4 py-3 font-medium">{t("table.event")}</th>
              <th className="px-4 py-3 font-medium">{t("table.match")}</th>
              <th className="px-4 py-3 font-medium">{t("table.enabled")}</th>
              <th className="px-4 py-3" />
            </tr>
          </thead>
          <tbody>
            {isLoading ? (
              Array.from({ length: 4 }).map((_, i) => (
                <tr key={i} className="border-b border-border/60">
                  {Array.from({ length: 6 }).map((__, j) => (
                    <td key={j} className="px-4 py-3.5">
                      <Skeleton className="h-4 w-full" />
                    </td>
                  ))}
                </tr>
              ))
            ) : rules && rules.length > 0 ? (
              rules.map((rule, idx) => (
                <tr
                  key={rule.id}
                  className="border-b border-border/60 transition-colors last:border-0 hover:bg-accent/50"
                >
                  <td className="px-4 py-3.5 font-mono text-xs text-muted-foreground">
                    {rule.position + 1}
                  </td>
                  <td className="px-4 py-3.5">
                    <div className="font-medium text-foreground">{rule.name}</div>
                    {rule.description && (
                      <div className="mt-0.5 line-clamp-1 text-xs text-muted-foreground">
                        {rule.description}
                      </div>
                    )}
                  </td>
                  <td className="px-4 py-3.5">
                    <Badge tone="blue">{t(`events.${rule.event}` as Parameters<typeof t>[0])}</Badge>
                  </td>
                  <td className="px-4 py-3.5 font-mono text-xs text-muted-foreground">
                    {t(`match.${rule.match}` as Parameters<typeof t>[0])}
                  </td>
                  <td className="px-4 py-3.5">
                    <Badge tone={rule.enabled ? "green" : "slate"}>
                      {rule.enabled ? "on" : "off"}
                    </Badge>
                  </td>
                  <td className="px-2 py-3.5">
                    <div className="flex items-center justify-end gap-1">
                      <Button
                        variant="ghost"
                        size="icon"
                        disabled={idx === 0}
                        title={t("reorder.up")}
                        onClick={() => rules && moveRule(rules, idx, "up")}
                      >
                        <ChevronUp />
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        disabled={idx === (rules?.length ?? 0) - 1}
                        title={t("reorder.down")}
                        onClick={() => rules && moveRule(rules, idx, "down")}
                      >
                        <ChevronDown />
                      </Button>
                      <ToggleRuleButton rule={rule} />
                      <RuleFormDialog
                        rule={rule}
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
                        onClick={() => setToDelete({ id: rule.id, name: rule.name })}
                      >
                        <Trash2 />
                      </Button>
                    </div>
                  </td>
                </tr>
              ))
            ) : (
              <tr>
                <td colSpan={6} className="px-4 py-16 text-center">
                  <Workflow className="mx-auto size-8 text-muted-foreground/40" />
                  <p className="mt-3 text-sm text-muted-foreground">{t("empty")}</p>
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </Card>

      {/* Delete confirm */}
      <Dialog open={toDelete != null} onOpenChange={(v) => !v && setToDelete(null)}>
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
              disabled={deleteRule.isPending}
              onClick={confirmDelete}
            >
              {deleteRule.isPending ? t("deleteDialog.confirmPending") : t("deleteDialog.confirm")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

/** Isolated toggle button component so it can call its own hook per rule id. */
function ToggleRuleButton({ rule }: { rule: AutomationRule }) {
  const { t } = useTranslation("automations");
  const update = useUpdateAutomationRule(rule.id);

  async function toggle() {
    try {
      await update.mutateAsync({ enabled: !rule.enabled });
    } catch (err) {
      toast.error(apiError(err, t("toast.toggleFailed")));
    }
  }

  return (
    <Button
      variant="ghost"
      size="sm"
      disabled={update.isPending}
      onClick={toggle}
      title={rule.enabled ? t("toggle.disable") : t("toggle.enable")}
    >
      {rule.enabled ? t("toggle.disable") : t("toggle.enable")}
    </Button>
  );
}
