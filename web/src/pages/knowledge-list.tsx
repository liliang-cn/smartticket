import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { Search, ChevronLeft, ChevronRight, BookOpen } from "lucide-react";
import { useArticles, type ArticleFilters } from "@/features/knowledge/api";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/misc";
import { ArticleStatusBadge } from "@/features/knowledge/status-badge";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { relativeTime } from "@/lib/utils";

const ALL = "__all__";

const STATUS_OPTIONS = [
  { value: "draft", label: "Draft" },
  { value: "published", label: "Published" },
  { value: "archived", label: "Archived" },
];

export function KnowledgeListPage() {
  const navigate = useNavigate();
  const [filters, setFilters] = useState<ArticleFilters>({
    page: 1,
    page_size: 15,
  });
  const { data, isLoading, isFetching } = useArticles(filters);

  const set = (patch: Partial<ArticleFilters>) =>
    setFilters((f) => ({ ...f, page: 1, ...patch }));

  return (
    <div className="w-full">
      <div className="mb-6 flex flex-wrap items-end justify-between gap-4">
        <div>
          <div className="font-mono text-xs uppercase tracking-[0.25em] text-muted-foreground">
            library
          </div>
          <h1 className="mt-1 text-3xl">Knowledge</h1>
        </div>
      </div>

      {/* Filter bar */}
      <div className="mb-4 flex flex-wrap items-center gap-3">
        <div className="relative min-w-56 flex-1">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="pl-9"
            placeholder="Search title, content…"
            value={filters.search ?? ""}
            onChange={(e) => set({ search: e.target.value })}
          />
        </div>
        <Input
          className="w-44"
          placeholder="Category"
          value={filters.category ?? ""}
          onChange={(e) => set({ category: e.target.value })}
        />
        <Select
          value={filters.status ?? ALL}
          onValueChange={(v) => set({ status: v === ALL ? undefined : v })}
        >
          <SelectTrigger className="w-40">
            <SelectValue placeholder="Status" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={ALL}>All statuses</SelectItem>
            {STATUS_OPTIONS.map((o) => (
              <SelectItem key={o.value} value={o.value}>
                {o.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <Card className="overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border text-left font-mono text-[11px] uppercase tracking-wider text-muted-foreground">
              <th className="px-4 py-3 font-medium">Title</th>
              <th className="px-4 py-3 font-medium">Category</th>
              <th className="px-4 py-3 font-medium">Status</th>
              <th className="px-4 py-3 text-right font-medium">Views</th>
              <th className="px-4 py-3 text-right font-medium">Updated</th>
            </tr>
          </thead>
          <tbody>
            {isLoading ? (
              Array.from({ length: 6 }).map((_, i) => (
                <tr key={i} className="border-b border-border/60">
                  {Array.from({ length: 5 }).map((__, j) => (
                    <td key={j} className="px-4 py-3.5">
                      <Skeleton className="h-4 w-full" />
                    </td>
                  ))}
                </tr>
              ))
            ) : data && data.items.length > 0 ? (
              data.items.map((a) => (
                <tr
                  key={a.id}
                  onClick={() => navigate(`/knowledge/${a.id}`)}
                  className="group cursor-pointer border-b border-border/60 transition-colors last:border-0 hover:bg-accent/50"
                >
                  <td className="px-4 py-3.5">
                    <div className="font-medium text-foreground group-hover:text-primary">
                      {a.title}
                    </div>
                  </td>
                  <td className="px-4 py-3.5">
                    {a.category ? (
                      <span className="font-mono text-xs text-primary/80">
                        {a.category}
                      </span>
                    ) : (
                      <span className="text-muted-foreground">—</span>
                    )}
                  </td>
                  <td className="px-4 py-3.5">
                    <ArticleStatusBadge status={a.status} />
                  </td>
                  <td className="px-4 py-3.5 text-right font-mono text-xs text-muted-foreground">
                    {a.view_count}
                  </td>
                  <td className="px-4 py-3.5 text-right font-mono text-xs text-muted-foreground">
                    {relativeTime(a.updated_at)}
                  </td>
                </tr>
              ))
            ) : (
              <tr>
                <td colSpan={5} className="px-4 py-16 text-center">
                  <BookOpen className="mx-auto size-8 text-muted-foreground/40" />
                  <p className="mt-3 text-sm text-muted-foreground">
                    No articles match these filters.
                  </p>
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </Card>

      {/* Pagination */}
      {data && data.total_pages > 1 && (
        <div className="mt-4 flex items-center justify-between">
          <div className="font-mono text-xs text-muted-foreground">
            {data.total} articles · page {data.page}/{data.total_pages}
            {isFetching && " · syncing…"}
          </div>
          <div className="flex gap-2">
            <Button
              variant="secondary"
              size="sm"
              disabled={filters.page <= 1}
              onClick={() => setFilters((f) => ({ ...f, page: f.page - 1 }))}
            >
              <ChevronLeft /> Prev
            </Button>
            <Button
              variant="secondary"
              size="sm"
              disabled={data.page >= data.total_pages}
              onClick={() => setFilters((f) => ({ ...f, page: f.page + 1 }))}
            >
              Next <ChevronRight />
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}
