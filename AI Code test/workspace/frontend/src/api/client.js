/**
 * Every backend endpoint answers HTTP 200 with
 * `{ code, message, error_text?, data: [] }`. apiRequest returns `data[0]`
 * on code 200 and throws ApiError otherwise. All API calls go through here:
 * never call fetch directly.
 */

export const API_BASE = '/api/v1'

export class ApiError extends Error {
  /**
   * @param {number} status   the envelope's `code` (0 = network failure)
   * @param {string} code     stable error code from `message`, e.g. TENANT_NOT_FOUND
   * @param {string} [errorText] backend translation, used as a fallback
   * @param {any} [data]      `data[0]`, e.g. `{ valid, issues }` for CONFIG_INVALID
   */
  constructor(status, code, errorText = '', data = null) {
    super(code)
    this.name = 'ApiError'
    this.status = status
    this.code = code
    this.errorText = errorText
    this.data = data
  }
}

/**
 * @param {string} path   relative to /api/v1, e.g. '/tenants'
 * @param {{ method?: string, body?: any, signal?: AbortSignal }} [options]
 * @returns {Promise<any>} data[0] (null when data is empty)
 */
export async function apiRequest(path, { method = 'GET', body, signal } = {}) {
  let response
  try {
    // Absolute same-origin URL: identical in the browser, and resolvable by fetch in tests.
    response = await fetch(`${window.location.origin}${API_BASE}${path}`, {
      method,
      credentials: 'same-origin',
      headers: body === undefined ? { Accept: 'application/json' } : { Accept: 'application/json', 'Content-Type': 'application/json' },
      body: body === undefined ? undefined : JSON.stringify(body),
      signal,
    })
  } catch (err) {
    if (err?.name === 'AbortError') throw err
    throw new ApiError(0, 'NETWORK_ERROR')
  }
  let envelope
  try {
    envelope = await response.json()
  } catch {
    throw new ApiError(response.status, 'NETWORK_ERROR')
  }
  const data = Array.isArray(envelope?.data) ? envelope.data : []
  if (envelope?.code !== 200) {
    throw new ApiError(envelope?.code ?? response.status, envelope?.message || 'INTERNAL_ERROR', envelope?.error_text || '', data[0] ?? null)
  }
  return data[0] ?? null
}
