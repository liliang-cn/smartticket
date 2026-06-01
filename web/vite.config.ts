import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import path from "node:path";

// https://vite.dev/config/
export default defineConfig({
  // Base path the app is served under. Default "/" (single-binary / Docker
  // serve the SPA at the root). The live site builds with VITE_BASE=/app/ so the
  // marketing landing can own "/" and the console lives under "/app".
  base: process.env.VITE_BASE || "/",
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: { "@": path.resolve(__dirname, "./src") },
  },
  server: {
    port: 5410,
    proxy: {
      // Proxy API calls to the SmartTicket backend during development.
      "/api": {
        target: "http://localhost:6533",
        changeOrigin: true,
      },
    },
  },
});
