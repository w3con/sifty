import { defineConfig } from 'vite';
export default defineConfig({ root: 'web', build: { outDir: process.env.BUILD_OUTDIR || '../dist', emptyOutDir: true } });
