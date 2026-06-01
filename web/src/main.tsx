import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { BrowserRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

// Fonts
import "@fontsource-variable/bricolage-grotesque/index.css";
import "@fontsource/ibm-plex-sans/400.css";
import "@fontsource/ibm-plex-sans/500.css";
import "@fontsource/ibm-plex-sans/600.css";
import "@fontsource/ibm-plex-mono/400.css";
import "@fontsource/ibm-plex-mono/500.css";

import "./index.css";
import App from "./App.tsx";
import { AuthProvider } from "@/lib/auth";
import { ThemeProvider } from "@/lib/theme";
import { BrandingProvider } from "@/lib/branding";
import { Toaster } from "@/components/ui/toaster";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { staleTime: 15_000, retry: 1, refetchOnWindowFocus: false },
  },
});

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <ThemeProvider>
      <QueryClientProvider client={queryClient}>
        <BrandingProvider>
          <BrowserRouter basename={import.meta.env.BASE_URL.replace(/\/$/, "") || undefined}>
            <AuthProvider>
              <App />
              <Toaster />
            </AuthProvider>
          </BrowserRouter>
        </BrandingProvider>
      </QueryClientProvider>
    </ThemeProvider>
  </StrictMode>
);
