import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import {
  Search,
  ChevronLeft,
  ChevronRight,
  BookOpen,
  Sparkles,
  X,
  Loader2,
} from "lucide-react";
import { toast } from "sonner";
import {
  useArticles,
  useKnowledgeSearch,
  type ArticleFilters,
  type SearchHit,
} from "@/features/knowledge/api";
import { apiError } from "@/lib/api";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/misc";
import { ArticleStatusBadge } from "@/features/knowledge/status-badge";
import { AskAiDialog } from "@/features/knowledge/ask-ai-dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { relativeTime } from "@/lib/utils";

const ALL = "__all__";

export function KnowledgeListPage() {
  const navigate = useNavigate();
  const { t } = useTranslation("knowledge");
  const [filters, setFilters] = useState<ArticleFilters>({
    page: 1,
    page_size: 15,
  });
  const { data, isLoading, isFetching } = useArticles(filters);

  // Semantic search: when an active query has produced results, we REPLACE
  // the normal article list with ranked hits until the box is cleared.
  const [semanticInput, setSemanticInput] = useState("");
  const [activeQuery, setActiveQuery] = useState<string | null>(null);
  const search = useKnowledgeSearch();
  const hits: SearchHit[] | null =
    activeQuery !== null ? search.data ?? null : null;
  const semanticActive = activeQuery !== null;

  const runSearch = () => {
    const q = semanticInput.trim();
    if (!q || search.isPending) return;
    setActiveQuery(q);
    search.mutate(
      { query: q },
      {
        onError: (err) => {
          toast.error(apiError(err, t("list.search_error")));
        },
      }
    );
  };

  const clearSearch = () => {
    setSemanticInput("");
    setActiveQuery(null);
    search.reset();
  };

  const set = (patch: Partial<ArticleFilters>) =>
    setFilters((f) => ({ ...f, page: 1, ...patch }));

  return (
    <div className="w-full">
      <div className="mb-6 flex flex-wrap items-end justify-between gap-4">
        <div>
          <div className="font-mono text-xs uppercase tracking-[0.25em] text-muted-foreground">
            {t("list.library_label")}
          </div>
          <h1 className="mt-1 text-3xl">{t("list.heading")}</h1>
        </div>
      </div>

      {/* Semantic search + Ask AI */}
      <div className="mb-4 flex flex-wrap items-center gap-3">
        <div className="relative min-w-56 flex-1">
          <Sparkles className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-primary" />
          <Input
            className="pl-9 pr-9"
            placeholder={t("list.semantic_placeholder")}
            value={semanticInput}
            onChange={(e) => setSemanticInput(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") {
                e.preventDefault();
                runSearch();
              }
            }}
          />
          {semanticActive && (
            <button
              type="button"
              onClick={clearSearch}
              className="absolute right-2.5 top-1/2 -translate-y-1/2 rounded p-1 text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
              aria-label={t("list.clear_search_aria")}
            >
              <X className="size-4" />
            </button>
          )}
        </div>
        <Button onClick={runSearch} disabled={search.isPending || !semanticInput.trim()}>
          {search.isPending ? (
            <>
              <Loader2 className="size-4 animate-spin" /> {t("list.searching")}
            </>
          ) : (
            <>
              <Search className="size-4" /> {t("list.search_button")}
            </>
          )}
        </Button>
        <AskAiDialog />
      </div>

      {/* Filter bar — hidden while a semantic query is active */}
      {!semanticActive && (
        <div className="mb-4 flex flex-wrap items-center gap-3">
          <div className="relative min-w-56 flex-1">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="pl-9"
              placeholder={t("list.filter_placeholder")}
              value={filters.search ?? ""}
              onChange={(e) => set({ search: e.target.value })}
            />
          </div>
          <Input
            className="w-44"
            placeholder={t("list.category_placeholder")}
            value={filters.category ?? ""}
            onChange={(e) => set({ category: e.target.value })}
          />
          <Select
            value={filters.status ?? ALL}
            onValueChange={(v) => set({ status: v === ALL ? undefined : v })}
          >
            <SelectTrigger className="w-40">
              <SelectValue placeholder={t("list.status_placeholder")} />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={ALL}>{t("list.all_statuses")}</SelectItem>
              {(["draft", "published", "archived"] as const).map((s) => (
                <SelectItem key={s} value={s}>
                  {t(`status.${s}`)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      )}

      {/* Ranked semantic results replace the list while a query is active */}
      {semanticActive ? (
        <Card className="overflow-hidden">
          {search.isPending ? (
            <div className="space-y-px">
              {Array.from({ length: 5 }).map((_, i) => (
                <div key={i} className="px-4 py-4">
                  <Skeleton className="h-4 w-1/3" />
                  <Skeleton className="mt-2 h-3 w-3/4" />
                </div>
              ))}
            </div>
          ) : hits && hits.length > 0 ? (
            <ul>
              {hits.map((h, i) => (
                <li
                  key={`${h.article_id}-${i}`}
                  onClick={() => navigate(`/knowledge/${h.article_id}`)}
                  className="group cursor-pointer border-b border-border/60 px-4 py-4 transition-colors last:border-0 hover:bg-accent/50"
                >
                  <div className="flex items-start justify-between gap-3">
                    <div className="min-w-0">
                      <div className="font-medium text-foreground group-hover:text-primary">
                        {h.title}
                      </div>
                      {h.snippet && (
                        <p className="mt-1 line-clamp-2 text-sm text-muted-foreground">
                          {h.snippet}
                        </p>
                      )}
                    </div>
                    <Badge tone="amber" className="shrink-0">
                      {Math.round(h.score * 100)}%
                    </Badge>
                  </div>
                </li>
              ))}
            </ul>
          ) : (
            <div className="px-4 py-16 text-center">
              <Sparkles className="mx-auto size-8 text-muted-foreground/40" />
              <p className="mt-3 text-sm text-muted-foreground">
                {t("list.no_matches", { query: activeQuery })}
              </p>
            </div>
          )}
        </Card>
      ) : (
      <Card className="overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border text-left font-mono text-[11px] uppercase tracking-wider text-muted-foreground">
              <th className="px-4 py-3 font-medium">{t("list.col_title")}</th>
              <th className="px-4 py-3 font-medium">{t("list.col_category")}</th>
              <th className="px-4 py-3 font-medium">{t("list.col_status")}</th>
              <th className="px-4 py-3 text-right font-medium">{t("list.col_views")}</th>
              <th className="px-4 py-3 text-right font-medium">{t("list.col_updated")}</th>
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
                    {t("list.no_articles")}
                  </p>
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </Card>
      )}

      {/* Pagination — list mode only */}
      {!semanticActive && data && data.total_pages > 1 && (
        <div className="mt-4 flex items-center justify-between">
          <div className="font-mono text-xs text-muted-foreground">
            {t("list.pagination_info", { total: data.total, page: data.page, totalPages: data.total_pages })}
            {isFetching && " " + t("list.syncing")}
          </div>
          <div className="flex gap-2">
            <Button
              variant="secondary"
              size="sm"
              disabled={filters.page <= 1}
              onClick={() => setFilters((f) => ({ ...f, page: f.page - 1 }))}
            >
              <ChevronLeft /> {t("list.prev")}
            </Button>
            <Button
              variant="secondary"
              size="sm"
              disabled={data.page >= data.total_pages}
              onClick={() => setFilters((f) => ({ ...f, page: f.page + 1 }))}
            >
              {t("list.next")} <ChevronRight />
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}
