import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import path from "node:path";

// https://vite.dev/config/
export default defineConfig({
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
