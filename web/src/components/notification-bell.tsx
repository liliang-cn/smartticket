import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { Bell } from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { cn, formatShortDateTime } from "@/lib/utils";
import {
  useMarkAllRead,
  useMarkRead,
  useNotifications,
  useUnreadCount,
  type Notification,
} from "@/features/notifications/api";

export function NotificationBell() {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const navigate = useNavigate();

  const { data: unreadCount = 0 } = useUnreadCount();
  const { data: notifications = [], isLoading } = useNotifications(false, open);
  const markRead = useMarkRead();
  const markAllRead = useMarkAllRead();

  const hasUnread = unreadCount > 0;
  const badge = unreadCount > 9 ? "9+" : String(unreadCount);

  function handleClick(n: Notification) {
    if (!n.is_read) markRead.mutate(n.id);
    setOpen(false);
    if (n.ref_type === "ticket" && n.ref_id) {
      navigate(`/tickets/${n.ref_id}`);
    }
  }

  return (
    <DropdownMenu open={open} onOpenChange={setOpen}>
      <DropdownMenuTrigger
        aria-label={t("notifications.aria")}
        className="relative grid size-9 place-items-center rounded-full border border-border bg-card/60 text-muted-foreground outline-none transition-colors hover:bg-accent hover:text-foreground"
      >
        <Bell className="size-4" />
        {hasUnread && (
          <span className="absolute -right-0.5 -top-0.5 grid min-w-4 place-items-center rounded-full bg-primary px-1 text-[10px] font-bold leading-4 text-primary-foreground">
            {badge}
          </span>
        )}
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-80 p-0">
        <div className="flex items-center justify-between px-3 py-2">
          <DropdownMenuLabel className="p-0">{t("notifications.title")}</DropdownMenuLabel>
          {hasUnread && (
            <button
              type="button"
              onClick={() => markAllRead.mutate()}
              disabled={markAllRead.isPending}
              className="text-xs font-medium text-primary outline-none transition-colors hover:text-primary/80 disabled:opacity-50"
            >
              {t("notifications.mark_all_read")}
            </button>
          )}
        </div>
        <DropdownMenuSeparator className="my-0" />
        <div className="max-h-96 overflow-auto">
          {isLoading ? (
            <div className="px-3 py-6 text-center text-sm text-muted-foreground">
              {t("notifications.loading")}
            </div>
          ) : notifications.length === 0 ? (
            <div className="px-3 py-6 text-center text-sm text-muted-foreground">
              {t("notifications.empty")}
            </div>
          ) : (
            notifications.map((n) => {
              const time = formatShortDateTime(n.created_at);
              return (
                <button
                  key={n.id}
                  type="button"
                  onClick={() => handleClick(n)}
                  className={cn(
                    "flex w-full items-start gap-2 px-3 py-2.5 text-left outline-none transition-colors hover:bg-accent",
                    !n.is_read && "bg-primary/5"
                  )}
                >
                  <span
                    className={cn(
                      "mt-1.5 size-1.5 shrink-0 rounded-full",
                      n.is_read ? "bg-transparent" : "bg-primary"
                    )}
                  />
                  <span className="min-w-0 flex-1">
                    <span
                      className={cn(
                        "block truncate text-sm",
                        n.is_read ? "font-normal" : "font-semibold"
                      )}
                    >
                      {n.title}
                    </span>
                    {n.body && (
                      <span className="mt-0.5 line-clamp-2 block text-xs text-muted-foreground">
                        {n.body}
                      </span>
                    )}
                    {time && (
                      <span className="mt-1 block font-mono text-[10px] uppercase tracking-wide text-muted-foreground/70">
                        {time}
                      </span>
                    )}
                  </span>
                </button>
              );
            })
          )}
        </div>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
