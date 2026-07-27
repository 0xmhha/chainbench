import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'

// The SPA is served by the Go dashboard server at the site root and its built
// assets are committed to pkg/dashboard/spa for go:embed. base '/' keeps asset
// URLs rooted so the FileServer serves them at /assets/.
export default defineConfig({
  plugins: [svelte()],
  base: '/',
  build: { outDir: '../pkg/dashboard/spa', emptyOutDir: true },
})
