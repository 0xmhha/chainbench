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
