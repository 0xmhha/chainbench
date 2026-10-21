import { webApi } from './web-api.mjs'

// Login requirements and the browser's execution context are separate. The
// local context is provisioned automatically and never registers an account.
export async function openWebAccess(fetcher = (...args) => fetch(...args)) {
  const response = await fetcher('/api/v1/auth/config')
  if (response.status === 404) return { session: undefined, accountsEnabled: false, bootstrapRequired: false }
  if (!response.ok) throw new Error('접속 설정을 읽을 수 없습니다')
  const config = await response.json()
  const accountsEnabled = config.accountsEnabled !== false
  const api = webApi(undefined, fetcher)
  if (!accountsEnabled) {
    return { session: await api('auth/local', 'POST'), accountsEnabled: false, bootstrapRequired: false }
  }
  let session = null
  try { session = await api('auth/me') } catch { session = null }
  return { session, accountsEnabled: true, bootstrapRequired: config.bootstrapRequired }
}
