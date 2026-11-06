import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';
import tailwindcss from '@tailwindcss/vite';

export default defineConfig({
  plugins: [tailwindcss(), svelte()],
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    target: 'es2022',
    minify: 'esbuild',
  },
  server: {
    port: 5173,
    proxy: {
      '/api': 'http://localhost:8090',
      '/uploads': 'http://localhost:8090',
    },
  },
});
