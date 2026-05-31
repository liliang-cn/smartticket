import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { Search, ChevronLeft, ChevronRight, Building2 } from "lucide-react";
import { useCustomers, type CustomerFilters } from "@/features/customers/api";
import { CustomerFormDialog } from "@/features/customers/customer-form-dialog";
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

export function CustomersListPage() {
  const navigate = useNavigate();
  const [filters, setFilters] = useState<CustomerFilters>({
    page: 1,
    page_size: 15,
  });
  const { data, isLoading, isFetching } = useCustomers(filters);

  const set = (patch: Partial<CustomerFilters>) =>
    setFilters((f) => ({ ...f, page: 1, ...patch }));

  return (
    <div className="w-full">
      <div className="mb-6 flex flex-wrap items-end justify-between gap-4">
        <div>
          <div className="font-mono text-xs uppercase tracking-[0.25em] text-muted-foreground">
            directory
          </div>
          <h1 className="mt-1 text-3xl">Customers</h1>
        </div>
        <CustomerFormDialog />
      </div>

      {/* Filter bar */}
      <div className="mb-4 flex flex-wrap items-center gap-3">
        <div className="relative min-w-56 flex-1">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="pl-9"
            placeholder="Search name, code, domain…"
            value={filters.search ?? ""}
            onChange={(e) => set({ search: e.target.value })}
          />
        </div>
        <Select
          value={
            filters.is_active === undefined
              ? ALL
              : filters.is_active
                ? "active"
                : "inactive"
          }
          onValueChange={(v) =>
            set({ is_active: v === ALL ? undefined : v === "active" })
          }
        >
          <SelectTrigger className="w-40">
            <SelectValue placeholder="Status" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={ALL}>All statuses</SelectItem>
            <SelectItem value="active">Active</SelectItem>
            <SelectItem value="inactive">Inactive</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <Card className="overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border text-left font-mono text-[11px] uppercase tracking-wider text-muted-foreground">
              <th className="px-4 py-3 font-medium">Name</th>
              <th className="px-4 py-3 font-medium">Code</th>
              <th className="px-4 py-3 font-medium">Domain</th>
              <th className="px-4 py-3 font-medium">Active</th>
              <th className="px-4 py-3 text-right font-medium">Created</th>
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
              data.items.map((c) => (
                <tr
                  key={c.id}
                  onClick={() => navigate(`/customers/${c.id}`)}
                  className="group cursor-pointer border-b border-border/60 transition-colors last:border-0 hover:bg-accent/50"
                >
                  <td className="px-4 py-3.5">
                    <div className="font-medium text-foreground group-hover:text-primary">
                      {c.name}
                    </div>
                  </td>
                  <td className="px-4 py-3.5">
                    {c.code ? (
                      <span className="font-mono text-xs text-primary/80">
                        {c.code}
                      </span>
                    ) : (
                      <span className="text-muted-foreground">—</span>
                    )}
                  </td>
                  <td className="px-4 py-3.5 font-mono text-xs text-muted-foreground">
                    {c.domain || "—"}
                  </td>
                  <td className="px-4 py-3.5">
                    <Badge tone={c.is_active ? "green" : "slate"}>
                      {c.is_active ? "active" : "inactive"}
                    </Badge>
                  </td>
                  <td className="px-4 py-3.5 text-right font-mono text-xs text-muted-foreground">
                    {relativeTime(c.created_at)}
                  </td>
                </tr>
              ))
            ) : (
              <tr>
                <td colSpan={5} className="px-4 py-16 text-center">
                  <Building2 className="mx-auto size-8 text-muted-foreground/40" />
                  <p className="mt-3 text-sm text-muted-foreground">
                    No customers match these filters.
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
            {data.total} customers · page {data.page}/{data.total_pages}
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
