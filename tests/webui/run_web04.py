"""WEB-04 fixture runner; writes incomplete receipts on any missing live evidence."""
from runtime_contract import runtime_root, base_commit

import datetime
import importlib.util
import os
import secrets
import socket
import time
import urllib.request
import shutil
import json
from pathlib import Path
import subprocess
import sys
import uuid
from evidence_web04 import REQUIRED, digest, verify

def now():
    return datetime.datetime.now(datetime.timezone.utc).isoformat()

def output_directories(requested):
    """Keep canonical evidence in the workspace and publish a requested copy."""
    output = Path('chainbench-out/web-ui-acceptance/WEB-04')
    if not output.resolve().is_relative_to(Path.cwd().resolve()):
        raise ValueError('workspace evidence directory must stay inside the workspace')
    return output, (Path(requested) / 'WEB-04').resolve()

def publish_output(output, destination):
    if output.resolve() != destination:
        destination.mkdir(parents=True, exist_ok=True)
        for artifact in output.iterdir():
            if artifact.is_file():
                shutil.copy2(artifact, destination / artifact.name)

def main():
    out, destination = output_directories(sys.argv[1])
    out.mkdir(parents=True, exist_ok=True)
    invocation = str(uuid.uuid4())
    started = now()
    for old in out.iterdir():
        if old.is_file(): old.unlink()
    runtime = runtime_root() / invocation
    runtime.mkdir(parents=True,mode=0o700)
    server = None
    handles = []
    evidence = {'criterion': 'WEB-04', 'invocationId': invocation, 'scenarios': [], 'artifacts': []}
    failures = []
    try:
        for command in ('npm --prefix web run build', 'go test ./internal/chains/external', 'go test ./internal/app ./internal/dashboard -run Manifest'):
            run = subprocess.run(command,shell=True,text=True,capture_output=True)
            log = 'build.log' if command.startswith('npm') else ('external-tests.log' if command.endswith('/external') else 'tests.log')
            (out / log).write_text(run.stdout+run.stderr)
            if run.returncode: raise RuntimeError(command+' failed')
        subprocess.run(['bash','tests/webui/prepare.sh',invocation],check=True,capture_output=True)
        spec=importlib.util.spec_from_file_location('fixture','tests/webui/fixtures/prepare_web04.py')
        fixture=importlib.util.module_from_spec(spec);spec.loader.exec_module(fixture)
        evidence['binaries']=fixture.prepare(runtime,out)
        account={'id':'manifest-operator','username':'manifest-operator','role':'operator','password':secrets.token_urlsafe(24)}
        (runtime/'accounts.go').write_bytes(Path('tests/webui/fixtures/accounts.go.txt').read_bytes())
        provision=subprocess.run(['go','run',str(runtime/'accounts.go')],input=json.dumps([account]),text=True,capture_output=True,check=True)
        (runtime/'accounts.json').write_text(provision.stdout);(runtime/'accounts.json').chmod(0o600)
        store=runtime/'store'
        (runtime/'browser-fixture.json').write_text(json.dumps({'account':account,'store':str(store)}));(runtime/'browser-fixture.json').chmod(0o600)
        with socket.socket() as sock:
            sock.bind(('127.0.0.1',0));port=sock.getsockname()[1]
        url='http://127.0.0.1:'+str(port)
        logfile=(out/'server.log').open('w');handles.append(logfile)
        command=[str(runtime/'chainbench-dashboard'),'-addr','127.0.0.1:'+str(port),'-deployment-accounts',str(runtime/'accounts.json'),'-deployment-root',str(store),'-manifest-assets',str(runtime/'assets.json')]
        def launch():
            process=subprocess.Popen(command,stdout=logfile,stderr=logfile)
            for attempt in range(100):
                if process.poll() is not None: raise RuntimeError('dashboard exited')
                try:
                    urllib.request.urlopen(url+'/healthz',timeout=1).close();return process
                except OSError: time.sleep(.1)
            raise RuntimeError('dashboard health timeout')
        server=launch()
        browser=subprocess.run(['node','tests/webui/browser_web04.mjs',url,str(out),str(runtime/'browser-fixture.json')],text=True,capture_output=True)
        (out/'browser.log').write_text(browser.stdout+browser.stderr)
        if browser.returncode: raise RuntimeError('live browser assertions failed; see browser.log')
        observations=json.loads((out/'browser.json').read_text())
        for setup in observations['setups']:
            directory=Path(setup['dir'])
            states=list(directory.rglob('chain-record.json'))
            if not states: states=list(directory.rglob('state.json'))
            if len(states)!=1: raise RuntimeError('authoritative chain state not found: '+str(directory))
            state=json.loads(states[0].read_text());setup['state']=state
            if state['chain']!=setup['chain']: raise RuntimeError('engine applied wrong manifest')
            databases=[p for p in directory.rglob('*') if p.is_file() and p.name.startswith('MANIFEST-') and p.parent.name=='chaindata']
            if not databases: raise RuntimeError('real init did not produce database: '+str(directory))
            setup['databaseFiles']=[]
            for index,p in enumerate(databases):
                artifact=out/(setup['source']+'-'+setup['chain'].removeprefix('external-')+'-db-'+str(index)+'.txt')
                shutil.copy2(p,artifact)
                setup['databaseFiles'].append({'path':str(artifact),'sha256':digest(artifact),'node':int(p.relative_to(directory).parts[0].removeprefix('node'))})
            genesis=out/(setup['source']+'-'+setup['chain'].removeprefix('external-')+'-genesis.json')
            shutil.copy2(directory/'genesis.json',genesis)
            setup['genesis']={'path':str(genesis),'sha256':digest(genesis)}
            shutil.copy2(states[0],out/(setup['source']+'-'+setup['chain'].removeprefix('external-')+'-state.json'))
            setup['targetFingerprint']=__import__('hashlib').sha256(str(directory.resolve()).encode()).hexdigest()
        # Independent restart observation proves records were loaded from persistence.
        server.terminate();server.wait(timeout=10);server=launch()
        authorization=__import__('base64').b64encode((account['username']+':'+account['password']).encode()).decode()
        req=urllib.request.Request(url+'/api/v1/manifests',headers={'Authorization':'Basic '+authorization})
        restored=json.load(urllib.request.urlopen(req,timeout=10))['items']
        if len(restored)!=6: raise RuntimeError('external manifests not restored after restart')
        audit=json.loads('[]')
        for line in (store/'manifest-audit.jsonl').read_text().splitlines(): audit.append(json.loads(line))
        if sum(a['status']==200 and a['operation'].endswith('/setup') for a in audit)!=6: raise RuntimeError('missing setup audit receipts')
        if sum(a['status']==422 for a in audit)<6: raise RuntimeError('missing rejection audit receipts')
        (out/'audit.json').write_text(json.dumps(audit,indent=2))
        evidence['scenarios']=observations['scenarios']
        (out/'observations.json').write_text(json.dumps(observations,indent=2))
        evidence['server']={'sha256':digest(runtime/'chainbench-dashboard'),'baseCommit':base_commit(),'contractVersion':'2','url':url}
        evidence['frontend']=[{'path':str(p),'sha256':digest(p)} for p in Path('internal/dashboard/spa').rglob('*') if p.is_file()]
        evidence['fixture']={'runtime':str(runtime),'transport':'local','ownership':'owned','browserVersion':observations['browserVersion'],'mockTargets':0}
        evidence['artifacts']=[{'path':str(p),'sha256':digest(p)} for p in sorted(out.iterdir()) if p.is_file()]
    except Exception as exc:
        failures.append(str(exc))
    finally:
        if server is not None:
            server.terminate()
            try: server.wait(timeout=10)
            except subprocess.TimeoutExpired: server.kill();server.wait()
        for handle in handles: handle.close()
        for name in ('browser-fixture.json','accounts.json'):
            (runtime/name).unlink(missing_ok=True)
    ep = out / 'evidence.json' 
    ep.write_text(json.dumps(evidence, indent=2) + '\n')
    (out / 'result.json').write_text(json.dumps({'criterion': 'WEB-04', 'invocationId': invocation, 'startedAt': started, 'finishedAt': now(), 'outcome': 'incomplete' if failures else 'pass', 'requiredScenarios': REQUIRED, 'executedScenarios': [s['id'] for s in evidence['scenarios']], 'skippedScenarios': [s for s in REQUIRED if s not in [item['id'] for item in evidence['scenarios']]], 'failedAssertions': failures, 'evidenceDigest': digest(ep)}, indent=2) + '\n')

    if not failures:
        try: verify(out)
        except Exception as exc:
            result=json.loads((out/'result.json').read_text())
            result['outcome']='incomplete';result['failedAssertions']=[str(exc)]
            (out/'result.json').write_text(json.dumps(result,indent=2)+'\n')
    publish_output(out, destination)

if __name__ == '__main__':
    main()
