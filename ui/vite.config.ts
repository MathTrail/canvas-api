import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import federation from '@originjs/vite-plugin-federation'

export default defineConfig({
  server: { port: 3001 },
  // Buffer polyfill: required by some protobuf/centrifuge-js builds — do not remove
  define: { global: 'window' },
  plugins: [
    react(),
    tailwindcss(),
    federation({
      name: 'canvas',
      filename: 'remoteEntry.js',
      exposes: {
        './OlympiadCanvas': './src/components/OlympiadCanvas',
      },
      shared: ['react', 'react-dom', 'zustand'],
    }),
  ],
  build: {
    target: 'esnext',
    minify: false,
  },
})
