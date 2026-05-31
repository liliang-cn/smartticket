import { useState } from "react";
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
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
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
  if (totalPages <= 1) return null;
  return (
    <div className="mt-4 flex items-center justify-between">
      <div className="font-mono text-xs text-muted-foreground">
        {total} {noun} · page {page}/{totalPages}
        {isFetching && " · syncing…"}
      </div>
      <div className="flex gap-2">
        <Button
          variant="secondary"
          size="sm"
          disabled={page <= 1}
          onClick={onPrev}
        >
          <ChevronLeft /> Prev
        </Button>
        <Button
          variant="secondary"
          size="sm"
          disabled={page >= totalPages}
          onClick={onNext}
        >
          Next <ChevronRight />
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
  const setDefault = useSetDefaultSLATemplate();
  const setActive = useSetSLATemplateActive();
  const remove = useDeleteSLATemplate();
  const busy = setDefault.isPending || setActive.isPending || remove.isPending;

  return (
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
              Edit
            </DropdownMenuItem>
          }
        />
        {!template.is_default && (
          <DropdownMenuItem
            disabled={busy}
            onSelect={() => setDefault.mutate(template.id)}
          >
            Set as default
          </DropdownMenuItem>
        )}
        <DropdownMenuItem
          disabled={busy}
          onSelect={() =>
            setActive.mutate({ id: template.id, active: !template.is_active })
          }
        >
          {template.is_active ? "Deactivate" : "Activate"}
        </DropdownMenuItem>
        <DropdownMenuItem
          disabled={busy}
          className="text-destructive"
          onSelect={() => remove.mutate(template.id)}
        >
          Delete
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function TemplatesSection() {
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
            placeholder="Search templates…"
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
              <th className="px-4 py-3 font-medium">Name</th>
              <th className="px-4 py-3 font-medium">Default</th>
              <th className="px-4 py-3 font-medium">Active</th>
              <th className="px-4 py-3 font-medium">Levels</th>
              <th className="px-4 py-3 font-medium">Holidays</th>
              <th className="px-4 py-3 text-right font-medium">Updated</th>
              <th className="px-4 py-3" />
            </tr>
          </thead>
          <tbody>
            {isLoading ? (
              <TableSkeleton cols={7} />
            ) : data && data.items.length > 0 ? (
              data.items.map((t) => (
                <tr
                  key={t.id}
                  className="group border-b border-border/60 transition-colors last:border-0 hover:bg-accent/50"
                >
                  <td className="px-4 py-3.5">
                    <div className="font-medium text-foreground">{t.name}</div>
                    {t.description && (
                      <div className="mt-0.5 line-clamp-1 text-xs text-muted-foreground">
                        {t.description}
                      </div>
                    )}
                  </td>
                  <td className="px-4 py-3.5">
                    {t.is_default ? (
                      <Badge tone="amber">default</Badge>
                    ) : (
                      <span className="text-muted-foreground">—</span>
                    )}
                  </td>
                  <td className="px-4 py-3.5">
                    <Badge tone={t.is_active ? "green" : "slate"}>
                      {t.is_active ? "active" : "inactive"}
                    </Badge>
                  </td>
                  <td className="px-4 py-3.5 font-mono text-xs text-muted-foreground">
                    {(t.priority_levels?.length ?? 0)}p ·{" "}
                    {(t.severity_levels?.length ?? 0)}s
                  </td>
                  <td className="px-4 py-3.5 font-mono text-xs text-muted-foreground">
                    {t.holidays?.length ?? 0}
                  </td>
                  <td className="px-4 py-3.5 text-right font-mono text-xs text-muted-foreground">
                    {relativeTime(t.updated_at)}
                  </td>
                  <td className="px-2 py-3.5 text-right">
                    <TemplateActions template={t} />
                  </td>
                </tr>
              ))
            ) : (
              <tr>
                <td colSpan={7} className="px-4 py-16 text-center">
                  <Gauge className="mx-auto size-8 text-muted-foreground/40" />
                  <p className="mt-3 text-sm text-muted-foreground">
                    No SLA templates yet.
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
          noun="templates"
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
  const setActive = useSetSLARuleActive();
  const remove = useDeleteSLARule();
  const busy = setActive.isPending || remove.isPending;

  return (
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
              Edit
            </DropdownMenuItem>
          }
        />
        <DropdownMenuItem
          disabled={busy}
          onSelect={() =>
            setActive.mutate({ id: rule.id, active: !rule.is_active })
          }
        >
          {rule.is_active ? "Deactivate" : "Activate"}
        </DropdownMenuItem>
        <DropdownMenuItem
          disabled={busy}
          className="text-destructive"
          onSelect={() => remove.mutate(rule.id)}
        >
          Delete
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function RulesSection() {
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
            placeholder="Search rules…"
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
              <th className="px-4 py-3 font-medium">Template</th>
              <th className="px-4 py-3 font-medium">Priority</th>
              <th className="px-4 py-3 font-medium">Severity</th>
              <th className="px-4 py-3 font-medium">Response</th>
              <th className="px-4 py-3 font-medium">Resolution</th>
              <th className="px-4 py-3 font-medium">Hours</th>
              <th className="px-4 py-3 font-medium">Active</th>
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
                    {r.business_only ? "business" : "calendar"}
                  </td>
                  <td className="px-4 py-3.5">
                    <Badge tone={r.is_active ? "green" : "slate"}>
                      {r.is_active ? "active" : "inactive"}
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
                    No SLA rules yet.
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
          noun="rules"
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
  const [section, setSection] = useState<Section>("templates");

  return (
    <div className="w-full">
      <div className="mb-6 flex flex-wrap items-end justify-between gap-4">
        <div>
          <div className="font-mono text-xs uppercase tracking-[0.25em] text-muted-foreground">
            policy
          </div>
          <h1 className="mt-1 text-3xl">Service levels</h1>
        </div>
        <div className="flex gap-2">
          <Button
            variant={section === "templates" ? "default" : "secondary"}
            size="sm"
            onClick={() => setSection("templates")}
          >
            <Gauge /> Templates
          </Button>
          <Button
            variant={section === "rules" ? "default" : "secondary"}
            size="sm"
            onClick={() => setSection("rules")}
          >
            <ListChecks /> Rules
          </Button>
        </div>
      </div>

      {section === "templates" ? <TemplatesSection /> : <RulesSection />}
    </div>
  );
}
