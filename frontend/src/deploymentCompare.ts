interface GitRevision {
  repository: string
  provider: 'github' | 'gitlab'
  revision: string
}

// Infer the provider and revision from commit links, including self-hosted instances.
// Do not guess revisions from freeform version labels or repository-only URLs.
function parseGitRevision(value: string): GitRevision | null {
  try {
    const url = new URL(value)
    if (!['https:', 'http:'].includes(url.protocol) || url.username || url.password) return null

    const gitlab = url.pathname.match(/^\/(.+)\/-\/commit\/([a-f0-9]+)\/?$/i)
    const github = url.pathname.match(/^\/([^/]+\/[^/]+)\/commit\/([a-f0-9]+)\/?$/i)
    const match = gitlab ?? github
    if (!match) return null

    return {
      repository: `${url.origin}/${match[1]}`,
      provider: gitlab ? 'gitlab' : 'github',
      revision: match[2].toLowerCase(),
    }
  } catch {
    return null
  }
}

export function hasSameDeploymentRevision(baseURL: string, targetURL: string): boolean {
  const base = parseGitRevision(baseURL)
  const target = parseGitRevision(targetURL)
  return !!base && !!target && base.provider === target.provider
    && base.repository === target.repository && base.revision === target.revision
}

export function buildDeploymentCompareURL(baseURL: string, targetURL: string): string | null {
  const base = parseGitRevision(baseURL)
  const target = parseGitRevision(targetURL)
  if (!base || !target || base.provider !== target.provider || base.repository !== target.repository) return null
  if (base.revision === target.revision) return null

  const comparePath = base.provider === 'gitlab' ? '/-/compare/' : '/compare/'
  return `${base.repository}${comparePath}${base.revision}...${target.revision}`
}
