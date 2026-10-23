// statusIcon pairs a state the server reports with an icon and a tone, so a
// state is never shown by color alone. Unknown states are marked as unknown.
const groups = [
 ['✓', 'ok', ['succeeded', 'success', 'pass', 'passed', 'verified', 'supported', 'cleaned', 'compatible']],
 ['✕', 'bad', ['failed', 'fail', 'error', 'incompatible', 'cleanup_failed', 'refused', 'binary-absent']],
 ['●', 'active', ['running', 'accepted', 'started']],
 ['◐', 'warn', ['cancelling', 'pending']],
 ['!', 'warn', ['interrupted', 'stale', 'unknown', 'unverified', 'ownership_mismatch', 'blocked']],
 ['○', 'neutral', ['cancelled', 'stopped', 'retained', 'documented', 'absent', 'skip', 'skipped']],
]
export function statusIcon(state) {
 for (const [icon, tone, states] of groups) if (states.includes(state)) return { icon, tone }
 return { icon: '?', tone: 'neutral' }
}
