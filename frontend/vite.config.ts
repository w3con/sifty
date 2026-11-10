import react from '@vitejs/plugin-react';
import { defineConfig } from 'vite';

// Stable internal staging API address. The preview's private backend port may
// change, so it is never referenced here.
const apiTarget = 'http://127.0.0.1:8082';

export default defineConfig({
  plugins: [react()],
  build: {
    outDir: process.env.BUILD_OUTDIR || 'dist',
    emptyOutDir: true,
  },
  server: {
    host: '127.0.0.1',
    port: 8081,
    strictPort: true,
    proxy: {
      // Regex: only /api/ requests. A plain '/api' key also captures /api.ts
      // and proxies the application's own module to the backend as HTML.
      '^/api/': {
        target: apiTarget,
        changeOrigin: true,
        // The Go server validates Host/Origin against its own public address.
        headers: {
          Host: new URL(apiTarget).host,
          Origin: apiTarget,
        },
      },
    },
  },
});
