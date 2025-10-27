import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import ViteRuby from 'vite-plugin-ruby'

export default defineConfig({
  plugins: [
    react(),
    ViteRuby()
  ],
  server: {
    port: 3036
  }
})
