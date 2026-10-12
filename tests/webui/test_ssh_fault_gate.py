"""Verify the real-command test gate without substituting a transport."""
import hashlib
import os
from pathlib import Path
import shlex
import subprocess
import tempfile
import time
import unittest

from ssh_fault_gate import prepare_ssh_gate


class SSHFaultGateTests(unittest.TestCase):
    def test_probes_pass_through_without_pausing(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            digest = 'a' * 64
            gate = prepare_ssh_gate(root, digest, .3)
            result = subprocess.run([str(gate)], env={**os.environ, 'SSH_ORIGINAL_COMMAND': 'printf %s ' + digest}, capture_output=True, text=True, timeout=3)
            self.assertEqual(result.returncode, 0)
            self.assertEqual(result.stdout, digest)
            self.assertFalse((root / 'ssh/gate-copied').exists())

    def test_real_copy_finishes_before_pause_and_explicit_release(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            data = b'owned transfer bytes'
            digest = hashlib.sha256(data).hexdigest()
            gate = prepare_ssh_gate(root, digest, 2)
            target = root / digest
            command = 'cat > ' + shlex.quote(str(target)) + '; chmod 755 ' + shlex.quote(str(target))
            child = subprocess.Popen([str(gate)], env={**os.environ, 'SSH_ORIGINAL_COMMAND': command}, stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
            try:
                child.stdin.write(data)
                child.stdin.close()
                child.stdin = None
                for _ in range(100):
                    if (root / 'ssh/gate-copied').exists():
                        break
                    time.sleep(.01)
                self.assertTrue((root / 'ssh/gate-copied').exists())
                self.assertEqual(target.read_bytes(), data)
                self.assertIsNone(child.poll(), 'command response was not held')
                self.assertEqual((root / 'ssh/gate-copied').stat().st_mode & 0o777, 0o600)
                (root / 'ssh/gate-release').write_text('release this owned command')
                stdout, stderr = child.communicate(timeout=3)
                self.assertEqual(child.returncode, 0, stderr)
                self.assertEqual(stdout, b'')
            finally:
                if child.poll() is None:
                    child.kill()
                    child.wait(timeout=3)

    def test_unreleased_command_has_a_bounded_failure(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            digest = 'b' * 64
            gate = prepare_ssh_gate(root, digest, .2)
            command = 'cat > ' + shlex.quote(str(root / digest))
            started = time.monotonic()
            result = subprocess.run([str(gate)], env={**os.environ, 'SSH_ORIGINAL_COMMAND': command}, input=b'owned data', capture_output=True, timeout=3)
            self.assertEqual(result.returncode, 124)
            self.assertLess(time.monotonic() - started, 2)


if __name__ == '__main__':
    unittest.main()
