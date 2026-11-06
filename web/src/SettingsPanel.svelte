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
{/if}
<AssetManager {webSession} {onuploaded} />
<ManifestManager {webSession} {assetRevision} />
{#if access && role === 'administrator'}<details class="audit"><summary>감사 기록</summary><AuditPanel {webSession} /></details>{/if}

<style>
  .audit { margin-top: var(--space-4); }
  .audit > summary { cursor: pointer; padding: var(--space-2) 0; font: 600 var(--text-sm) var(--font-cond); color: var(--text); }
</style>
