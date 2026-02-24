export interface ProjectSummary {
  project: string
  latest_status: string
  total_runs: number
  passed_runs: number
  failed_runs: number
  last_run: string
}

export interface TestResult {
  id: number
  project: string
  spec: string
  browser: string
  status: string
  pipeline_id: string
  job_id: string
  job_url: string
  created_at: string
}

const normalizePathPrefix = (value: string | undefined): string => {
  const trimmed = (value ?? '').trim()
  if (trimmed === '' || trimmed === '/') return ''
  const withSlash = trimmed.startsWith('/') ? trimmed : `/${trimmed}`
  return withSlash.replace(/\/$/, '')
}

const fromWindow = typeof window !== 'undefined'
  ? (window as Window & { __PATH_PREFIX__?: string }).__PATH_PREFIX__
  : undefined

const pathPrefix = normalizePathPrefix(fromWindow ?? import.meta.env.VITE_PATH_PREFIX)
export { pathPrefix }

export function statusClass(status: string): string {
  if (status === 'success') return 'success'
  if (status === 'failed') return 'failed'
  return 'other'
}

export function formatDate(dt: string): string {
  if (!dt) return '—'
  return new Date(dt).toLocaleString(undefined, {
    year: 'numeric', month: 'short', day: 'numeric',
    hour: '2-digit', minute: '2-digit',
  })
}

export async function fetchJSON<T>(url: string): Promise<T> {
  const normalizedUrl = url.startsWith('/') && pathPrefix ? `${pathPrefix}${url}` : url
  const res = await fetch(normalizedUrl)
  if (!res.ok) throw new Error(`HTTP ${res.status}`)
  return res.json() as Promise<T>
}
