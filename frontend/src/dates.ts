export function parseDate(value: string): Date {
  // SQLite CURRENT_TIMESTAMP is UTC but may arrive without an explicit timezone.
  const normalized = /^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}(\.\d+)?$/.test(value)
    ? `${value.replace(' ', 'T')}Z`
    : value
  return new Date(normalized)
}

export function formatDate(value: string): string {
  const date = parseDate(value)
  if (!Number.isFinite(date.getTime())) return '—'
  return date.toLocaleString(undefined, {
    year: 'numeric', month: 'short', day: 'numeric',
    hour: '2-digit', minute: '2-digit', second: '2-digit', timeZoneName: 'short',
  })
}

export function formatRelativeDate(value: string, now = Date.now(), locale?: string): string {
  const difference = parseDate(value).getTime() - now
  if (!Number.isFinite(difference)) return '—'
  const seconds = Math.abs(difference) / 1000
  if (seconds < 60) {
    if (difference <= 0) return 'just now'
    return new Intl.RelativeTimeFormat(locale, { numeric: 'always' }).format(Math.max(1, Math.floor(seconds)), 'second')
  }
  const units: [Intl.RelativeTimeFormatUnit, number][] = [
    ['year', 365 * 24 * 60 * 60],
    ['month', 30 * 24 * 60 * 60],
    ['week', 7 * 24 * 60 * 60],
    ['day', 24 * 60 * 60],
    ['hour', 60 * 60],
    ['minute', 60],
  ]
  const [unit, duration] = units.find(([, duration]) => seconds >= duration)!
  const amount = Math.floor(seconds / duration) * Math.sign(difference)
  return new Intl.RelativeTimeFormat(locale, { numeric: 'always' }).format(amount, unit)
}

export function formatVersionTooltip(createdAt: string, deployedAt: string, gitURL: string): string {
  const created = parseDate(createdAt).getTime()
  if (!Number.isFinite(created)) return gitURL
  const lines = [`Created ${formatDate(createdAt)}`]
  const deployed = parseDate(deployedAt).getTime()
  if (Number.isFinite(deployed)) {
    lines.push(`Deployed ${formatDate(deployedAt)}`)
    const seconds = Math.abs(deployed - created) / 1000
    const units: [string, number][] = [['day', 86400], ['hour', 3600], ['minute', 60], ['second', 1]]
    const unit = units.find(([, duration]) => seconds >= duration)
    if (!unit) {
      lines.push('Deployed at creation time')
    } else {
      const [name, duration] = unit
      const age = new Intl.NumberFormat(undefined, { style: 'unit', unit: name, unitDisplay: 'long' })
        .format(Math.floor(seconds / duration))
      lines.push(deployed >= created ? `Deployed ${age} after creation` : `Deployment predates creation by ${age}`)
    }
  }
  lines.push(gitURL)
  return lines.join('\n')
}
