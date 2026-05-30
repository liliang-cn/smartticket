import { ShieldCheck, KeyRound, Lock } from "lucide-react";
import { useRoles, usePermissions } from "@/features/rbac/api";
import type { RbacPermission } from "@/lib/types";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/misc";
import { useReveal } from "@/lib/use-reveal";

function groupByCategory(perms: RbacPermission[]): [string, RbacPermission[]][] {
  const map = new Map<string, RbacPermission[]>();
  for (const p of perms) {
    const key = p.category || "uncategorized";
    const arr = map.get(key) ?? [];
    arr.push(p);
    map.set(key, arr);
  }
  return Array.from(map.entries()).sort((a, b) => a[0].localeCompare(b[0]));
}

export function AccessPage() {
  const { data: roles, isLoading: rolesLoading } = useRoles();
  const { data: permissions, isLoading: permsLoading } = usePermissions();
  const ref = useReveal<HTMLDivElement>();

  const grouped = groupByCategory(permissions ?? []);

  return (
    <div ref={ref} className="mx-auto max-w-6xl">
      <div data-reveal className="mb-6">
        <div className="font-mono text-xs uppercase tracking-[0.25em] text-muted-foreground">
          access control
        </div>
        <h1 className="mt-1 text-3xl">Roles &amp; Permissions</h1>
      </div>

      {/* Roles */}
      <section data-reveal className="mb-10">
        <div className="mb-3 flex items-center gap-2">
          <ShieldCheck className="size-4 text-primary" />
          <h2 className="text-sm font-medium">Roles</h2>
          {roles && (
            <span className="font-mono text-xs text-muted-foreground">
              {roles.length}
            </span>
          )}
        </div>

        {rolesLoading ? (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {Array.from({ length: 3 }).map((_, i) => (
              <Card key={i} className="p-5">
                <Skeleton className="h-5 w-32" />
                <Skeleton className="mt-3 h-4 w-full" />
                <Skeleton className="mt-2 h-4 w-2/3" />
              </Card>
            ))}
          </div>
        ) : roles && roles.length > 0 ? (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {roles.map((role) => (
              <Card key={role.id} className="flex flex-col p-5">
                <div className="flex items-start justify-between gap-2">
                  <h3 className="font-medium capitalize text-foreground">
                    {role.name}
                  </h3>
                  {role.is_system && (
                    <Badge tone="amber">
                      <Lock className="size-3" /> system
                    </Badge>
                  )}
                </div>
                {role.description && (
                  <p className="mt-1.5 text-sm text-muted-foreground">
                    {role.description}
                  </p>
                )}
                <div className="mt-3 flex flex-1 flex-wrap items-end gap-1.5">
                  {role.permissions && role.permissions.length > 0 ? (
                    <>
                      {role.permissions.slice(0, 8).map((p) => (
                        <span
                          key={p.id}
                          className="rounded-sm bg-muted px-1.5 py-0.5 font-mono text-[10px] text-muted-foreground"
                          title={p.name}
                        >
                          {p.code}
                        </span>
                      ))}
                      {role.permissions.length > 8 && (
                        <span className="font-mono text-[10px] text-muted-foreground/70">
                          +{role.permissions.length - 8} more
                        </span>
                      )}
                    </>
                  ) : (
                    <span className="font-mono text-[11px] text-muted-foreground/60">
                      no permissions
                    </span>
                  )}
                </div>
              </Card>
            ))}
          </div>
        ) : (
          <Card className="py-16 text-center">
            <ShieldCheck className="mx-auto size-8 text-muted-foreground/40" />
            <p className="mt-3 text-sm text-muted-foreground">
              No roles defined.
            </p>
          </Card>
        )}
      </section>

      {/* Permissions */}
      <section data-reveal>
        <div className="mb-3 flex items-center gap-2">
          <KeyRound className="size-4 text-primary" />
          <h2 className="text-sm font-medium">Permissions</h2>
          {permissions && (
            <span className="font-mono text-xs text-muted-foreground">
              {permissions.length}
            </span>
          )}
        </div>

        {permsLoading ? (
          <Card className="p-5">
            <Skeleton className="h-4 w-40" />
            <Skeleton className="mt-3 h-4 w-full" />
            <Skeleton className="mt-2 h-4 w-3/4" />
          </Card>
        ) : grouped.length > 0 ? (
          <div className="space-y-5">
            {grouped.map(([category, perms]) => (
              <Card key={category} className="p-5">
                <div className="mb-3 flex items-center gap-2">
                  <span className="font-mono text-[11px] uppercase tracking-wider text-primary/80">
                    {category}
                  </span>
                  <span className="font-mono text-[11px] text-muted-foreground/60">
                    {perms.length}
                  </span>
                </div>
                <div className="grid gap-x-6 gap-y-2 sm:grid-cols-2 lg:grid-cols-3">
                  {perms.map((p) => (
                    <div key={p.id} className="min-w-0">
                      <div className="truncate font-mono text-xs text-foreground">
                        {p.code}
                      </div>
                      <div className="truncate text-[11px] text-muted-foreground">
                        {p.name}
                      </div>
                    </div>
                  ))}
                </div>
              </Card>
            ))}
          </div>
        ) : (
          <Card className="py-16 text-center">
            <KeyRound className="mx-auto size-8 text-muted-foreground/40" />
            <p className="mt-3 text-sm text-muted-foreground">
              No permissions defined.
            </p>
          </Card>
        )}
      </section>
    </div>
  );
}
