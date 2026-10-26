// @ts-check
// One client for every /api/v1 request: it sends the session's CSRF token, an
// If-Match revision and an Idempotency-Key when given, turns an error
// response into the error the screens already show (api-error.mjs), and runs
// the response through a decoder so a screen receives a checked value or a
// FormatError, never a half-read object.
import { responseError } from './api-error.mjs'

/**
 * @typedef {object} RequestOptions
 * @property {string} [method] GET when absent
 * @property {unknown} [body] sent as JSON, or as is when it is FormData
 * @property {number|string} [revision] sent as If-Match
 * @property {string} [idempotencyKey]
 * @property {AbortSignal} [signal]
 * @property {RequestCache} [cache]
 * @property {(text: string) => unknown} [parse] reads the body text; JSON.parse when absent
 * @property {(value: unknown) => any} [decode] checks the parsed body
 */

/**
 * @param {{csrfToken?: string}|null|undefined} session
 * @param {typeof fetch} [fetcher]
 */
export function apiClient(session, fetcher = (...args) => fetch(...args)) {
 /**
  * @param {string} path below /api/v1/, query included
  * @param {RequestOptions} [options]
  * @returns {Promise<any>}
  */
 async function request(path, options = {}) {
  const response = await send(path, options)
  if (response.status === 204) return null
  const text = await response.text()
  const value = text === '' ? null : (options.parse ?? JSON.parse)(text)
  return options.decode ? options.decode(value) : value
 }

 /**
  * send returns the successful response itself, for downloads.
  * @param {string} path
  * @param {RequestOptions} [options]
  */
 async function send(path, options = {}) {
  const { method = 'GET', body, revision, idempotencyKey, signal, cache } = options
  /** @type {Record<string, string>} */
  const headers = {}
  const form = typeof FormData !== 'undefined' && body instanceof FormData
  if (body !== undefined && !form) headers['Content-Type'] = 'application/json'
  if (session?.csrfToken) headers['X-CSRF-Token'] = session.csrfToken
  if (revision) headers['If-Match'] = `"${revision}"`
  if (idempotencyKey) headers['Idempotency-Key'] = idempotencyKey
  const response = await fetcher(`/api/v1/${path}`, {
   method, headers, signal, cache,
   body: body === undefined ? undefined : form ? /** @type {FormData} */ (body) : JSON.stringify(body),
  })
  if (!response.ok) throw await responseError(response)
  return response
 }

 return { request, send }
}

/**
 * readApi reads without a session token: GET requests are authorised by the
 * session cookie alone.
 * @param {string} path
 * @param {RequestOptions} [options]
 */
export function readApi(path, options = {}) {
 return apiClient(null).request(path, options)
}
