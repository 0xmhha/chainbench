// differingParts splits a value into runs that match the other value and runs
// that do not, so a difference of one digit in a long hash can be seen. Only
// strings of equal length line up; anything else is returned whole.
export function differingParts(value, other) {
 const text = typeof value === 'string' ? value : JSON.stringify(value)
 if (typeof value !== 'string' || typeof other !== 'string' || value.length !== other.length) return [{ text, differs: false }]
 const parts = []
 for (let i = 0; i < text.length; i++) {
  const differs = text[i] !== other[i]
  const last = parts.at(-1)
  if (last?.differs === differs) last.text += text[i]
  else parts.push({ text: text[i], differs })
 }
 return parts
}
