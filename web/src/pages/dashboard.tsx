import { Link } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { Inbox, Loader2, CheckCircle2, AlertTriangle, ArrowUpRight } from "lucide-react";
import { useTicketStats } from "@/features/tickets/api";
import { useReveal } from "@/lib/use-reveal";
import { Card } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/misc";
import { CountUp } from "@/components/count-up";
import { useAuth } from "@/lib/auth";

const CARDS = [
  { key: "open_tickets", labelKey: "cards.open", icon: Inbox, tone: "text-primary" },
  {
    key: "in_progress_tickets",
    labelKey: "cards.in_progress",
    icon: Loader2,
    tone: "text-sky-400",
  },
  {
    key: "resolved_tickets",
    labelKey: "cards.resolved",
    icon: CheckCircle2,
    tone: "text-emerald-400",
  },
  {
    key: "overdue_tickets",
    labelKey: "cards.overdue",
    icon: AlertTriangle,
    tone: "text-red-400",
  },
] as const;

export function DashboardPage() {
  const { t } = useTranslation("dashboard");
  const { user } = useAuth();
  const { data, isLoading } = useTicketStats();
  const ref = useReveal(isLoading ? "loading" : "ready");

  return (
    <div ref={ref} className="w-full">
      <div data-reveal className="mb-8">
        <div className="font-mono text-xs uppercase tracking-[0.25em] text-muted-foreground">
          {t("overview")}
        </div>
        <h1 className="mt-1 text-3xl">
          {t("greeting", { name: user?.first_name || user?.username })}
        </h1>
        <p className="mt-1 text-sm text-muted-foreground">
          {t("subtitle")}
        </p>
      </div>

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {CARDS.map((c) => (
          <Card key={c.key} data-reveal className="relative overflow-hidden p-5">
            <div className="grid-texture pointer-events-none absolute inset-0 opacity-40" />
            <div className="relative flex items-start justify-between">
              <span className="font-mono text-[11px] uppercase tracking-widest text-muted-foreground">
                {t(c.labelKey)}
              </span>
              <c.icon className={`size-4 ${c.tone}`} />
            </div>
            <div className="relative mt-4 font-display text-4xl font-bold tabular-nums">
              {isLoading ? (
                <Skeleton className="h-9 w-16" />
              ) : (
                <CountUp value={(data?.[c.key as keyof typeof data] as number) ?? 0} />
              )}
            </div>
          </Card>
        ))}
      </div>

      <Card data-reveal className="mt-6 flex items-center justify-between p-5">
        <div>
          <div className="font-semibold">{t("queue.title")}</div>
          <div className="text-sm text-muted-foreground">
            {isLoading
              ? t("queue.total_loading")
              : t("queue.total", { count: data?.total_tickets ?? 0 })}
          </div>
        </div>
        <Link
          to="/tickets"
          className="inline-flex items-center gap-1.5 rounded-md border border-border bg-secondary px-4 py-2 text-sm font-medium transition-colors hover:bg-accent"
        >
          {t("queue.cta")} <ArrowUpRight className="size-4" />
        </Link>
      </Card>
    </div>
  );
}
