// The fixed screens, and the detail routes the server also serves the
// shell for (internal/dashboard/spa.go). Anything else shows the dashboard.
export const pages = ['/', '/chains', '/tests', '/monitoring', '/history', '/settings']
const job = /^\/jobs\/([0-9a-f]{32})$/
const node = /^\/nodes\/([0-9a-f]{32})\/([A-Za-z0-9_.-]{1,64})$/

export function parseRoute(path) {
 const m = job.exec(path)
 if (m) return { page: '/jobs', jobId: m[1] }
 const n = node.exec(path)
 if (n) return { page: '/nodes', networkId: n[1], nodeId: n[2] }
 return { page: pages.includes(path) ? path : '/' }
}
