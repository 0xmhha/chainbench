// Pure helpers behind the log viewer: what to ask the server, how to merge
// nodes, which rows to render and where a search hit sits in a line.

// The levels the server reports, most severe first.
export const LOG_LEVELS = ['CRIT', 'ERROR', 'WARN', 'INFO', 'DEBUG', 'TRACE']

const tones = { CRIT: 'danger', ERROR: 'danger', WARN: 'warn', INFO: 'info', DEBUG: 'faint', TRACE: 'faint' }

export function levelTone(level) {
 return tones[level] ?? 'none'
}

// highlightSegments splits a line around each case-insensitive occurrence of
// needle. The segments rejoin into the exact line, so the rendered text stays
// what the node wrote.
export function highlightSegments(text, needle) {
 if (!needle) return [{ text, match: false }]
 const haystack = text.toLowerCase(), probe = needle.toLowerCase()
 const out = []
 let at = 0
 for (let hit = haystack.indexOf(probe); hit !== -1; hit = haystack.indexOf(probe, at)) {
  if (hit > at) out.push({ text: text.slice(at, hit), match: false })
  out.push({ text: text.slice(hit, hit + probe.length), match: true })
  at = hit + probe.length
 }
 if (at < text.length || out.length === 0) out.push({ text: text.slice(at), match: false })
 return out
}

// mergeByTime interleaves each node's entries into one time-ordered list,
// tagging every entry with its node. Equal times keep the node order given.
export function mergeByTime(byNode, order) {
 const rank = new Map(order.map((node, i) => [node, i]))
 const all = []
 for (const [node, entries] of Object.entries(byNode)) {
  entries.forEach((entry, i) => all.push({ ...entry, node, seq: i }))
 }
 return all.sort((a, b) => (Date.parse(a.time) - Date.parse(b.time)) || (rank.get(a.node) - rank.get(b.node)) || (a.seq - b.seq))
}

// LOG_PAGE_MAX is the largest page the log API serves; internal/dashboard
// refuses a larger limit with 400, so a query never asks for more.
export const LOG_PAGE_MAX = 100

export function logQuery({ from, to, runId, search, levels, cursor, limit }) {
 const q = new URLSearchParams()
 if (runId) q.set('runId', runId)
 if (from) q.set('from', from)
 if (to) q.set('to', to)
 const needle = search?.trim()
 if (needle) q.set('q', needle)
 if (levels?.length) q.set('level', levels.join(','))
 if (cursor) q.set('cursor', cursor)
 if (limit) q.set('limit', String(Math.min(limit, LOG_PAGE_MAX)))
 return q.toString()
}

// visibleRange is the slice of fixed-height rows inside the viewport, widened
// by overscan rows on each side so scrolling does not show blank rows.
export function visibleRange({ scrollTop, height, rowHeight, count, overscan }) {
 // A list that shrank under an old scroll offset shows its last screen.
 const rows = Math.ceil(height / rowHeight)
 const first = Math.min(Math.floor(scrollTop / rowHeight), Math.max(0, count - rows))
 const last = first + rows
 const start = Math.max(0, Math.min(first, count) - overscan)
 const end = Math.min(count, last + overscan)
 return { start: Math.min(start, end), end }
}

export function nearestIndex(entries, time) {
 const at = Date.parse(time)
 let best = -1, distance = Infinity
 entries.forEach((entry, i) => {
  const d = Math.abs(Date.parse(entry.time) - at)
  if (d < distance) { best = i; distance = d }
 })
 return best
}
