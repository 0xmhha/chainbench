"""Preparation must select the same server as the gated Go tests."""
import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]
SCRIPT = ROOT / 'tests/webui/fixtures/docker_regression_prepare.sh'
ID = 'a' * 64


class DockerPreparationTests(unittest.TestCase):
    def invoke(self, case='valid', own_ssh=False):
        with tempfile.TemporaryDirectory() as raw:
            root = Path(raw).resolve()
            build = root / 'chosen'
            build.mkdir()
            (build / 'docker-compose.yml').write_text('services: {}\n')
            ssh = "{user: customlogin, port: 10022, password: fixture-only, sudo: true, insecure_host_key: true}"
            host = '{name: server1, addr: 172.31.42.11' + (', ssh: ' + ssh if own_ssh else '') + '}'
            (build / 'server-set.yaml').write_text('version: 2\npool:\n  hosts: [' + host + ']\n  slots: 1\nssh: ' + (ssh.replace('customlogin', 'defaultlogin') if own_ssh else ssh) + '\n')
            (build / 'localmap.yaml').write_text('hosts:\n  172.31.42.11:\n    host: 127.0.0.1\n    ports: {10022: 45001}\n')
            inspect = {'Id': ID, 'State': {'Running': True}, 'Config': {'Labels': {
                'com.docker.compose.service': 'server1',
                'com.docker.compose.project.config_files': str(build / 'docker-compose.yml'),
                'com.docker.compose.project.working_dir': str(build)}},
                'NetworkSettings': {'Networks': {'owned': {'IPAddress': '172.31.42.11'}},
                                    'Ports': {'10022/tcp': [{'HostIp': '127.0.0.1', 'HostPort': '45001'}]}}}
            if case == 'wrong-compose':
                inspect['Config']['Labels']['com.docker.compose.project.config_files'] = '/other/docker-compose.yml'
            if case == 'wrong-ip':
                inspect['NetworkSettings']['Networks']['owned']['IPAddress'] = '172.31.42.12'
            if case == 'wrong-port':
                inspect['NetworkSettings']['Ports']['10022/tcp'][0]['HostPort'] = '45002'
            if case == 'stopped':
                inspect['State']['Running'] = False
            if case == 'wrong-service':
                inspect['Config']['Labels']['com.docker.compose.service'] = 'server2'
            if case == 'missing-map':
                (build / 'localmap.yaml').unlink()
            payload = root / 'inspect.json'
            payload.write_text(json.dumps([inspect]))
            log = root / 'calls.jsonl'
            fake = root / 'docker'
            fake.write_text('#!/usr/bin/env python3\nimport json,os,sys\na=sys.argv[1:]\nwith open(os.environ["FAKE_LOG"],"a") as f: f.write(json.dumps(a)+"\\n")\n'
                            'if "ps" in a: print(("b"*64+"\\n" if os.environ["FAKE_CASE"]=="ambiguous" else "")+"a"*64)\n'
                            'elif a and a[0]=="inspect":\n print("true" if "-f" in a else open(os.environ["FAKE_INSPECT"]).read())\n'
                            'elif "mktemp" in " ".join(a): print("/tmp/chainbench-regression.FIXTURE")\n')
            fake.chmod(0o755)
            environment = {**os.environ, 'PATH': str(root) + os.pathsep + os.environ['PATH'],
                           'CHAINBENCH_DOCKER_SERVERS': str(build), 'FAKE_LOG': str(log),
                           'FAKE_INSPECT': str(payload), 'FAKE_CASE': case, 'PYTHONDONTWRITEBYTECODE': '1'}
            if case == 'unset':
                environment.pop('CHAINBENCH_DOCKER_SERVERS')
            result = subprocess.run(['bash', str(SCRIPT)], cwd=ROOT, env=environment, text=True, capture_output=True)
            calls = [json.loads(line) for line in log.read_text().splitlines()] if log.exists() else []
            return result, calls

    def test_selected_config_and_resolved_login_are_used(self):
        result, calls = self.invoke()
        self.assertEqual(result.returncode, 0, result.stderr)
        writes = [a for a in calls if a[0] in ['exec', 'cp']]
        self.assertTrue(writes)
        self.assertTrue(all(ID in a or any(x.startswith(ID + ':') for x in a) for a in writes), calls)
        self.assertNotIn('devuser1', json.dumps(calls))
        self.assertIn('customlogin', json.dumps(calls))

    def test_host_specific_login_is_resolved(self):
        result, calls = self.invoke(own_ssh=True)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn('customlogin', json.dumps(calls))

    def test_missing_gate_refuses_before_docker(self):
        result, calls = self.invoke('unset')
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(calls, [])

    def test_wrong_target_refuses_without_mutation(self):
        for case in ['wrong-compose', 'wrong-ip', 'wrong-port', 'stopped', 'wrong-service', 'missing-map', 'ambiguous']:
            with self.subTest(case=case):
                result, calls = self.invoke(case)
                self.assertNotEqual(result.returncode, 0, case)
                self.assertFalse([a for a in calls if a[0] in ['exec', 'cp']], calls)


if __name__ == '__main__':
    unittest.main()
