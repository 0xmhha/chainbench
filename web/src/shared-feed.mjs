// @ts-check
// shareFeed lets every screen on a page read one live feed: the first screen
// that starts it opens the connection, and it closes when the last one stops.
// Without it the always-present app frame and the screen inside it each held
// an event stream, doubling requests and reconnects.

/**
 * @template S
 * @param {() => {state: S, refresh: () => void, start: () => () => void}} create
 */
export function shareFeed(create) {
 /** @type {ReturnType<typeof create>|null} */
 let feed = null
 let users = 0
 /** @type {(() => void)|null} */
 let stop = null
 return function shared() {
  feed ??= create()
  const current = feed
  return {
   state: current.state,
   refresh: () => current.refresh(),
   start() {
    if (users++ === 0) stop = current.start()
    let released = false
    return () => {
     if (released) return
     released = true
     if (--users === 0) { stop?.(); stop = null }
    }
   },
  }
 }
}
