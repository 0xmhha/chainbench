// WEB-14 routes: every SPA page served by the real Go server is reachable by
// navigation, survives a refresh and opens from a direct link.
import {launchOwnedBrowser} from './owned-browser.mjs'
import fs from 'node:fs'
import path from 'node:path'
import assert from 'node:assert/strict'

const [fixtureFile, out] = process.argv.slice(2)
const f = JSON.parse(fs.readFileSync(fixtureFile, 'utf8'))
const owned = await launchOwnedBrowser()
const routes = [
  ['/', '대시보드'], ['/chains', '체인 · Workspace'], ['/tests', '테스트'],
  ['/monitoring', '모니터링'], ['/history', '히스토리'], ['/settings', '설정'],
]
try {
  const context = await owned.browser.newContext({viewport: {width: 1280, height: 900}})
  const r = await context.request.post(f.url + '/api/v1/bootstrap', {data: {username: 'web14-admin', password: f.password, setupToken: f.setupToken}})
  assert.equal(r.status(), 201, await r.text())
  const page = await context.newPage(), errors = []
  page.on('pageerror', e => errors.push(e.message))
  const current = label => page.locator('nav a[aria-current="page"]').filter({hasText: label}).waitFor({timeout: 15000})
  await page.goto(f.url + '/')
  await current('대시보드')
  const observed = []
  for (const [route, label] of routes) {
    await page.locator('nav a', {hasText: label}).first().click()
    await current(label)
    assert.equal(new URL(page.url()).pathname, route)
    await page.reload()
    await current(label)
    const direct = await context.newPage()
    direct.on('pageerror', e => errors.push(e.message))
    const response = await direct.goto(f.url + route)
    assert.equal(response.status(), 200, route + ' is not served on a direct load')
    await direct.locator('nav a[aria-current="page"]').filter({hasText: label}).waitFor({timeout: 15000})
    await direct.screenshot({path: path.join(out, 'route' + (route === '/' ? '-root' : route.replaceAll('/', '-')) + '.png'), fullPage: true})
    await direct.close()
    observed.push({path: route, label})
  }
  await page.goBack(); await page.goForward()
  assert.deepEqual(errors, [], 'page errors: ' + errors.join('; '))
  fs.writeFileSync(path.join(out, 'browser.json'), JSON.stringify({browserVersion: owned.browser.version(), routes: observed}, null, 2))
} finally { await owned.stop() }
