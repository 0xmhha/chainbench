// allItems reads a paged list to its end by following nextCursor. get takes a
// path and returns the parsed page ({items, nextCursor}).
export async function allItems(get, path) {
 const items = []
 const sep = path.includes('?') ? '&' : '?'
 let cursor = null
 do {
  const page = await get(`${path}${sep}limit=100${cursor ? `&cursor=${encodeURIComponent(cursor)}` : ''}`)
  items.push(...(page?.items ?? []))
  cursor = page?.nextCursor ?? null
 } while (cursor)
 return items
}
