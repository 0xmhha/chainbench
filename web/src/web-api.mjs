import { apiClient } from './api-client.mjs'

// webApi returns a JSON caller for /api/v1 bound to the signed-in session. It
// sends the session's CSRF token on every request and an If-Match revision
// when one is given. A 204 response returns null. It is the shared client in
// the call shape the settings screens already use.
export function webApi(session, fetcher = (...args) => fetch(...args)) {
 const { request } = apiClient(session, fetcher)
 return function api(path, method = 'GET', body, revision) {
  return request(path, { method, body, revision })
 }
}
