// The fixed screens, and the one detail route the server also serves the
// shell for (internal/dashboard/spa.go). Anything else shows the dashboard.
export const pages = ['/', '/chains', '/tests', '/monitoring', '/history', '/settings']
const job = /^\/jobs\/([0-9a-f]{32})$/

export function parseRoute(path) {
 const m = job.exec(path)
 if (m) return { page: '/jobs', jobId: m[1] }
 return { page: pages.includes(path) ? path : '/' }
}
