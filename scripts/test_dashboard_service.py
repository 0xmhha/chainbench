#!/usr/bin/env python3
"""Verify the dashboard login service script against stand-in system commands."""
from __future__ import annotations

import os
import plistlib
import stat
import subprocess
import tempfile
import unittest
from pathlib import Path

SCRIPT = Path(__file__).resolve().with_name('dashboard-service.sh')
LABEL = 'com.chainbench.dashboard'


class DashboardServiceTest(unittest.TestCase):
    def setUp(self) -> None:
        temp = tempfile.TemporaryDirectory()
        self.addCleanup(temp.cleanup)
        self.root = Path(temp.name)
        self.home = self.root / 'home'
        self.home.mkdir()
        self.stubs = self.root / 'stubs'
        self.stubs.mkdir()
        self.calls = self.root / 'calls'
        self.plist = self.home / 'Library' / 'LaunchAgents' / f'{LABEL}.plist'
        self.stub('uname', 'echo Darwin')
        self.stub('lsof', 'exit 1')
        self.stub('curl', 'exit 0')
        self.stub('launchctl', f'''echo "$*" >> "{self.calls}"
[ "$1" = print ] && [ ! -e "{self.root}/loaded" ] && exit 113
[ "$1" = bootstrap ] && touch "{self.root}/loaded"
[ "$1" = bootout ] && rm -f "{self.root}/loaded"
exit 0''')

    def stub(self, name: str, body: str) -> None:
        path = self.stubs / name
        path.write_text('#!/bin/sh\n' + body + '\n')
        path.chmod(path.stat().st_mode | stat.S_IXUSR)

    def run_script(self, *args: str) -> subprocess.CompletedProcess[str]:
        env = {'HOME': str(self.home), 'PATH': f'{self.stubs}:{os.environ["PATH"]}'}
        return subprocess.run(['bash', str(SCRIPT), *args], env=env, text=True, capture_output=True, timeout=30)

    def calls_made(self) -> list[str]:
        return self.calls.read_text().splitlines() if self.calls.exists() else []

    def test_install_writes_an_agent_that_runs_from_the_checkout(self) -> None:
        result = self.run_script('install')
        self.assertEqual(result.returncode, 0, result.stderr)
        agent = plistlib.loads(self.plist.read_bytes())
        repo = SCRIPT.parent.parent
        self.assertEqual(agent['Label'], LABEL)
        self.assertEqual(agent['ProgramArguments'], [str(repo / 'bin' / 'chainbench-dashboard')])
        self.assertEqual(agent['WorkingDirectory'], str(repo))
        self.assertTrue(agent['RunAtLoad'])
        self.assertTrue(agent['KeepAlive'])
        self.assertEqual(agent['StandardErrorPath'], str(repo / 'chainbench-out' / 'dashboard.log'))
        self.assertIn(str(self.stubs), agent['EnvironmentVariables']['PATH'])
        self.assertEqual(stat.S_IMODE(self.plist.stat().st_mode), 0o600)
        self.assertTrue(any(call.startswith('bootstrap ') for call in self.calls_made()))

    def test_install_replaces_a_loaded_service_so_a_new_build_runs(self) -> None:
        self.run_script('install')
        result = self.run_script('install')
        self.assertEqual(result.returncode, 0, result.stderr)
        verbs = [call.split()[0] for call in self.calls_made()]
        self.assertLess(verbs.index('bootout'), len(verbs) - 1 - verbs[::-1].index('bootstrap'))

    def test_install_refuses_a_port_held_by_another_process(self) -> None:
        self.stub('lsof', 'echo 4242')
        result = self.run_script('install')
        self.assertEqual(result.returncode, 1)
        self.assertIn('8787', result.stderr)
        self.assertIn('4242', result.stderr)
        self.assertFalse(self.plist.exists())
        self.assertFalse(any(call.startswith('bootstrap ') for call in self.calls_made()))

    def test_install_fails_when_the_dashboard_never_answers(self) -> None:
        self.stub('curl', 'exit 7')
        self.stub('sleep', 'exit 0')
        result = self.run_script('install')
        self.assertEqual(result.returncode, 1)
        self.assertIn('dashboard.log', result.stderr)

    def test_install_escapes_the_path_for_the_plist(self) -> None:
        link = self.root / 'a&b'
        link.mkdir()
        env = {'HOME': str(self.home), 'PATH': f'{link}:{self.stubs}:{os.environ["PATH"]}'}
        result = subprocess.run(['bash', str(SCRIPT), 'install'], env=env, text=True, capture_output=True, timeout=30)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn(str(link), plistlib.loads(self.plist.read_bytes())['EnvironmentVariables']['PATH'])

    def test_uninstall_removes_the_agent_and_keeps_the_data(self) -> None:
        self.run_script('install')
        result = self.run_script('uninstall')
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertFalse(self.plist.exists())
        self.assertIn('bootout', [call.split()[0] for call in self.calls_made()])

    def test_uninstall_without_a_service_is_quiet(self) -> None:
        result = self.run_script('uninstall')
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertNotIn('bootout', [call.split()[0] for call in self.calls_made()])

    def test_status_reports_whether_the_service_is_loaded(self) -> None:
        self.assertEqual(self.run_script('status').returncode, 1)
        self.run_script('install')
        self.assertEqual(self.run_script('status').returncode, 0)

    def test_other_platforms_are_refused_by_name(self) -> None:
        self.stub('uname', 'echo Linux')
        result = self.run_script('install')
        self.assertEqual(result.returncode, 1)
        self.assertIn('Linux', result.stderr)
        self.assertFalse(self.plist.exists())

    def test_unknown_commands_show_the_usage(self) -> None:
        result = self.run_script('restart')
        self.assertEqual(result.returncode, 2)
        self.assertIn('usage', result.stderr)


if __name__ == '__main__':
    unittest.main()
