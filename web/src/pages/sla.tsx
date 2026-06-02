import { useState } from "react";
import { useTranslation } from "react-i18next";
import {
  Search,
  ChevronLeft,
  ChevronRight,
  Gauge,
  ListChecks,
  MoreHorizontal,
} from "lucide-react";
import {
  useSLATemplates,
  useSLARules,
  useSetDefaultSLATemplate,
  useSetSLATemplateActive,
  useDeleteSLATemplate,
  useSetSLARuleActive,
  useDeleteSLARule,
  type SLAFilters,
  type SLATemplate,
  type SLARule,
} from "@/features/sla/api";
import { SLATemplateFormDialog } from "@/features/sla/sla-template-form-dialog";
import { SLARuleFormDialog } from "@/features/sla/sla-rule-form-dialog";
import { toast } from "sonner";
import { apiError } from "@/lib/api";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/misc";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { relativeTime } from "@/lib/utils";

type Section = "templates" | "rules";

function Pager({
  page,
  totalPages,
  total,
  noun,
  isFetching,
  onPrev,
  onNext,
}: {
  page: number;
  totalPages: number;
  total: number;
  noun: string;
  isFetching: boolean;
  onPrev: () => void;
  onNext: () => void;
}) {
  const { t } = useTranslation("sla");
  if (totalPages <= 1) return null;
  return (
    <div className="mt-4 flex items-center justify-between">
      <div className="font-mono text-xs text-muted-foreground">
        {t("pager.summary", { total, noun, page, totalPages })}
        {isFetching && ` ${t("pager.syncing")}`}
      </div>
      <div className="flex gap-2">
        <Button
          variant="secondary"
          size="sm"
          disabled={page <= 1}
          onClick={onPrev}
        >
          <ChevronLeft /> {t("pager.prev")}
        </Button>
        <Button
          variant="secondary"
          size="sm"
          disabled={page >= totalPages}
          onClick={onNext}
        >
          {t("pager.next")} <ChevronRight />
        </Button>
      </div>
    </div>
  );
}

function TableSkeleton({ cols }: { cols: number }) {
  return (
    <>
      {Array.from({ length: 6 }).map((_, i) => (
        <tr key={i} className="border-b border-border/60">
          {Array.from({ length: cols }).map((__, j) => (
            <td key={j} className="px-4 py-3.5">
              <Skeleton className="h-4 w-full" />
            </td>
          ))}
        </tr>
      ))}
    </>
  );
}

// ── Templates section ─────────────────────────────────────────────────────────

function TemplateActions({ template }: { template: SLATemplate }) {
  const { t } = useTranslation("sla");
  const setDefault = useSetDefaultSLATemplate();
  const setActive = useSetSLATemplateActive();
  const remove = useDeleteSLATemplate();
  const [toDelete, setToDelete] = useState<{ id: number; label: string } | null>(
    null
  );
  const busy = setDefault.isPending || setActive.isPending || remove.isPending;

  async function confirmDelete() {
    if (!toDelete) return;
    try {
      await remove.mutateAsync(toDelete.id);
      toast.success(t("templates.toast.deleted"));
      setToDelete(null);
    } catch (err) {
      toast.error(apiError(err, t("templates.toast.delete_error")));
    }
  }

  return (
    <>
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="icon" onClick={(e) => e.stopPropagation()}>
          <MoreHorizontal className="size-4" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" onClick={(e) => e.stopPropagation()}>
        <SLATemplateFormDialog
          template={template}
          trigger={
            <DropdownMenuItem onSelect={(e) => e.preventDefault()}>
              {t("templates.actions.edit")}
            </DropdownMenuItem>
          }
        />
        {!template.is_default && (
          <DropdownMenuItem
            disabled={busy}
            onSelect={() => setDefault.mutate(template.id)}
          >
            {t("templates.actions.set_default")}
          </DropdownMenuItem>
        )}
        <DropdownMenuItem
          disabled={busy}
          onSelect={() =>
            setActive.mutate({ id: template.id, active: !template.is_active })
          }
        >
          {template.is_active ? t("templates.actions.deactivate") : t("templates.actions.activate")}
        </DropdownMenuItem>
        <DropdownMenuItem
          disabled={busy}
          className="text-destructive"
          onSelect={(e) => {
            e.preventDefault();
            setToDelete({ id: template.id, label: template.name });
          }}
        >
          {t("templates.actions.delete")}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>

    <ConfirmDialog
      open={!!toDelete}
      onOpenChange={(o) => !o && setToDelete(null)}
      title={t("templates.confirm_delete.title")}
      description={
        toDelete
          ? t("templates.confirm_delete.description", { name: toDelete.label })
          : undefined
      }
      pending={remove.isPending}
      onConfirm={confirmDelete}
    />
    </>
  );
}

function TemplatesSection() {
  const { t } = useTranslation("sla");
  const [filters, setFilters] = useState<SLAFilters>({
    page: 1,
    page_size: 15,
  });
  const { data, isLoading, isFetching } = useSLATemplates(filters);

  const set = (patch: Partial<SLAFilters>) =>
    setFilters((f) => ({ ...f, page: 1, ...patch }));

  return (
    <div>
      <div className="mb-4 flex flex-wrap items-center gap-3">
        <div className="relative min-w-56 flex-1">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="pl-9"
            placeholder={t("templates.search_placeholder")}
            value={filters.search ?? ""}
            onChange={(e) => set({ search: e.target.value })}
          />
        </div>
        <SLATemplateFormDialog />
      </div>

      <Card className="overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border text-left font-mono text-[11px] uppercase tracking-wider text-muted-foreground">
              <th className="px-4 py-3 font-medium">{t("templates.columns.name")}</th>
              <th className="px-4 py-3 font-medium">{t("templates.columns.default")}</th>
              <th className="px-4 py-3 font-medium">{t("templates.columns.active")}</th>
              <th className="px-4 py-3 font-medium">{t("templates.columns.levels")}</th>
              <th className="px-4 py-3 font-medium">{t("templates.columns.holidays")}</th>
              <th className="px-4 py-3 text-right font-medium">{t("templates.columns.updated")}</th>
              <th className="px-4 py-3" />
            </tr>
          </thead>
          <tbody>
            {isLoading ? (
              <TableSkeleton cols={7} />
            ) : data && data.items.length > 0 ? (
              data.items.map((tmpl) => (
                <tr
                  key={tmpl.id}
                  className="group border-b border-border/60 transition-colors last:border-0 hover:bg-accent/50"
                >
                  <td className="px-4 py-3.5">
                    <div className="font-medium text-foreground">{tmpl.name}</div>
                    {tmpl.description && (
                      <div className="mt-0.5 line-clamp-1 text-xs text-muted-foreground">
                        {tmpl.description}
                      </div>
                    )}
                  </td>
                  <td className="px-4 py-3.5">
                    {tmpl.is_default ? (
                      <Badge tone="amber">{t("templates.default_badge")}</Badge>
                    ) : (
                      <span className="text-muted-foreground">—</span>
                    )}
                  </td>
                  <td className="px-4 py-3.5">
                    <Badge tone={tmpl.is_active ? "green" : "slate"}>
                      {tmpl.is_active ? t("templates.active_badge") : t("templates.inactive_badge")}
                    </Badge>
                  </td>
                  <td className="px-4 py-3.5 font-mono text-xs text-muted-foreground">
                    {t("templates.levels_summary", {
                      priority: tmpl.priority_levels?.length ?? 0,
                      severity: tmpl.severity_levels?.length ?? 0,
                    })}
                  </td>
                  <td className="px-4 py-3.5 font-mono text-xs text-muted-foreground">
                    {tmpl.holidays?.length ?? 0}
                  </td>
                  <td className="px-4 py-3.5 text-right font-mono text-xs text-muted-foreground">
                    {relativeTime(tmpl.updated_at)}
                  </td>
                  <td className="px-2 py-3.5 text-right">
                    <TemplateActions template={tmpl} />
                  </td>
                </tr>
              ))
            ) : (
              <tr>
                <td colSpan={7} className="px-4 py-16 text-center">
                  <Gauge className="mx-auto size-8 text-muted-foreground/40" />
                  <p className="mt-3 text-sm text-muted-foreground">
                    {t("templates.empty")}
                  </p>
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </Card>

      {data && (
        <Pager
          page={data.page}
          totalPages={data.total_pages}
          total={data.total}
          noun={t("pager.noun_templates")}
          isFetching={isFetching}
          onPrev={() => setFilters((f) => ({ ...f, page: f.page - 1 }))}
          onNext={() => setFilters((f) => ({ ...f, page: f.page + 1 }))}
        />
      )}
    </div>
  );
}

// ── Rules section ─────────────────────────────────────────────────────────────

function RuleActions({ rule }: { rule: SLARule }) {
  const { t } = useTranslation("sla");
  const setActive = useSetSLARuleActive();
  const remove = useDeleteSLARule();
  const [toDelete, setToDelete] = useState<{ id: number; label: string } | null>(
    null
  );
  const busy = setActive.isPending || remove.isPending;

  async function confirmDelete() {
    if (!toDelete) return;
    try {
      await remove.mutateAsync(toDelete.id);
      toast.success(t("rules.toast.deleted"));
      setToDelete(null);
    } catch (err) {
      toast.error(apiError(err, t("rules.toast.delete_error")));
    }
  }

  return (
    <>
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="icon" onClick={(e) => e.stopPropagation()}>
          <MoreHorizontal className="size-4" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" onClick={(e) => e.stopPropagation()}>
        <SLARuleFormDialog
          rule={rule}
          trigger={
            <DropdownMenuItem onSelect={(e) => e.preventDefault()}>
              {t("rules.actions.edit")}
            </DropdownMenuItem>
          }
        />
        <DropdownMenuItem
          disabled={busy}
          onSelect={() =>
            setActive.mutate({ id: rule.id, active: !rule.is_active })
          }
        >
          {rule.is_active ? t("rules.actions.deactivate") : t("rules.actions.activate")}
        </DropdownMenuItem>
        <DropdownMenuItem
          disabled={busy}
          className="text-destructive"
          onSelect={(e) => {
            e.preventDefault();
            setToDelete({
              id: rule.id,
              label: `${rule.priority || t("rules.any_priority")} / ${
                rule.severity || t("rules.any_severity")
              }`,
            });
          }}
        >
          {t("rules.actions.delete")}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>

    <ConfirmDialog
      open={!!toDelete}
      onOpenChange={(o) => !o && setToDelete(null)}
      title={t("rules.confirm_delete.title")}
      description={
        toDelete
          ? t("rules.confirm_delete.description", { label: toDelete.label })
          : undefined
      }
      pending={remove.isPending}
      onConfirm={confirmDelete}
    />
    </>
  );
}

function RulesSection() {
  const { t } = useTranslation("sla");
  const [filters, setFilters] = useState<SLAFilters>({
    page: 1,
    page_size: 15,
  });
  const { data, isLoading, isFetching } = useSLARules(filters);

  const set = (patch: Partial<SLAFilters>) =>
    setFilters((f) => ({ ...f, page: 1, ...patch }));

  return (
    <div>
      <div className="mb-4 flex flex-wrap items-center gap-3">
        <div className="relative min-w-56 flex-1">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="pl-9"
            placeholder={t("rules.search_placeholder")}
            value={filters.search ?? ""}
            onChange={(e) => set({ search: e.target.value })}
          />
        </div>
        <SLARuleFormDialog />
      </div>

      <Card className="overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border text-left font-mono text-[11px] uppercase tracking-wider text-muted-foreground">
              <th className="px-4 py-3 font-medium">{t("rules.columns.template")}</th>
              <th className="px-4 py-3 font-medium">{t("rules.columns.priority")}</th>
              <th className="px-4 py-3 font-medium">{t("rules.columns.severity")}</th>
              <th className="px-4 py-3 font-medium">{t("rules.columns.response")}</th>
              <th className="px-4 py-3 font-medium">{t("rules.columns.resolution")}</th>
              <th className="px-4 py-3 font-medium">{t("rules.columns.hours")}</th>
              <th className="px-4 py-3 font-medium">{t("rules.columns.active")}</th>
              <th className="px-4 py-3" />
            </tr>
          </thead>
          <tbody>
            {isLoading ? (
              <TableSkeleton cols={8} />
            ) : data && data.items.length > 0 ? (
              data.items.map((r) => (
                <tr
                  key={r.id}
                  className="group border-b border-border/60 transition-colors last:border-0 hover:bg-accent/50"
                >
                  <td className="px-4 py-3.5 font-mono text-xs text-primary/80">
                    #{r.template_id}
                  </td>
                  <td className="px-4 py-3.5">
                    <Badge tone="neutral" className="capitalize">
                      {r.priority || "—"}
                    </Badge>
                  </td>
                  <td className="px-4 py-3.5">
                    <Badge tone="neutral" className="capitalize">
                      {r.severity || "—"}
                    </Badge>
                  </td>
                  <td className="px-4 py-3.5 font-mono text-xs text-muted-foreground">
                    {r.response_time}m
                  </td>
                  <td className="px-4 py-3.5 font-mono text-xs text-muted-foreground">
                    {r.resolution_time}m
                  </td>
                  <td className="px-4 py-3.5 font-mono text-xs text-muted-foreground">
                    {r.business_only ? t("rules.hours_business") : t("rules.hours_calendar")}
                  </td>
                  <td className="px-4 py-3.5">
                    <Badge tone={r.is_active ? "green" : "slate"}>
                      {r.is_active ? t("templates.active_badge") : t("templates.inactive_badge")}
                    </Badge>
                  </td>
                  <td className="px-2 py-3.5 text-right">
                    <RuleActions rule={r} />
                  </td>
                </tr>
              ))
            ) : (
              <tr>
                <td colSpan={8} className="px-4 py-16 text-center">
                  <ListChecks className="mx-auto size-8 text-muted-foreground/40" />
                  <p className="mt-3 text-sm text-muted-foreground">
                    {t("rules.empty")}
                  </p>
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </Card>

      {data && (
        <Pager
          page={data.page}
          totalPages={data.total_pages}
          total={data.total}
          noun={t("pager.noun_rules")}
          isFetching={isFetching}
          onPrev={() => setFilters((f) => ({ ...f, page: f.page - 1 }))}
          onNext={() => setFilters((f) => ({ ...f, page: f.page + 1 }))}
        />
      )}
    </div>
  );
}

// ── Page ──────────────────────────────────────────────────────────────────────

export function SLAPage() {
  const { t } = useTranslation("sla");
  const [section, setSection] = useState<Section>("templates");

  return (
    <div className="w-full">
      <div className="mb-6 flex flex-wrap items-end justify-between gap-4">
        <div>
          <div className="font-mono text-xs uppercase tracking-[0.25em] text-muted-foreground">
            {t("page.eyebrow")}
          </div>
          <h1 className="mt-1 text-3xl">{t("page.heading")}</h1>
        </div>
        <div className="flex gap-2">
          <Button
            variant={section === "templates" ? "default" : "secondary"}
            size="sm"
            onClick={() => setSection("templates")}
          >
            <Gauge /> {t("nav.templates")}
          </Button>
          <Button
            variant={section === "rules" ? "default" : "secondary"}
            size="sm"
            onClick={() => setSection("rules")}
          >
            <ListChecks /> {t("nav.rules")}
          </Button>
        </div>
      </div>

      {section === "templates" ? <TemplatesSection /> : <RulesSection />}
    </div>
  );
}
