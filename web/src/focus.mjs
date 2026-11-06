import { tick } from 'svelte'

// keepFocus remembers the focused control before a busy period disables it. A
// disabled control drops focus to the body, which sends a keyboard user back to
// the top of the page. The returned function puts it back once the page has
// updated, unless the work moved focus on purpose.
export function keepFocus(document = globalThis.document) {
 const before = document?.activeElement
 return async () => {
  await tick()
  const now = document?.activeElement
  if (before && before !== document.body && before.isConnected && (!now || now === document.body)) before.focus()
 }
}
