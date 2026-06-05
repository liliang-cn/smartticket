import { useState } from "react";
import { useTranslation } from "react-i18next";
import { NavLink, Outlet, useNavigate } from "react-router-dom";
import {
  LayoutDashboard,
  Ticket,
  BookOpen,
  Building2,
  Users,
  ShieldCheck,
  Package,
  Layers,
  CreditCard,
  Timer,
  Database,
  Sparkles,
  Settings,
  PanelLeftClose,
  PanelLeftOpen,
  LogOut,
  Sun,
  Moon,
  Monitor,
  Workflow,
  MessageSquareText,
  UsersRound,
} from "lucide-react";
import { useAuth } from "@/lib/auth";
import { useBranding } from "@/lib/branding";
import { useTheme, type Theme } from "@/lib/theme";
import { useReveal } from "@/lib/use-reveal";
import { cn } from "@/lib/utils";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Avatar, AvatarFallback } from "@/components/ui/misc";
import { NotificationBell } from "@/components/notification-bell";
import { LanguageToggle } from "@/components/language-toggle";

interface NavItem {
  to: string;
  /** Translation key under the `common.nav` namespace. */
  labelKey: string;
  icon: typeof Ticket;
  soon?: boolean;
  /** Team-only (admin/engineer/...). Hidden from customer-role users. */
  team?: boolean;
  /** Admin-only. Shown only to the admin role. */
  admin?: boolean;
}

const NAV: NavItem[] = [
  { to: "/dashboard", labelKey: "nav.dashboard", icon: LayoutDashboard },
  { to: "/tickets", labelKey: "nav.tickets", icon: Ticket },
  { to: "/knowledge", labelKey: "nav.knowledge", icon: BookOpen },
  { to: "/customers", labelKey: "nav.customers", icon: Building2, team: true },
  { to: "/users", labelKey: "nav.users", icon: Users, team: true },
  { to: "/products", labelKey: "nav.products", icon: Package, team: true },
  { to: "/services", labelKey: "nav.services", icon: Layers, team: true },
  { to: "/subscriptions", labelKey: "nav.subscriptions", icon: CreditCard, team: true },
  { to: "/sla", labelKey: "nav.sla", icon: Timer, team: true },
  { to: "/data", labelKey: "nav.data", icon: Database, team: true },
  { to: "/rbac", labelKey: "nav.access", icon: ShieldCheck, team: true },
  { to: "/llm", labelKey: "nav.ai_providers", icon: Sparkles, team: true },
  { to: "/macros", labelKey: "nav.macros", icon: MessageSquareText, team: true },
  { to: "/teams", labelKey: "nav.teams", icon: UsersRound, team: true },
  { to: "/automations", labelKey: "nav.automations", icon: Workflow, admin: true },
  { to: "/settings", labelKey: "nav.settings", icon: Settings, admin: true },
];

export function AppShell() {
  const { t } = useTranslation();
  const { user, logout } = useAuth();
  const branding = useBranding();
  const navigate = useNavigate();
  const navRef = useReveal<HTMLElement>();

  // Collapsible left rail — icon-only when collapsed; preference persisted.
  const [collapsed, setCollapsed] = useState<boolean>(() => {
    try {
      return localStorage.getItem("st.sidebar") === "collapsed";
    } catch {
      return false;
    }
  });
  function toggleSidebar() {
    setCollapsed((c) => {
      const next = !c;
      try {
        localStorage.setItem("st.sidebar", next ? "collapsed" : "expanded");
      } catch {
        /* ignore */
      }
      return next;
    });
  }

  // Customer-role users see only the customer portal surface (their own
  // company's tickets + the knowledge base); team users see everything.
  // Admin-only items (e.g. Settings) are gated to the admin role.
  const isTeam = user?.role !== "customer";
  const isAdmin = user?.role === "admin";
  const navItems = NAV.filter((item) => {
    if (item.admin) return isAdmin;
    return isTeam || !item.team;
  });

  const initials = (
    (user?.first_name?.[0] ?? user?.username?.[0] ?? user?.email?.[0] ?? "?") +
    (user?.last_name?.[0] ?? "")
  ).toUpperCase();

  return (
    <div
      className={cn(
        "grid min-h-screen transition-[grid-template-columns] duration-200",
        collapsed ? "grid-cols-[4.25rem_1fr]" : "grid-cols-[15.5rem_1fr]"
      )}
    >
      {/* Left rail */}
      <aside className="sticky top-0 flex h-screen flex-col border-r border-border bg-card/40 backdrop-blur">
        <div
          className={cn(
            "flex h-16 items-center gap-2.5",
            collapsed ? "justify-center px-0" : "px-5"
          )}
        >
          <div className="grid size-8 shrink-0 place-items-center overflow-hidden rounded-md bg-primary text-primary-foreground shadow-[0_0_20px_-4px_color-mix(in_srgb,var(--primary)_70%,transparent)]">
            {branding.has_logo ? (
              <img
                src={branding.logo_url}
                alt={branding.app_name}
                className="size-full object-contain"
              />
            ) : (
              <Ticket className="size-4.5" strokeWidth={2.5} />
            )}
          </div>
          {!collapsed && (
            <div className="leading-none">
              <div className="font-display text-[15px] font-bold tracking-tight">
                {branding.app_name}
              </div>
              <div className="font-mono text-[10px] uppercase tracking-[0.2em] text-muted-foreground">
                {branding.app_subtitle}
              </div>
            </div>
          )}
        </div>

        <nav
          ref={navRef}
          className={cn(
            "flex flex-1 flex-col gap-0.5 py-2",
            collapsed ? "px-2" : "px-3"
          )}
        >
          {navItems.map((item) =>
            item.soon ? (
              <span
                key={item.to}
                data-reveal
                className={cn(
                  "flex cursor-not-allowed items-center gap-3 rounded-md py-2 text-sm text-muted-foreground/45",
                  collapsed ? "justify-center px-0" : "px-3"
                )}
                title={
                  collapsed
                    ? t("nav.coming_soon_item", { label: t(item.labelKey) })
                    : t("nav.coming_soon")
                }
              >
                <item.icon className="size-4 shrink-0" />
                {!collapsed && (
                  <>
                    {t(item.labelKey)}
                    <span className="ml-auto font-mono text-[9px] uppercase tracking-wider text-muted-foreground/40">
                      {t("nav.soon")}
                    </span>
                  </>
                )}
              </span>
            ) : (
              <NavLink
                key={item.to}
                to={item.to}
                data-reveal
                title={collapsed ? t(item.labelKey) : undefined}
                className={({ isActive }) =>
                  cn(
                    "group relative flex items-center gap-3 rounded-md py-2 text-sm font-medium transition-colors",
                    collapsed ? "justify-center px-0" : "px-3",
                    isActive
                      ? "bg-primary/10 text-foreground"
                      : "text-muted-foreground hover:bg-accent hover:text-foreground"
                  )
                }
              >
                {({ isActive }) => (
                  <>
                    <span
                      className={cn(
                        "absolute left-0 top-1/2 h-5 w-0.5 -translate-y-1/2 rounded-full bg-primary transition-opacity",
                        isActive ? "opacity-100" : "opacity-0"
                      )}
                    />
                    <item.icon className="size-4 shrink-0" />
                    {!collapsed && t(item.labelKey)}
                  </>
                )}
              </NavLink>
            )
          )}
        </nav>

        <div
          className={cn(
            "flex items-center gap-2 px-3 pb-4 pt-2",
            collapsed ? "justify-center" : "justify-between"
          )}
        >
          <button
            type="button"
            onClick={toggleSidebar}
            aria-label={t(collapsed ? "topbar.expand_sidebar" : "topbar.collapse_sidebar")}
            title={t(collapsed ? "topbar.expand_sidebar" : "topbar.collapse_sidebar")}
            className="grid size-8 shrink-0 place-items-center rounded-md text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
          >
            {collapsed ? (
              <PanelLeftOpen className="size-4" />
            ) : (
              <PanelLeftClose className="size-4" />
            )}
          </button>
        </div>
      </aside>

      {/* Main column */}
      <div className="flex min-h-screen flex-col">
        <header className="sticky top-0 z-20 flex h-16 items-center justify-between border-b border-border bg-background/70 px-7 backdrop-blur">
          <div className="font-mono text-xs uppercase tracking-[0.25em] text-muted-foreground">
            {branding.workspace_name}
          </div>
          <div className="flex items-center gap-3">
          <NotificationBell />
          <LanguageToggle />
          <ThemeToggle />
          <DropdownMenu>
            <DropdownMenuTrigger className="flex items-center gap-2.5 rounded-full border border-border bg-card/60 py-1 pl-1 pr-3 outline-none transition-colors hover:bg-accent">
              <Avatar className="size-7">
                <AvatarFallback>{initials}</AvatarFallback>
              </Avatar>
              <div className="text-left leading-tight">
                <div className="text-xs font-medium">
                  {user?.first_name || user?.username}
                </div>
                <div className="font-mono text-[10px] uppercase tracking-wide text-muted-foreground">
                  {user?.role}
                </div>
              </div>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuLabel>{user?.email}</DropdownMenuLabel>
              <DropdownMenuSeparator />
              <DropdownMenuItem
                onSelect={() => {
                  logout();
                  navigate("/login");
                }}
              >
                <LogOut /> {t("topbar.sign_out")}
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
          </div>
        </header>

        <main className="flex-1 px-7 py-7">
          <Outlet />
        </main>
      </div>
    </div>
  );
}

/** Light / dark / system theme switcher. */
function ThemeToggle() {
  const { t } = useTranslation();
  const { theme, resolved, setTheme } = useTheme();
  const options: { value: Theme; labelKey: string; icon: typeof Sun }[] = [
    { value: "light", labelKey: "theme.light", icon: Sun },
    { value: "dark", labelKey: "theme.dark", icon: Moon },
    { value: "system", labelKey: "theme.system", icon: Monitor },
  ];
  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        aria-label={t("theme.toggle")}
        className="grid size-9 place-items-center rounded-full border border-border bg-card/60 text-muted-foreground outline-none transition-colors hover:bg-accent hover:text-foreground"
      >
        {resolved === "dark" ? (
          <Moon className="size-4" />
        ) : (
          <Sun className="size-4" />
        )}
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuLabel>{t("theme.label")}</DropdownMenuLabel>
        <DropdownMenuSeparator />
        {options.map((o) => (
          <DropdownMenuItem
            key={o.value}
            onSelect={() => setTheme(o.value)}
            className={theme === o.value ? "text-primary" : undefined}
          >
            <o.icon /> {t(o.labelKey)}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
