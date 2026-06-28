import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// Build output goes into the Go embed directory so the binary can serve the SPA.
// base: '/' produces absolute asset URLs (/assets/...). The SPA is served at the
// origin root with index.html as a client-side-routing fallback, so absolute
// paths resolve correctly even on deep links like /projects/x/traces (relative
// './assets' would wrongly resolve under the nested route on refresh).
export default defineConfig({
  plugins: [react()],
  base: '/',
  build: {
    outDir: '../internal/webui/dist',
    emptyOutDir: true,
  },
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/v1': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
