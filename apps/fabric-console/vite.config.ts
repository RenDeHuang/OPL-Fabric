import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import type { ClientRequest } from "node:http";

interface ProxyWithRequestHook {
  on(event: "proxyReq", callback: (proxyReq: ClientRequest) => void): void;
}

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    strictPort: false,
    proxy: {
      "/api": {
        target: "http://127.0.0.1:8787",
        changeOrigin: true,
        configure(proxy) {
          (proxy as ProxyWithRequestHook).on("proxyReq", (proxyReq) => {
            const token = process.env.OPL_OPERATOR_TOKEN;
            if (token) {
              proxyReq.setHeader("Authorization", `Bearer ${token}`);
            }
          });
        }
      }
    }
  }
});
