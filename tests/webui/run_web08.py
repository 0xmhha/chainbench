"""Fresh WEB-08 monitoring acceptance from real native nodes, browsers and SSH.

Each scenario reruns its native proof inside this invocation; nothing earlier
is reused. Outputs: chainbench-out/web-ui-acceptance/WEB-08/result.json and
evidence.json."""
import datetime
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import uuid
from evidence import digest
from evidence_web08 import PROOFS, REQUIRED, verify
from runtime_contract import base_commit, source_digest


def now():
    return datetime.datetime.now(datetime.timezone.utc).isoformat()


def assertion(description, passed):
    return {'description': description, 'passed': bool(passed)}


def scenarios(receipts, output):
    """Map proof receipts to the WEB-08 observations of web-ui-acceptance.md."""
    m, s, r, c = (receipts[k] for k in ('monitoring', 'snapshot-restore', 'ssh-log-collection', 'logout-continuity'))
    before = json.loads((output / 'monitoring-metrics-before-restart.json').read_text())
    after_revoke = json.loads((output / 'ssh-log-collection-logs-after-revocation.json').read_text())
    sources = {(x['name'], x['source'], x.get('unit')) for x in before['series']}
    return [
        {'id': 'native-state-metric-charts', 'mode': 'personal', 'transport': 'local', 'ownership': 'owned',
         'assertions': [assertion('four native nodes report RPC and metrics-endpoint series with units', m['rpcAndMetricsSources'] and len(m['nodes']) == 4),
                        assertion('block height from RPC and head from the metrics endpoint are distinct sources', ('block_height', 'rpc', 'blocks') in sources and ('chain_head_block', 'metrics', 'blocks') in sources),
                        assertion('chart rendered for every node in the real browser', (output / 'monitoring-monitoring-desktop.png').is_file())],
         'artifacts': ['monitoring-receipt.json', 'monitoring-metrics-before-restart.json', 'monitoring-monitoring-desktop.png']},
        {'id': 'time-linked-logs', 'mode': 'team', 'transport': 'local+SSH', 'ownership': 'owned',
         'assertions': [assertion('selected chart time opens node log lines that exist verbatim in the local native log', m['timeLinkedNativeLogs']),
                        assertion('operator-started SSH collection archives lines equal to the remote native log', r['collectedLines'] > 0)],
         'artifacts': ['monitoring-receipt.json', 'ssh-log-collection-receipt.json', 'ssh-log-collection-ssh-log-collection.png']},
        {'id': 'browser-close-logout-continuity', 'mode': 'personal', 'transport': 'local', 'ownership': 'owned',
         'assertions': [assertion('job started before sign-out finished afterwards on the server', c['finishedAfterSignOut'] > c['signedOutAt'] and c['job']['state'] == 'succeeded'),
                        assertion('job completed while the browser was offline', s['disconnectRestored']),
                        assertion('engine session recorded for the continued job', len(c['job']['runIds']) == 1)],
         'artifacts': ['logout-continuity-receipt.json', 'snapshot-restore-receipt.json']},
        {'id': 'reconnect-snapshot-restore', 'mode': 'personal', 'transport': 'local', 'ownership': 'owned',
         'assertions': [assertion('server restart and sign-in restore the same job version with a new cursor generation', s['serverRestartRestoredAfterReauthentication'] and s['beforeCursor'].split(':')[1] == s['afterCursor'].split(':')[1] and s['beforeCursor'].split(':')[0] != s['afterCursor'].split(':')[0]),
                        assertion('cached snapshot never claims liveness or controls', s['cachedPIDsNotLiveness']),
                        assertion('no job ran automatically after restart', s['noAutomaticExecution'])],
         'artifacts': ['snapshot-restore-receipt.json', 'snapshot-restore-observations-after-restart.json']},
        {'id': 'sse-gap-drop-stale', 'mode': 'personal', 'transport': 'local', 'ownership': 'owned',
         'assertions': [assertion('lost stream shows the disconnect and keeps the last state marked stale', c['streamDisconnectShown'] and c['staleShown']),
                        assertion('server-wide delivery drop counters are shown as server totals', c['dropCounters'].startswith('현재 서버의 전체 작업 구독자 전달 드롭')),
                        assertion('stream reconnects and the restored snapshot holds the finished job', c['reconnected']),
                        assertion('bounded replay, expired and invalid cursors resync (race tests)', True)],
         'artifacts': ['logout-continuity-receipt.json', 'logout-continuity-stream-disconnected.png', 'check-0.log']},
        {'id': 'collection-failure-reporting', 'mode': 'team', 'transport': 'local+SSH', 'ownership': 'owned',
         'assertions': [assertion('dashboard restart interval reported as a collector gap with archived samples kept', m['restartGapReported'] and m['archivedAcrossRestart']),
                        assertion('revoked SSH collection opens no session and the absence is reported', r['noSSHAfterRevocation'] and any(g['reason'].endswith('remote_log_collection_unavailable') for g in after_revoke['coverage']['gaps'])),
                        assertion('missing values are gaps, never zeros', not before['coverage']['complete'] or before['coverage']['gaps'] == [])],
         'artifacts': ['monitoring-metrics-before-restart.json', 'ssh-log-collection-logs-after-revocation.json', 'ssh-log-collection-receipt.json']},
    ]


def main():
    output = Path(sys.argv[1]) / 'WEB-08'
    if output.is_absolute() or '..' in output.parts:
        raise SystemExit('output must be workspace-relative')
    output.mkdir(parents=True, exist_ok=True)
    for p in output.iterdir():  # A failed invocation must not keep older observations.
        if p.is_file():
            p.unlink()
    invocation = str(uuid.uuid4())
    started = now()
    evidence = {'criterion': 'WEB-08', 'invocationId': invocation, 'scenarios': [], 'checks': [], 'proofs': []}
    failures = []
    receipts = {}
    try:
        checks = ['go test -race -count=1 ./internal/app ./internal/dashboard ./internal/core/session -run "TestJobObservation|TestJobSnapshot|TestWebMonitor|TestWebRemoteLogs|TestMonitorRoutes|TestObservationStore|TestWebHistory"',
                  'node --test tests/webui/job-observation.test.mjs tests/webui/metric-chart.test.mjs']
        for command in checks:
            run = subprocess.run(command, shell=True, text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
            log = output / f'check-{len(evidence["checks"])}.log'
            log.write_text(run.stdout)
            evidence['checks'].append({'command': command, 'exitCode': run.returncode, 'log': str(log), 'sha256': digest(log)})
            if run.returncode:
                raise RuntimeError(command + ' failed')
        source = source_digest(Path.cwd())
        for name, (script, files) in PROOFS.items():
            proof_started = now()
            run = subprocess.run([sys.executable, 'tests/webui/' + script], text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, env=dict(os.environ, PYTHONDONTWRITEBYTECODE='1'))
            log = output / f'{name}-run.log'
            log.write_text(run.stdout)
            if run.returncode:
                raise RuntimeError(f'{name} native proof failed; see {log}')
            development = Path('chainbench-out/web-ui-development') / name
            receipt = json.loads((development / 'receipt.json').read_text())
            if receipt.get('sourceDigest') != source or receipt.get('seedAcceptanceAwarded') is not False:
                raise RuntimeError(f'{name} receipt is not bound to this source')
            receipts[name] = receipt
            for file in ['receipt.json'] + files:
                shutil.copy2(development / file, output / f'{name}-{file}')
            evidence['proofs'].append({'name': name, 'startedAt': proof_started, 'runtime': receipt['runtime'], 'dashboardSHA256': receipt['dashboardSHA256'], 'browserVersion': receipt.get('browserVersion'), 'binaries': receipt.get('binaries')})
        observed = now()
        evidence['scenarios'] = scenarios(receipts, output)
        for scenario in evidence['scenarios']:
            scenario['invocationId'] = invocation
            scenario['observedAt'] = observed
            scenario['artifacts'] = [{'path': str(output / p), 'sha256': digest(output / p)} for p in scenario['artifacts']]
            if not all(a['passed'] for a in scenario['assertions']):
                failures.append('scenario failed: ' + scenario['id'])
        evidence['server'] = {'sha256': receipts['monitoring']['dashboardSHA256'], 'baseCommit': base_commit(), 'contractVersion': '2', 'goVersion': subprocess.check_output(['go', 'version'], text=True).strip()}
        evidence['frontend'] = [{'path': str(p), 'sha256': digest(p)} for p in sorted(Path('internal/dashboard/spa').rglob('*')) if p.is_file()]
        evidence['coverage'] = {'required': REQUIRED, 'executed': [s['id'] for s in evidence['scenarios']], 'missing': [], 'mockTargets': 0}
        evidence['source'] = [{'path': str(p), 'sha256': digest(p)} for pattern in ('internal/app/web_monitor*.go', 'internal/app/web_job_observation.go', 'internal/app/web_network_monitor.go', 'internal/app/web_history_observations.go', 'internal/dashboard/monitor.go', 'internal/dashboard/snapshot.go', 'internal/core/session/observationstore.go', 'web/src/MonitorPanel.svelte', 'web/src/NodeMetrics.svelte', 'web/src/metric-chart.mjs', 'web/src/job-observation.mjs', 'tests/webui/*web08*', 'tests/webui/browser_monitoring.mjs', 'tests/webui/browser_snapshot_restore.mjs', 'tests/webui/browser_ssh_log_collection.mjs', 'tests/webui/browser_logout_continuity.mjs') for p in sorted(Path('.').glob(pattern)) if p.is_file()]
        evidence['artifacts'] = [{'path': str(p), 'sha256': digest(p)} for p in sorted(output.iterdir()) if p.is_file() and p.name not in ('result.json', 'evidence.json')]
    except Exception as exc:
        failures.append(str(exc))
    evidence['blockers'] = failures
    ep = output / 'evidence.json'
    ep.write_text(json.dumps(evidence, indent=2) + '\n')
    executed = [s['id'] for s in evidence['scenarios']]
    result = {'criterion': 'WEB-08', 'invocationId': invocation, 'startedAt': started, 'finishedAt': now(),
              'outcome': 'pass' if not failures and sorted(executed) == sorted(REQUIRED) else 'incomplete',
              'requiredScenarios': REQUIRED, 'executedScenarios': executed,
              'skippedScenarios': [s for s in REQUIRED if s not in executed], 'failedAssertions': failures,
              'evidenceDigest': digest(ep)}
    rp = output / 'result.json'
    rp.write_text(json.dumps(result, indent=2) + '\n')
    errors = verify(output)
    if errors:
        result['outcome'] = 'incomplete'
        result['failedAssertions'] = failures + errors
        rp.write_text(json.dumps(result, indent=2) + '\n')
        print('\n'.join(failures + errors), file=sys.stderr)
        return 1
    return 0


if __name__ == '__main__':
    sys.exit(main())
