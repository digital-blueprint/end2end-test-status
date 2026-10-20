import assert from 'node:assert/strict'
import test from 'node:test'
import { buildDeploymentCompareURL, hasSameDeploymentRevision } from '../src/deploymentCompare.ts'

test('GitHub compares the base deployed commit to the target deployed commit', () => {
  assert.equal(buildDeploymentCompareURL(
    'https://github.com/team/app/commit/abc123',
    'https://github.com/team/app/commit/def456',
  ), 'https://github.com/team/app/compare/abc123...def456')
})

test('GitLab supports nested groups and self-hosted instances', () => {
  assert.equal(buildDeploymentCompareURL(
    'https://gitlab.example.com/team/subgroup/app/-/commit/abc123',
    'https://gitlab.example.com/team/subgroup/app/-/commit/def456',
  ), 'https://gitlab.example.com/team/subgroup/app/-/compare/abc123...def456')
})

test('GitHub Enterprise commit links work without a hardcoded hostname', () => {
  assert.equal(buildDeploymentCompareURL(
    'https://git.example.com/team/app/commit/abc123',
    'https://git.example.com/team/app/commit/def456',
  ), 'https://git.example.com/team/app/compare/abc123...def456')
})

test('query strings, fragments, and trailing slashes are not included in comparisons', () => {
  assert.equal(buildDeploymentCompareURL(
    'https://github.com/team/app/commit/abc123/?diff=split#files',
    'https://github.com/team/app/commit/def456?foo=bar',
  ), 'https://github.com/team/app/compare/abc123...def456')
})

test('identical commits cannot be compared', () => {
  const url = 'https://github.com/team/app/commit/abc123'
  assert.equal(buildDeploymentCompareURL(url, url), null)
  assert.equal(hasSameDeploymentRevision(url, url), true)
})

test('identical commits are recognized despite hash casing and URL decorations', () => {
  for (const repository of ['https://github.com/team/app/commit/', 'https://gitlab.example.com/team/app/-/commit/']) {
    const base = `${repository}abc123`
    const target = `${repository}ABC123/?diff=split#files`
    assert.equal(hasSameDeploymentRevision(base, target), true)
    assert.equal(buildDeploymentCompareURL(base, target), null)
  }
})

test('different commits, repositories, and unsupported URLs are not treated as identical', () => {
  const base = 'https://github.com/team/app/commit/abc123'
  for (const target of [
    'https://github.com/team/app/commit/def456',
    'https://github.com/team/other/commit/abc123',
    'https://gitlab.example.com/team/app/-/commit/abc123',
    '', 'not a URL',
  ]) assert.equal(hasSameDeploymentRevision(base, target), false)
})

test('real dashboard GitHub and TU Graz GitLab URLs retain full deployed hashes', () => {
  for (const [repository, commitPath, comparePath, hash] of [
    ['https://github.com/digital-blueprint/esign-app', '/commit/', '/compare/', '78f1617142354a23f5efcd3cf21496884f3827b6'],
    ['https://gitlab.tugraz.at/dbp/middleware/api', '/-/commit/', '/-/compare/', 'f9c4489925e07611c73e10006d7dad6bb0744b32'],
  ]) {
    const url = `${repository}${commitPath}${hash}`
    assert.equal(buildDeploymentCompareURL(url, url), null)
    assert.equal(buildDeploymentCompareURL(url, `${repository}${commitPath}def456`), `${repository}${comparePath}${hash}...def456`)
  }
})

test('different repositories, hosts, protocols, or providers cannot be compared', () => {
  const base = 'https://github.com/team/app/commit/abc123'
  for (const target of [
    'https://github.com/team/other/commit/def456',
    'https://git.example.com/team/app/commit/def456',
    'http://github.com/team/app/commit/def456',
    'https://github.com/team/app/-/commit/def456',
  ]) assert.equal(buildDeploymentCompareURL(base, target), null)
})

test('unsupported or unsafe URLs do not generate compare links', () => {
  const valid = 'https://github.com/team/app/commit/abc123'
  for (const url of [
    '', 'not a URL', 'javascript:alert(1)',
    'https://github.com/team/app',
    'https://github.com/team/app/tree/main',
    'https://gitlab.com/team/app/-/pipelines/123',
    'https://user:secret@github.com/team/app/commit/abc123',
    'ftp://github.com/team/app/commit/abc123',
    'https://github.com/team/app/commit/not-a-hash',
    'https://github.com/team/app/commit/abc123/extra',
    'https://github.com/team/app/commit/abc123%2Fdef456',
  ]) {
    assert.equal(buildDeploymentCompareURL(url, valid), null, url)
    assert.equal(buildDeploymentCompareURL(valid, url), null, url)
  }
})
