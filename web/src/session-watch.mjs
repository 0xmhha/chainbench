// @ts-check
// A session lasts a fixed time the server states, so the page waits for that
// moment instead of asking every few seconds whether it is still signed in.

const longestTimer = 2 ** 31 - 1

/**
 * expiryDelay is how long until the session expires, for a timer: zero once
 * it has, and null when the session states no readable time.
 * @param {{expiresAt?: string}|null|undefined} session
 * @param {number} [now]
 */
export function expiryDelay(session, now = Date.now()) {
 const at = Date.parse(session?.expiresAt ?? '')
 if (Number.isNaN(at)) return null
 return Math.min(Math.max(at - now, 0), longestTimer)
}
