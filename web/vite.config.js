import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'

// The SPA is served by the Go dashboard server under /app/ and its built assets
// are committed to pkg/dashboard/spa for go:embed. base must match the serve path.
export default defineConfig({
  plugins: [svelte()],
  base: '/app/',
  build: { outDir: '../pkg/dashboard/spa', emptyOutDir: true },
})
