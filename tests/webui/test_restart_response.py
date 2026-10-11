"""The browser proof reads restart.response as soon as it exists, so the file must appear complete."""
import json
import os
from pathlib import Path
import tempfile
import unittest
from unittest import mock

import run_test_jobs


class RestartResponseTest(unittest.TestCase):
    def test_response_appears_only_once_complete(self):
        with tempfile.TemporaryDirectory() as root:
            runtime = Path(root)
            target = runtime / 'restart.response'
            seen = []
            real_replace = os.replace

            def replace(src, dst):
                # At the moment the name appears, the content must already be whole.
                seen.append((Path(dst), target.exists(), json.loads(Path(src).read_text())))
                real_replace(src, dst)

            with mock.patch.object(run_test_jobs.os, 'replace', side_effect=replace):
                run_test_jobs.publish_restart_response(runtime, {'oldPid': 1, 'newPid': 2})
            self.assertEqual(seen, [(target, False, {'oldPid': 1, 'newPid': 2})])
            self.assertEqual(json.loads(target.read_text()), {'oldPid': 1, 'newPid': 2})
            self.assertEqual(sorted(p.name for p in runtime.iterdir()), ['restart.response'])


if __name__ == '__main__':
    unittest.main()
