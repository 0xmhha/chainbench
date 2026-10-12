import { responseError } from './api-error.mjs'

// webApi returns a JSON caller for /api/v1 bound to the signed-in session. It
// sends the session's CSRF token on every request and an If-Match revision
// when one is given. A 204 response returns null.
export function webApi(session, fetcher = (...args) => fetch(...args)) {
 return async function api(path, method = 'GET', body, revision) {
  const headers = { 'Content-Type': 'application/json' }
  if (session?.csrfToken) headers['X-CSRF-Token'] = session.csrfToken
  if (revision) headers['If-Match'] = `"${revision}"`
  const response = await fetcher(`/api/v1/${path}`, { method, headers, body: body === undefined ? undefined : JSON.stringify(body) })
  if (!response.ok) throw await responseError(response)
  return response.status === 204 ? null : response.json()
 }
}
