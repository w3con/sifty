import { defineConfig } from 'vite';
export default defineConfig({ build: { outDir: process.env.BUILD_OUTDIR || '../dist', emptyOutDir: true } });
