import { useState } from "react";
import { useNavigate } from "react-router-dom";
import {
  Search,
  ChevronLeft,
  ChevronRight,
  Users as UsersIcon,
  MoreHorizontal,
} from "lucide-react";
import { useUsers, useSetUserActive, type UserFilters } from "@/features/users/api";
import { UserFormDialog } from "@/features/users/user-form-dialog";
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
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { relativeTime } from "@/lib/utils";

const ALL = "__all__";
const ROLE_OPTIONS = ["admin", "engineer", "support", "sales", "customer"];

function fullName(u: { first_name: string; last_name: string; username: string }) {
  const name = `${u.first_name} ${u.last_name}`.trim();
  return name || u.username;
}

function ActiveToggle({ id, isActive }: { id: number; isActive: boolean }) {
  const set = useSetUserActive(id);
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          variant="ghost"
          size="icon"
          onClick={(e) => e.stopPropagation()}
        >
          <MoreHorizontal className="size-4" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" onClick={(e) => e.stopPropagation()}>
        <DropdownMenuItem
          disabled={set.isPending}
          onSelect={() => set.mutate(!isActive)}
        >
          {isActive ? "Deactivate" : "Activate"}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

export function UsersListPage() {
  const navigate = useNavigate();
  const [filters, setFilters] = useState<UserFilters>({
    page: 1,
    page_size: 15,
  });
  const { data, isLoading, isFetching } = useUsers(filters);

  const set = (patch: Partial<UserFilters>) =>
    setFilters((f) => ({ ...f, page: 1, ...patch }));

  return (
    <div className="w-full">
      <div className="mb-6 flex flex-wrap items-end justify-between gap-4">
        <div>
          <div className="font-mono text-xs uppercase tracking-[0.25em] text-muted-foreground">
            directory
          </div>
          <h1 className="mt-1 text-3xl">Users</h1>
        </div>
        <UserFormDialog />
      </div>

      {/* Filter bar */}
      <div className="mb-4 flex flex-wrap items-center gap-3">
        <div className="relative min-w-56 flex-1">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="pl-9"
            placeholder="Search name, email, username…"
            value={filters.search ?? ""}
            onChange={(e) => set({ search: e.target.value })}
          />
        </div>
        <Select
          value={filters.role ?? ALL}
          onValueChange={(v) => set({ role: v === ALL ? undefined : v })}
        >
          <SelectTrigger className="w-40">
            <SelectValue placeholder="Role" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={ALL}>All roles</SelectItem>
            {ROLE_OPTIONS.map((r) => (
              <SelectItem key={r} value={r} className="capitalize">
                {r}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <Card className="overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border text-left font-mono text-[11px] uppercase tracking-wider text-muted-foreground">
              <th className="px-4 py-3 font-medium">Name</th>
              <th className="px-4 py-3 font-medium">Email</th>
              <th className="px-4 py-3 font-medium">Role</th>
              <th className="px-4 py-3 font-medium">Active</th>
              <th className="px-4 py-3 font-medium">Created</th>
              <th className="px-4 py-3" />
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
              data.items.map((u) => (
                <tr
                  key={u.id}
                  onClick={() => navigate(`/users/${u.id}`)}
                  className="group cursor-pointer border-b border-border/60 transition-colors last:border-0 hover:bg-accent/50"
                >
                  <td className="px-4 py-3.5">
                    <div className="font-medium text-foreground group-hover:text-primary">
                      {fullName(u)}
                    </div>
                    <div className="font-mono text-xs text-muted-foreground">
                      {u.username}
                    </div>
                  </td>
                  <td className="px-4 py-3.5 font-mono text-xs text-muted-foreground">
                    {u.email}
                  </td>
                  <td className="px-4 py-3.5">
                    <Badge tone="neutral" className="capitalize">
                      {u.role}
                    </Badge>
                  </td>
                  <td className="px-4 py-3.5">
                    <Badge tone={u.is_active ? "green" : "slate"}>
                      {u.is_active ? "active" : "inactive"}
                    </Badge>
                  </td>
                  <td className="px-4 py-3.5 font-mono text-xs text-muted-foreground">
                    {u.created_at ? relativeTime(u.created_at) : "—"}
                  </td>
                  <td className="px-2 py-3.5 text-right">
                    <ActiveToggle id={u.id} isActive={u.is_active} />
                  </td>
                </tr>
              ))
            ) : (
              <tr>
                <td colSpan={6} className="px-4 py-16 text-center">
                  <UsersIcon className="mx-auto size-8 text-muted-foreground/40" />
                  <p className="mt-3 text-sm text-muted-foreground">
                    No users match these filters.
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
            {data.total} users · page {data.page}/{data.total_pages}
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
