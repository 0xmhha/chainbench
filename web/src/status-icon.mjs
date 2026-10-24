// statusIcon pairs a state the server reports with a drawn mark and a tone, so a
// state is never shown by color alone. Unknown states are marked as unknown.
const groups = [
 ['check', 'ok', ['succeeded', 'success', 'pass', 'passed', 'verified', 'supported', 'cleaned', 'compatible']],
 ['cross', 'bad', ['failed', 'fail', 'error', 'incompatible', 'cleanup_failed', 'refused', 'binary-absent']],
 ['dot', 'active', ['running', 'accepted', 'started']],
 ['half', 'warn', ['cancelling', 'pending']],
 ['bang', 'warn', ['interrupted', 'stale', 'unknown', 'unverified', 'ownership_mismatch', 'blocked']],
 ['ring', 'neutral', ['cancelled', 'stopped', 'retained', 'documented', 'absent', 'skip', 'skipped']],
]
export function statusIcon(state) {
 for (const [icon, tone, states] of groups) if (states.includes(state)) return { icon, tone }
 return { icon: 'unknown', tone: 'neutral' }
}
