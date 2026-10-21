<script>
  import { webApi } from './web-api.mjs'
  import CredentialManager from './CredentialManager.svelte'
  import UserManager from './UserManager.svelte'
  import AuditPanel from './AuditPanel.svelte'
  import AssetManager from './AssetManager.svelte'
  import ManifestManager from './ManifestManager.svelte'
  let { webSession, assetRevision = 0, onuploaded = () => {} } = $props()
  let access = $state(null), status = $state('')
  const role = $derived(webSession?.user.role)
  $effect(() => { if (webSession) webApi(webSession)('contracts/access').then(c => { access = c }).catch(e => { status = e.message }) })
</script>

{#if status}<p class="error" role="alert">{status}</p>{/if}
{#if access && role !== 'viewer'}<CredentialManager {webSession} kinds={access.credentialKinds} />{/if}
{#if access && role === 'administrator'}
  {#if webSession.accountsEnabled !== false}<UserManager {webSession} roles={access.roles} />{/if}
  <AuditPanel {webSession} />
{/if}
<AssetManager {webSession} {onuploaded} />
<ManifestManager {webSession} {assetRevision} />

<style>
  .error { padding: 12px 16px; border-left: 3px solid var(--danger); color: var(--danger); background: var(--danger-bg); }
</style>
