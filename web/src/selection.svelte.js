// selection is the workspace, network and chain the user picked on one screen,
// shared by every screen until they pick another one.
export const selection = $state({ workspaceId: '', networkId: '', chain: '' })

export function selectWorkspace(id) { selection.workspaceId = id ?? '' }
export function selectNetwork(id) { selection.networkId = id ?? '' }
export function selectChain(id) { selection.chain = id ?? '' }
