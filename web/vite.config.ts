import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// The build lands in dist/, which the Go binary embeds (see web/embed.go).
// In dev, /api is proxied to a running `bazel run //cmd/viewer`.
export default defineConfig({
  plugins: [react()],
  build: { outDir: "dist", emptyOutDir: true },
  server: { proxy: { "/api": "http://127.0.0.1:8090" } },
});
