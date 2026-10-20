import assert from 'node:assert/strict'
import test from 'node:test'
import { formatDate, formatRelativeDate, formatVersionTooltip, parseDate } from '../src/dates.ts'

const now = Date.parse('2026-10-20T12:00:00Z')
const ago = (seconds: number) => new Date(now - seconds * 1000).toISOString()

test('relative dates cover seconds through years', () => {
  for (const [seconds, expected] of [
    [0, 'just now'], [59, 'just now'], [60, '1 minute ago'], [120, '2 minutes ago'],
    [3599, '59 minutes ago'], [3600, '1 hour ago'], [7200, '2 hours ago'],
    [86400, '1 day ago'], [7 * 86400, '1 week ago'],
    [30 * 86400, '1 month ago'], [183 * 86400, '6 months ago'], [365 * 86400, '1 year ago'],
  ] as const) {
    assert.equal(formatRelativeDate(ago(seconds), now, 'en'), expected)
  }
})

test('future timestamps are not described as past deployments', () => {
  assert.equal(formatRelativeDate(ago(-30), now, 'en'), 'in 30 seconds')
  assert.equal(formatRelativeDate(ago(-120), now, 'en'), 'in 2 minutes')
  assert.equal(formatRelativeDate(ago(-86400), now, 'en'), 'in 1 day')
})

test('timezone offsets and SQLite timestamps represent the same UTC instant', () => {
  const previous = process.env.TZ
  process.env.TZ = 'America/New_York'
  try {
    assert.equal(parseDate('2026-10-20 10:00:00').toISOString(), '2026-10-20T10:00:00.000Z')
    assert.equal(formatRelativeDate('2026-10-20 10:00:00', now, 'en'), '2 hours ago')
    assert.equal(formatRelativeDate('2026-10-20T12:00:00+02:00', now, 'en'), '2 hours ago')
  } finally {
    if (previous === undefined) delete process.env.TZ
    else process.env.TZ = previous
  }
})

test('missing and invalid timestamps render a placeholder', () => {
  for (const value of ['', 'not a timestamp']) {
    assert.equal(formatRelativeDate(value, now, 'en'), '—')
    assert.equal(formatDate(value), '—')
  }
})

test('relative labels change when the clock advances', () => {
  assert.equal(formatRelativeDate(ago(30), now, 'en'), 'just now')
  assert.equal(formatRelativeDate(ago(30), now + 120_000, 'en'), '2 minutes ago')
})

test('exact date retains seconds and timezone information', () => {
  const formatted = formatDate('2026-10-20T12:34:56Z')
  assert.ok(formatted.includes('2026'))
  assert.ok(formatted.includes('56'))
  assert.notEqual(formatted, 'Invalid Date')
})

test('version tooltip shows creation time and age at deployment, not current age', () => {
  const tooltip = formatVersionTooltip('2026-10-13T12:00:00Z', '2026-10-20T12:00:00Z', 'https://git.example/commit')
  assert.ok(tooltip.includes(`Created ${formatDate('2026-10-13T12:00:00Z')}`))
  assert.ok(tooltip.includes(`Deployed ${formatDate('2026-10-20T12:00:00Z')}`))
  assert.ok(tooltip.includes('Deployed 7 days after creation'))
  assert.ok(tooltip.endsWith('https://git.example/commit'))
})

test('version tooltip handles short delays, missing dates, and reversed timestamps', () => {
  const created = '2026-10-20T12:00:00Z'
  assert.ok(formatVersionTooltip(created, created, 'url').includes('Deployed at creation time'))
  assert.ok(formatVersionTooltip(created, '2026-10-20T14:00:00Z', 'url').includes('Deployed 2 hours after creation'))
  assert.ok(formatVersionTooltip(created, '2026-10-20T12:01:00Z', 'url').includes('Deployed 1 minute after creation'))
  assert.ok(formatVersionTooltip(created, '2026-10-20T12:00:30Z', 'url').includes('Deployed 30 seconds after creation'))
  assert.ok(formatVersionTooltip(created, '2026-10-19T12:00:00Z', 'url').includes('Deployment predates creation by 1 day'))
  for (const missing of ['', 'invalid']) {
    assert.equal(formatVersionTooltip(missing, created, 'url'), 'url')
    assert.equal(formatVersionTooltip(created, missing, 'url'), `Created ${formatDate(created)}\nurl`)
  }
})
