import { createRouter, createWebHistory } from 'vue-router'
import Dashboard from '../views/Dashboard.vue'
import ProjectView from '../views/ProjectView.vue'
import AllResults from '../views/AllResults.vue'

const normalizePathPrefix = (value: string | undefined) => {
  const trimmed = (value ?? '').trim()
  if (trimmed === '' || trimmed === '/') return ''
  const withSlash = trimmed.startsWith('/') ? trimmed : `/${trimmed}`
  return withSlash.replace(/\/$/, '')
}

const fromWindow = typeof window !== 'undefined'
  ? (window as Window & { __PATH_PREFIX__?: string }).__PATH_PREFIX__
  : undefined

const base = normalizePathPrefix(fromWindow ?? import.meta.env.VITE_PATH_PREFIX)

export default createRouter({
  history: createWebHistory(base || undefined),
  routes: [
    { path: '/', component: Dashboard },
    { path: '/project/:name', component: ProjectView },
    { path: '/results', component: AllResults },
  ],
})
