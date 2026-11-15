import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  base: '/Supernova/',
  build: {
    outDir: '../docs',
    // Preserve hand-written documentation beside the published website.
    emptyOutDir: false
  }
})
