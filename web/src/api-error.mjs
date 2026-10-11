// Turns an API error response into the message shown to the user. Validation
// failures (400, 409, 422) carry the server's redacted reason; other failures
// carry only their status text, which is shown once rather than as a reason.
export function errorMessage(status, text) {
 let body = null
 try { body = JSON.parse(text) } catch { body = null }
 if (!body || typeof body !== 'object') return `${status}: ${(text ?? '').trim() || 'request failed'}`
 const list = [...(Array.isArray(body.fieldErrors) ? body.fieldErrors : []), ...(Array.isArray(body.errors) ? body.errors : [])]
 const errors = list.map(e => e?.path ? `${e.path}: ${e.message}` : e?.message).filter(Boolean)
 if (errors.length) return `${status}: ${errors.join('; ')}`
 const message = typeof body.message === 'string' && body.message ? body.message : typeof body.code === 'string' ? body.code : 'request failed'
 return `${status}: ${message}`
}

// responseError also keeps the status and the body's details (busy holders,
// missing access), so a screen can act on them beyond showing the message.
export async function responseError(response) {
 const text = await response.text()
 const error = new Error(errorMessage(response.status, text))
 let details = {}
 try { const body = JSON.parse(text); if (body?.details && typeof body.details === 'object') details = body.details } catch { details = {} }
 error.status = response.status
 error.details = details
 return error
}
