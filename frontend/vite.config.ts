import react from "@vitejs/plugin-react";
import { defineConfig, loadEnv } from "vite";

declare const process: {
  env: Record<string, string | undefined>;
};

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, ".", "");
  const apiUrl = process.env.TXSIM_API_URL ?? "http://127.0.0.1:8080";

  return {
    plugins: [
      react(),
      {
        name: "txsim-runtime-config",
        configureServer(server) {
          server.middlewares.use("/config.js", (_request, response) => {
            response.setHeader("Content-Type", "application/javascript");
            response.end(`window.__TXSIM_CONFIG__ = ${JSON.stringify({ apiUrl })};\n`);
          });
        }
      }
    ],
    server: {
      port: parsePort(env.TXSIM_FRONTEND_PORT, 5173),
      strictPort: false
    }
  };
});

function parsePort(value: string | undefined, fallback: number): number {
  const parsed = Number(value);
  if (!Number.isInteger(parsed) || parsed < 1 || parsed > 65535) {
    return fallback;
  }
  return parsed;
}
