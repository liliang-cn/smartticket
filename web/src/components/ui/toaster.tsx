import { Toaster as Sonner } from "sonner";

export function Toaster() {
  return (
    <Sonner
      theme="dark"
      position="bottom-right"
      toastOptions={{
        style: {
          background: "var(--color-popover)",
          border: "1px solid var(--color-border)",
          color: "var(--color-foreground)",
          fontFamily: "var(--font-sans)",
        },
      }}
    />
  );
}
