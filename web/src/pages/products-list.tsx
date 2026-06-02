import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { Search, ChevronLeft, ChevronRight, Package } from "lucide-react";
import { useTranslation } from "react-i18next";
import { useProducts, type ProductFilters } from "@/features/products/api";
import { ProductFormDialog } from "@/features/products/product-form-dialog";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/misc";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { relativeTime } from "@/lib/utils";

const ALL = "__all__";

export function ProductsListPage() {
  const { t } = useTranslation("products");
  const navigate = useNavigate();
  const [filters, setFilters] = useState<ProductFilters>({
    page: 1,
    page_size: 15,
  });
  const { data, isLoading, isFetching } = useProducts(filters);

  const set = (patch: Partial<ProductFilters>) =>
    setFilters((f) => ({ ...f, page: 1, ...patch }));

  return (
    <div className="w-full">
      <div className="mb-6 flex flex-wrap items-end justify-between gap-4">
        <div>
          <div className="font-mono text-xs uppercase tracking-[0.25em] text-muted-foreground">
            {t("list.eyebrow")}
          </div>
          <h1 className="mt-1 text-3xl">{t("list.title")}</h1>
        </div>
        <ProductFormDialog />
      </div>

      {/* Filter bar */}
      <div className="mb-4 flex flex-wrap items-center gap-3">
        <div className="relative min-w-56 flex-1">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="pl-9"
            placeholder={t("list.search_placeholder")}
            value={filters.search ?? ""}
            onChange={(e) => set({ search: e.target.value })}
          />
        </div>
        <Select
          value={filters.status ?? ALL}
          onValueChange={(v) => set({ status: v === ALL ? undefined : v })}
        >
          <SelectTrigger className="w-40">
            <SelectValue placeholder={t("list.status_placeholder")} />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={ALL}>{t("list.all_statuses")}</SelectItem>
            <SelectItem value="active">{t("list.status_active")}</SelectItem>
            <SelectItem value="inactive">{t("list.status_inactive")}</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <Card className="overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border text-left font-mono text-[11px] uppercase tracking-wider text-muted-foreground">
              <th className="px-4 py-3 font-medium">{t("list.col_name")}</th>
              <th className="px-4 py-3 font-medium">{t("list.col_code")}</th>
              <th className="px-4 py-3 font-medium">{t("list.col_category")}</th>
              <th className="px-4 py-3 font-medium">{t("list.col_version")}</th>
              <th className="px-4 py-3 font-medium">{t("list.col_status")}</th>
              <th className="px-4 py-3 text-right font-medium">{t("list.col_created")}</th>
            </tr>
          </thead>
          <tbody>
            {isLoading ? (
              Array.from({ length: 6 }).map((_, i) => (
                <tr key={i} className="border-b border-border/60">
                  {Array.from({ length: 6 }).map((__, j) => (
                    <td key={j} className="px-4 py-3.5">
                      <Skeleton className="h-4 w-full" />
                    </td>
                  ))}
                </tr>
              ))
            ) : data && data.items.length > 0 ? (
              data.items.map((p) => (
                <tr
                  key={p.id}
                  onClick={() => navigate(`/products/${p.id}`)}
                  className="group cursor-pointer border-b border-border/60 transition-colors last:border-0 hover:bg-accent/50"
                >
                  <td className="px-4 py-3.5">
                    <div className="font-medium text-foreground group-hover:text-primary">
                      {p.name}
                    </div>
                  </td>
                  <td className="px-4 py-3.5">
                    {p.code ? (
                      <span className="font-mono text-xs text-primary/80">
                        {p.code}
                      </span>
                    ) : (
                      <span className="text-muted-foreground">—</span>
                    )}
                  </td>
                  <td className="px-4 py-3.5 text-muted-foreground">
                    {p.category || "—"}
                  </td>
                  <td className="px-4 py-3.5 font-mono text-xs text-muted-foreground">
                    {p.version || "—"}
                  </td>
                  <td className="px-4 py-3.5">
                    <Badge tone={p.status === "active" ? "green" : "slate"}>
                      {p.status || "—"}
                    </Badge>
                  </td>
                  <td className="px-4 py-3.5 text-right font-mono text-xs text-muted-foreground">
                    {relativeTime(p.created_at)}
                  </td>
                </tr>
              ))
            ) : (
              <tr>
                <td colSpan={6} className="px-4 py-16 text-center">
                  <Package className="mx-auto size-8 text-muted-foreground/40" />
                  <p className="mt-3 text-sm text-muted-foreground">
                    {t("list.empty")}
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
            {t("list.pagination.summary", {
              total: data.total,
              page: data.page,
              total_pages: data.total_pages,
            })}
            {isFetching && t("list.pagination.syncing")}
          </div>
          <div className="flex gap-2">
            <Button
              variant="secondary"
              size="sm"
              disabled={filters.page <= 1}
              onClick={() => setFilters((f) => ({ ...f, page: f.page - 1 }))}
            >
              <ChevronLeft /> {t("list.pagination.prev")}
            </Button>
            <Button
              variant="secondary"
              size="sm"
              disabled={data.page >= data.total_pages}
              onClick={() => setFilters((f) => ({ ...f, page: f.page + 1 }))}
            >
              {t("list.pagination.next")} <ChevronRight />
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}
