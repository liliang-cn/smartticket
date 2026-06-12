import { BarChart3, ExternalLink, MousePointerClick, Users, Eye } from "lucide-react";
import { useAnalyticsSummary, type AnalyticsBucket } from "@/features/analytics/api";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/misc";
import { CountUp } from "@/components/count-up";

function MetricCard({
  label,
  value,
  icon: Icon,
  loading,
}: {
  label: string;
  value: number;
  icon: typeof Eye;
  loading: boolean;
}) {
  return (
    <Card className="p-5">
      <div className="flex items-start justify-between">
        <span className="font-mono text-[11px] uppercase tracking-widest text-muted-foreground">
          {label}
        </span>
        <Icon className="size-4 text-primary" />
      </div>
      <div className="mt-4 font-display text-4xl font-bold tabular-nums">
        {loading ? <Skeleton className="h-9 w-16" /> : <CountUp value={value} />}
      </div>
    </Card>
  );
}

function BucketList({ title, items }: { title: string; items: AnalyticsBucket[] }) {
  return (
    <Card className="p-5">
      <div className="mb-4 flex items-center justify-between">
        <h2 className="text-base font-semibold">{title}</h2>
        <Badge tone="slate">{items.length}</Badge>
      </div>
      {items.length === 0 ? (
        <p className="text-sm text-muted-foreground">No data yet.</p>
      ) : (
        <div className="space-y-3">
          {items.map((item) => (
            <div key={`${title}-${item.name}`} className="grid grid-cols-[1fr_auto] gap-3">
              <div className="min-w-0 truncate text-sm text-foreground">{item.name}</div>
              <div className="font-mono text-xs tabular-nums text-muted-foreground">
                {item.count}
              </div>
            </div>
          ))}
        </div>
      )}
    </Card>
  );
}

export function AnalyticsPage() {
  const { data, isLoading } = useAnalyticsSummary(30);

  return (
    <div className="w-full">
      <div className="mb-8">
        <div className="font-mono text-xs uppercase tracking-[0.25em] text-muted-foreground">
          Launch analytics
        </div>
        <h1 className="mt-1 text-3xl">Website traffic</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          Privacy-preserving pageviews, CTA clicks, sources, and recent visits for the last 30 days.
        </p>
      </div>

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <MetricCard label="Pageviews" value={data?.pageviews ?? 0} icon={Eye} loading={isLoading} />
        <MetricCard label="Visitors" value={data?.unique_visitors ?? 0} icon={Users} loading={isLoading} />
        <MetricCard label="Clicks" value={data?.clicks ?? 0} icon={MousePointerClick} loading={isLoading} />
        <MetricCard label="Events" value={data?.total_events ?? 0} icon={BarChart3} loading={isLoading} />
      </div>

      <div className="mt-6 grid gap-4 lg:grid-cols-2">
        {isLoading ? (
          <>
            <Card className="p-5"><Skeleton className="h-36 w-full" /></Card>
            <Card className="p-5"><Skeleton className="h-36 w-full" /></Card>
          </>
        ) : (
          <>
            <BucketList title="Top sources" items={data?.top_sources ?? []} />
            <BucketList title="Top referrers" items={data?.top_referrers ?? []} />
            <BucketList title="Top pages" items={data?.top_paths ?? []} />
            <BucketList title="CTA clicks" items={data?.top_targets ?? []} />
          </>
        )}
      </div>

      <Card className="mt-6 p-5">
        <div className="mb-4 flex items-center gap-2">
          <ExternalLink className="size-4 text-primary" />
          <h2 className="text-base font-semibold">Recent events</h2>
        </div>
        {isLoading ? (
          <Skeleton className="h-24 w-full" />
        ) : !data?.recent_events?.length ? (
          <p className="text-sm text-muted-foreground">No visits recorded yet.</p>
        ) : (
          <div className="overflow-hidden rounded-md border border-border">
            <table className="w-full text-left text-sm">
              <thead className="bg-muted/40 font-mono text-[11px] uppercase tracking-widest text-muted-foreground">
                <tr>
                  <th className="px-3 py-2">Type</th>
                  <th className="px-3 py-2">Source</th>
                  <th className="px-3 py-2">Path</th>
                  <th className="px-3 py-2">Device</th>
                </tr>
              </thead>
              <tbody>
                {data.recent_events.map((event, index) => (
                  <tr key={`${event.created_at}-${index}`} className="border-t border-border">
                    <td className="px-3 py-2">
                      <Badge tone={event.event_type === "click" ? "blue" : "green"}>
                        {event.event_type}
                      </Badge>
                    </td>
                    <td className="px-3 py-2 text-muted-foreground">{event.source || "direct"}</td>
                    <td className="max-w-[340px] truncate px-3 py-2">{event.path}</td>
                    <td className="px-3 py-2 text-muted-foreground">{event.device_type}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </Card>
    </div>
  );
}
