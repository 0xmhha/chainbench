"""Development proof for native durable jobs; does not award any Seed criterion."""
import importlib.util
import json
import os
from pathlib import Path
import secrets
import socket
import subprocess
import sys
import time
import urllib.request
import uuid
from runtime_contract import runtime_root

def main():
    output=Path('chainbench-out/web-ui-development/native-jobs')
    output.mkdir(parents=True,exist_ok=True)
    runtime=runtime_root()/str(uuid.uuid4())
    runtime.mkdir(parents=True,mode=0o700)
    spec=importlib.util.spec_from_file_location('native_fixture','tests/webui/fixtures/prepare_web04.py')
    module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
    provenance=module.prepare(runtime,output)
    with (output/'build.log').open('w') as log:
        subprocess.run(['go','build','-o',str(runtime/'dashboard'),'./cmd/chainbench-dashboard'],stdout=log,stderr=log,check=True)
    with socket.socket() as listener:
        listener.bind(('127.0.0.1',0));port=listener.getsockname()[1]
    store=runtime/'store'
    log=(output/'server.log').open('w')
    command=[str(runtime/'dashboard'),'-addr','127.0.0.1:'+str(port),'-deployment-root',str(store),'-manifest-assets',str(runtime/'assets.json'),'-manifest-keys',str(Path('presets/keys').resolve())]
    server=subprocess.Popen(command,stdout=log,stderr=log)
    try:
        url='http://127.0.0.1:'+str(port)
        for _ in range(200):
            if server.poll() is not None:raise RuntimeError('dashboard exited; see server.log')
            try:
                urllib.request.urlopen(url+'/healthz',timeout=1).close();break
            except OSError:time.sleep(.1)
        else:raise RuntimeError('dashboard did not listen')
        fixture={'url':url,'setupToken':(store/'setup.token').read_text().strip(),'password':secrets.token_urlsafe(24),'runtime':str(runtime),'store':str(store)}
        private=runtime/'browser-fixture.json';private.write_text(json.dumps(fixture));private.chmod(0o600)
        result=subprocess.run(['node','tests/webui/browser_native_jobs.mjs',str(private),str(output.resolve())],text=True,capture_output=True,timeout=300)
        (output/'browser.log').write_text(result.stdout+result.stderr)
        if result.returncode:raise RuntimeError('native browser jobs failed; see browser.log')
        server.terminate();server.wait(timeout=10)
        server=subprocess.Popen(command,stdout=log,stderr=log)
        for _ in range(200):
            if server.poll() is not None:raise RuntimeError('dashboard restart failed')
            try:
                urllib.request.urlopen(url+'/healthz',timeout=1).close();break
            except OSError:time.sleep(.1)
        else:raise RuntimeError('dashboard did not restart')
        result=subprocess.run(['node','tests/webui/browser_history_restart.mjs',str(private),str(output.resolve())],text=True,capture_output=True,timeout=90)
        (output/'history-restart.log').write_text(result.stdout+result.stderr)
        if result.returncode:raise RuntimeError('history restart regression failed; see history-restart.log')
        for record in (store/'networks').glob('*/chain-record.json'):
            state=json.loads(record.read_text())
            for node in state['nodes']:
                data=Path(node['dataDir'])
                if len(list(data.glob('*/chaindata/CURRENT')))!=1:raise RuntimeError('missing initialized native database: '+str(data))
                dumped=subprocess.run([state['binary'],'--datadir',str(data),'dumpgenesis'],capture_output=True,text=True,check=True,timeout=20)
                genesis=json.loads(dumped.stdout)
                declared=json.loads(Path(state['genesisPath']).read_text())
                if genesis['config']['chainId']!=declared['config']['chainId']:raise RuntimeError('native database genesis differs from declared chain')
        receipt=json.loads((output/'browser.json').read_text())
        receipt.update({'nativeDatabaseCheck':'passed','historyRestart':json.loads((output/'history-restart.json').read_text()),'binaries':provenance,'runtime':str(runtime),'seedAcceptanceAwarded':False})
        (output/'receipt.json').write_text(json.dumps(receipt,indent=2))
        print('Native development proof PASS: three native chains, owned node start/stop, browser history comparison/export/delete and restart preservation.')
    finally:
        # Only PIDs from this exclusively owned fixture tree are eligible.
        for record in (store/'networks').glob('*/chain-record.json') if (store/'networks').exists() else []:
            for node in json.loads(record.read_text()).get('nodes',[]):
                pid=node.get('pid',0)
                if pid:
                    try:os.kill(pid,15)
                    except ProcessLookupError:pass
        server.terminate()
        try:server.wait(timeout=10)
        except subprocess.TimeoutExpired:server.kill();server.wait()
        log.close()

if __name__=='__main__':main()
