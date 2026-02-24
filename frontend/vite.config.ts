import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'

const normalizePathPrefix = (value: string) => {
  const trimmed = value.trim()
  if (trimmed === '' || trimmed === '/') return ''
  const withSlash = trimmed.startsWith('/') ? trimmed : `/${trimmed}`
  return withSlash.replace(/\/$/, '')
}

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const rawPrefix = env.PATH_PREFIX ?? env.VITE_PATH_PREFIX ?? ''
  const pathPrefix = normalizePathPrefix(rawPrefix)
  const base = pathPrefix === '' ? '/' : `${pathPrefix}/`

  return {
    plugins: [vue()],
    base,
    define: {
      'import.meta.env.VITE_PATH_PREFIX': JSON.stringify(pathPrefix),
    },
    build: {
      outDir: 'dist',
      emptyOutDir: true,
    },
    server: {
      proxy: {
        '/api': 'http://localhost:8080',
        '/webhook': 'http://localhost:8080',
      },
    },
  }
})
