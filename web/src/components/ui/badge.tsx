import { cva, type VariantProps } from "class-variance-authority";
import * as React from "react";
import { cn } from "@/lib/utils";

const badgeVariants = cva(
  "inline-flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-xs font-medium font-mono tracking-tight transition-colors",
  {
    variants: {
      tone: {
        neutral: "border-border bg-muted text-muted-foreground",
        amber: "border-primary/30 bg-primary/10 text-primary",
        green: "border-emerald-500/30 bg-emerald-500/10 text-emerald-300",
        blue: "border-sky-500/30 bg-sky-500/10 text-sky-300",
        red: "border-red-500/30 bg-red-500/10 text-red-300",
        slate: "border-slate-500/30 bg-slate-500/10 text-slate-300",
      },
    },
    defaultVariants: { tone: "neutral" },
  }
);

export interface BadgeProps
  extends React.HTMLAttributes<HTMLSpanElement>,
    VariantProps<typeof badgeVariants> {}

export function Badge({ className, tone, ...props }: BadgeProps) {
  return <span className={cn(badgeVariants({ tone, className }))} {...props} />;
}

export { badgeVariants };
