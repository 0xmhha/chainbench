import test from 'node:test'
import assert from 'node:assert/strict'
import { shareFeed } from '../../web/src/shared-feed.mjs'

function fakeFeed(log) {
 return () => ({ state: { id: log.created = (log.created ?? 0) + 1 }, refresh: () => log.refreshes++, start() { log.starts++; return () => log.stops++ } })
}

test('every screen on a page reads one stream, opened by the first and closed by the last', () => {
 const log = { starts: 0, stops: 0, refreshes: 0 }
 const shared = shareFeed(fakeFeed(log))
 const app = shared(), monitor = shared()
 assert.equal(app.state, monitor.state)
 assert.equal(log.created, 1)
 const stopApp = app.start(), stopMonitor = monitor.start()
 assert.equal(log.starts, 1)
 stopMonitor()
 assert.equal(log.stops, 0)
 stopMonitor()
 assert.equal(log.stops, 0, 'a screen releasing twice does not close the stream for the others')
 stopApp()
 assert.equal(log.stops, 1)
 monitor.refresh()
 assert.equal(log.refreshes, 1)
 const again = shared().start()
 assert.equal(log.starts, 2, 'a later screen opens it again')
 again()
 assert.equal(log.stops, 2)
})
