import { defineConfig } from 'vite';
export default defineConfig({ root: '../../frontend', build: { outDir: process.env.BUILD_OUTDIR || 'dist', emptyOutDir: true } });
