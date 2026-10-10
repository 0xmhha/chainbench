"""Live regression: verification must leave workspace SPA and receipts untouched."""
import hashlib
from pathlib import Path
import subprocess
import unittest


class VerificationIsolationTests(unittest.TestCase):
    def test_live_verifier_preserves_workspace_outputs(self):
        workspace = Path(__file__).resolve().parents[2]
        roots = [workspace / 'internal/dashboard/spa',
                 workspace / 'chainbench-out/web-ui-acceptance/WEB-05']
        def snapshot():
            return {str(p.relative_to(workspace)): hashlib.sha256(p.read_bytes()).hexdigest()
                    for root in roots for p in root.rglob('*') if p.is_file()}
        before = snapshot()
        self.assertTrue(before, 'workspace outputs must exist before checking isolation')
        result = subprocess.run(
            ['bash', 'tests/webui/verify.sh', '--criterion', 'WEB-05', '--require-live',
             '--output', 'chainbench-out/web-ui-acceptance'],
            cwd=workspace, capture_output=True, text=True, timeout=1800)
        self.assertIn(result.returncode, (0, 1), result.stdout + result.stderr)
        self.assertEqual(snapshot(), before, 'verifier changed workspace build/receipts')
        if result.returncode == 1:
            self.assertIn('Live recheck incomplete:', result.stderr)
        else:
            self.assertIn('WEB-05 PASS', result.stdout)


if __name__ == '__main__':
    unittest.main()
