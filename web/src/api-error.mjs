// Turns an API error response into the message shown to the user. Validation
// failures (400, 409, 422) carry the server's redacted reason; other failures
// carry only their status text, which is shown once rather than as a reason.
export function errorMessage(status, text) {
 let body = null
 try { body = JSON.parse(text) } catch { body = null }
 if (!body || typeof body !== 'object') return `${status}: ${(text ?? '').trim() || 'request failed'}`
 const errors = Array.isArray(body.errors) ? body.errors.map(e => e.path ? `${e.path}: ${e.message}` : e.message).filter(Boolean) : []
 if (errors.length) return `${status}: ${errors.join('; ')}`
 const message = typeof body.message === 'string' && body.message ? body.message : typeof body.code === 'string' ? body.code : 'request failed'
 return `${status}: ${message}`
}

export async function responseError(response) {
 return new Error(errorMessage(response.status, await response.text()))
}
