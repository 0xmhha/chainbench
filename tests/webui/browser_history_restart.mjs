import { chromium } from 'playwright'
import fs from 'node:fs'
import assert from 'node:assert/strict'

// Development regression against the same native fixture after daemon restart.
// It does not substitute for the complete WEB-09 acceptance scenarios.
const [fixturePath, output] = process.argv.slice(2), f = JSON.parse(fs.readFileSync(fixturePath, 'utf8'))
const browser = await chromium.launch({headless: true, channel: process.env.WEBUI_BROWSER_CHANNEL ?? 'chrome'})
try {
  const context = await browser.newContext(), api = context.request
  const login = await api.post(f.url + '/api/v1/auth/login', {data: {username: 'native-admin', password: f.password}})
  assert.equal(login.status(), 200)
  const history = await (await api.get(f.url + '/api/v1/history')).json()
  const jobs = await (await api.get(f.url + '/api/v1/jobs')).json()
  const networks = await (await api.get(f.url + '/api/v1/networks')).json()
  const before = JSON.parse(fs.readFileSync(output + '/browser.json', 'utf8'))
  assert.equal(history.items.length, before.history.remaining)
  assert.ok(!history.items.some(r => r.id === before.history.deleted), 'deleted capture reappeared after restart')
  assert.equal(jobs.items.length, 5); assert.ok(jobs.items.every(j => j.state === 'succeeded'))
  assert.equal(networks.items.length, 3)
  const page = await context.newPage(); await page.goto(f.url + '/history')
  await page.getByLabel('히스토리 검색', {exact: true}).waitFor()
  await page.getByRole('button', {name: /^node.start/}).waitFor()
  fs.writeFileSync(output + '/history-restart.json', JSON.stringify({historyPreserved: 4, tombstonePreserved: true, jobsPreserved: 5, networksPreserved: 3, browserVersion: browser.version()}, null, 2))
} finally { await browser.close() }
