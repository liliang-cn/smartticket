import { useState, useRef, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import gsap from "gsap";
import { Ticket, ArrowRight } from "lucide-react";
import { useAuth } from "@/lib/auth";
import { apiError } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

export function LoginPage() {
  const { login } = useAuth();
  const navigate = useNavigate();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const rootRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const ctx = gsap.context(() => {
      gsap
        .timeline({ defaults: { ease: "power3.out" } })
        .from(".login-brand", { y: 20, opacity: 0, duration: 0.6 })
        .from(
          ".login-field",
          { y: 16, opacity: 0, duration: 0.5, stagger: 0.08 },
          "-=0.3"
        )
        .from(".login-aside", { opacity: 0, duration: 0.8 }, "-=0.6");
    }, rootRef);
    return () => ctx.revert();
  }, []);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setBusy(true);
    try {
      await login(email.trim(), password);
      navigate("/dashboard");
    } catch (err) {
      setError(apiError(err, "Invalid email or password"));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div ref={rootRef} className="grid min-h-screen lg:grid-cols-2">
      {/* Form side */}
      <div className="flex items-center justify-center px-6 py-12">
        <div className="w-full max-w-sm">
          <div className="login-brand mb-10 flex items-center gap-3">
            <div className="grid size-10 place-items-center rounded-lg bg-primary text-primary-foreground shadow-[0_0_30px_-6px_rgba(255,176,31,0.9)]">
              <Ticket className="size-5" strokeWidth={2.5} />
            </div>
            <div>
              <div className="font-display text-xl font-bold tracking-tight">
                SmartTicket
              </div>
              <div className="font-mono text-[10px] uppercase tracking-[0.25em] text-muted-foreground">
                operations console
              </div>
            </div>
          </div>

          <h1 className="login-field text-2xl font-bold">Sign in</h1>
          <p className="login-field mt-1 text-sm text-muted-foreground">
            Access the support workspace.
          </p>

          <form onSubmit={onSubmit} className="mt-8 space-y-5">
            <div className="login-field space-y-1.5">
              <Label htmlFor="email">Email</Label>
              <Input
                id="email"
                type="email"
                autoComplete="username"
                placeholder="you@company.com"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                required
              />
            </div>
            <div className="login-field space-y-1.5">
              <Label htmlFor="password">Password</Label>
              <Input
                id="password"
                type="password"
                autoComplete="current-password"
                placeholder="••••••••"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
              />
            </div>
            {error && (
              <p className="login-field rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive-foreground">
                {error}
              </p>
            )}
            <Button
              type="submit"
              size="lg"
              className="login-field w-full"
              disabled={busy}
            >
              {busy ? "Signing in…" : "Sign in"}
              {!busy && <ArrowRight />}
            </Button>
          </form>
        </div>
      </div>

      {/* Aesthetic side */}
      <div className="login-aside relative hidden overflow-hidden border-l border-border lg:block">
        <div className="grid-texture absolute inset-0 opacity-60" />
        <div className="absolute inset-0 bg-[radial-gradient(40rem_30rem_at_70%_30%,rgba(255,176,31,0.16),transparent_70%)]" />
        <div className="relative flex h-full flex-col justify-end p-12">
          <div className="font-mono text-xs uppercase tracking-[0.3em] text-primary/80">
            // mission control
          </div>
          <p className="mt-4 max-w-md font-display text-3xl font-bold leading-tight tracking-tight">
            Every ticket, SLA and customer — under one calm, fast surface.
          </p>
          <p className="mt-4 max-w-sm text-sm text-muted-foreground">
            Self-hosted. Single-tenant. Your data, your rules.
          </p>
        </div>
      </div>
    </div>
  );
}
