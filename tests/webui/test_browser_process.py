import subprocess
import sys
import time
import unittest
from browser_process import run_browser


class BrowserProcessTests(unittest.TestCase):
    def test_preserves_failure_and_output(self):
        result = run_browser([sys.executable, '-c', 'print("fixture failure");raise SystemExit(3)'], 5)
        self.assertEqual(result.returncode, 3)
        self.assertIn('fixture failure', result.stdout)

    def test_timeout_kills_descendants_holding_output_pipe(self):
        script = 'import subprocess,sys,time;subprocess.Popen([sys.executable,"-c","import time;time.sleep(60)"]);print("owned child",flush=True);time.sleep(60)'
        started = time.monotonic()
        with self.assertRaises(subprocess.TimeoutExpired) as caught:
            run_browser([sys.executable, '-c', script], .3)
        self.assertIn('owned child', caught.exception.output)
        self.assertLess(time.monotonic() - started, 5)


if __name__ == '__main__':
    unittest.main()
