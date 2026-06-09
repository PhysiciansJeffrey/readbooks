import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import wails from "@wailsio/runtime/plugins/vite";
import path from "path";
import fs from 'fs';

// 调试阶段需要server来激活hmr
let proxyTarget = '';
try {
  const state = JSON.parse(fs.readFileSync(path.resolve(__dirname, '../bin/state.json'), 'utf-8'));
  proxyTarget = state.local_http || '';
} catch {}

const server = {};
if (proxyTarget) {
  server.proxy = {
    '/api': {
      target: proxyTarget,
      changeOrigin: true,
    }
  };
}

export default defineConfig({
  plugins: [vue(), wails("./bindings")],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "src"),
    },
  },
  server,
});




