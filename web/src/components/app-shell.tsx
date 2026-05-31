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
  LogOut,
  Sun,
  Moon,
  Monitor,
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

interface NavItem {
  to: string;
  label: string;
  icon: typeof Ticket;
  soon?: boolean;
  /** Team-only (admin/engineer/...). Hidden from customer-role users. */
  team?: boolean;
  /** Admin-only. Shown only to the admin role. */
  admin?: boolean;
}

const NAV: NavItem[] = [
  { to: "/dashboard", label: "Dashboard", icon: LayoutDashboard },
  { to: "/tickets", label: "Tickets", icon: Ticket },
  { to: "/knowledge", label: "Knowledge", icon: BookOpen },
  { to: "/customers", label: "Customers", icon: Building2, team: true },
  { to: "/users", label: "Users", icon: Users, team: true },
  { to: "/products", label: "Products", icon: Package, team: true },
  { to: "/services", label: "Services", icon: Layers, team: true },
  { to: "/subscriptions", label: "Subscriptions", icon: CreditCard, team: true },
  { to: "/sla", label: "SLA", icon: Timer, team: true },
  { to: "/data", label: "Data", icon: Database, team: true },
  { to: "/rbac", label: "Access", icon: ShieldCheck, team: true },
  { to: "/llm", label: "AI Providers", icon: Sparkles, team: true },
  { to: "/settings", label: "Settings", icon: Settings, admin: true },
];

export function AppShell() {
  const { user, logout } = useAuth();
  const branding = useBranding();
  const navigate = useNavigate();
  const navRef = useReveal<HTMLElement>();

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
    <div className="grid min-h-screen grid-cols-[15.5rem_1fr]">
      {/* Left rail */}
      <aside className="sticky top-0 flex h-screen flex-col border-r border-border bg-card/40 backdrop-blur">
        <div className="flex h-16 items-center gap-2.5 px-5">
          <div className="grid size-8 place-items-center overflow-hidden rounded-md bg-primary text-primary-foreground shadow-[0_0_20px_-4px_color-mix(in_srgb,var(--primary)_70%,transparent)]">
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
          <div className="leading-none">
            <div className="font-display text-[15px] font-bold tracking-tight">
              {branding.app_name}
            </div>
            <div className="font-mono text-[10px] uppercase tracking-[0.2em] text-muted-foreground">
              {branding.app_subtitle}
            </div>
          </div>
        </div>

        <nav ref={navRef} className="flex flex-1 flex-col gap-0.5 px-3 py-2">
          {navItems.map((item) =>
            item.soon ? (
              <span
                key={item.to}
                data-reveal
                className="flex cursor-not-allowed items-center gap-3 rounded-md px-3 py-2 text-sm text-muted-foreground/45"
                title="Coming soon"
              >
                <item.icon className="size-4" />
                {item.label}
                <span className="ml-auto font-mono text-[9px] uppercase tracking-wider text-muted-foreground/40">
                  soon
                </span>
              </span>
            ) : (
              <NavLink
                key={item.to}
                to={item.to}
                data-reveal
                className={({ isActive }) =>
                  cn(
                    "group relative flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors",
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
                    <item.icon className="size-4" />
                    {item.label}
                  </>
                )}
              </NavLink>
            )
          )}
        </nav>

        <div className="px-3 pb-4 font-mono text-[10px] text-muted-foreground/50">
          v0.1 · single-tenant
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
                <LogOut /> Sign out
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
  const { theme, resolved, setTheme } = useTheme();
  const options: { value: Theme; label: string; icon: typeof Sun }[] = [
    { value: "light", label: "Light", icon: Sun },
    { value: "dark", label: "Dark", icon: Moon },
    { value: "system", label: "System", icon: Monitor },
  ];
  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        aria-label="Toggle theme"
        className="grid size-9 place-items-center rounded-full border border-border bg-card/60 text-muted-foreground outline-none transition-colors hover:bg-accent hover:text-foreground"
      >
        {resolved === "dark" ? (
          <Moon className="size-4" />
        ) : (
          <Sun className="size-4" />
        )}
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuLabel>Theme</DropdownMenuLabel>
        <DropdownMenuSeparator />
        {options.map((o) => (
          <DropdownMenuItem
            key={o.value}
            onSelect={() => setTheme(o.value)}
            className={theme === o.value ? "text-primary" : undefined}
          >
            <o.icon /> {o.label}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
