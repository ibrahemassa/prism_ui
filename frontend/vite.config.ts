import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    port: 5180,
    proxy: {
      // dev: forward API calls to the Go backend (read-only)
      "/api": "http://localhost:8080",
    },
  },
});
