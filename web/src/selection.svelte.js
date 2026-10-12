// selection is the workspace, network and chain the user picked on one screen,
// shared by every screen until they pick another one. It survives a reload in
// this browser tab only; storage that is unavailable leaves it empty.
const key = 'chainbench.selection'
function restore() {
 try { const v = JSON.parse(sessionStorage.getItem(key) ?? '{}'); return { workspaceId: String(v.workspaceId ?? ''), networkId: String(v.networkId ?? ''), chain: String(v.chain ?? '') } } catch { return { workspaceId: '', networkId: '', chain: '' } }
}
export const selection = $state(restore())
function save() { try { sessionStorage.setItem(key, JSON.stringify(selection)) } catch { /* the choice still holds for this page */ } }

export function selectWorkspace(id) { selection.workspaceId = id ?? ''; save() }
export function selectNetwork(id) { selection.networkId = id ?? ''; save() }
export function selectChain(id) { selection.chain = id ?? ''; save() }

// pendingControl is a node control chosen on a node page and not yet planned.
// The job form takes it once, so a later visit starts from a clean form.
export const pendingControl = $state({ value: null })
export function requestControl(workspaceId, nodeId, operation) { selectWorkspace(workspaceId); pendingControl.value = { workspaceId, nodeId, operation } }
export function takeControl() { const v = pendingControl.value; pendingControl.value = null; return v }

// monitorRange is the time range every monitoring view reads: preset minutes
// ending now, or a custom from/to when preset is 0. Node and metric narrow the
// charts; empty means all.
export const monitorRange = $state({ preset: 15, from: '', to: '', node: '', metric: '' })
