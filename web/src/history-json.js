// Keep exact large-number literals for display; the downloaded bundle retains
// the original JSON numbers. Older browsers show an explicit unavailable value.
export function parseHistoryJSON(text) {
  return JSON.parse(text, (_key, value, context) => {
    if (typeof value === 'number' && ((!Number.isSafeInteger(value) && Number.isInteger(value)) || !Number.isFinite(value))) {
      return context?.source ?? '[정확한 수치 표시 불가 · 결과 내보내기에서 확인]'
    }
    return value
  })
}
