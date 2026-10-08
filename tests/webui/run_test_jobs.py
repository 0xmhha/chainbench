"""Real browser/engine test job proof; does not award a Seed criterion."""
import importlib.util
import json
import os
from pathlib import Path
import secrets
import shutil
import socket
import subprocess
import time
import urllib.request
import uuid
from runtime_contract import runtime_root
from browser_process import run_browser

def main():
    output=Path('chainbench-out/web-ui-development/test-jobs')
    output.mkdir(parents=True,exist_ok=True)
    runtime=runtime_root()/str(uuid.uuid4())
    runtime.mkdir(parents=True,mode=0o700)
    spec=importlib.util.spec_from_file_location('native_fixture','tests/webui/fixtures/prepare_web04.py')
    module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
    provenance=module.prepare(runtime,output)
    source_keys=runtime/'source-keys'
    shutil.copytree('presets/keys',source_keys)
    with (output/'build.log').open('w') as log:
        subprocess.run(['go','build','-o',str(runtime/'dashboard'),'./cmd/chainbench-dashboard'],stdout=log,stderr=log,check=True)
    with socket.socket() as listener:
        listener.bind(('127.0.0.1',0));port=listener.getsockname()[1]
    store=runtime/'store'
    log=(output/'server.log').open('w')
    command=[str(runtime/'dashboard'),'-addr','127.0.0.1:'+str(port),'-deployment-root',str(store),'-manifest-assets',str(runtime/'assets.json'),'-manifest-keys',str(source_keys)]
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
        result=run_browser(['node','tests/webui/browser_test_jobs.mjs',str(private),str(output.resolve())],timeout=300)
        (output/'browser.log').write_text(result.stdout+result.stderr)
        if result.returncode:raise RuntimeError('test browser jobs failed:\n'+result.stdout+result.stderr)
        receipt=json.loads((output/'browser.json').read_text())
        receipt.update({'binaries':provenance,'runtime':str(runtime),'seedAcceptanceAwarded':False})
        (output/'receipt.json').write_text(json.dumps(receipt,indent=2))
        print('Test job development proof PASS: real DSL verdicts, sessions, history and accepted case revisions.')
    finally:
        # Only PIDs from this exclusively owned fixture tree are eligible.
        for record in (store/'networks').glob('*/chain-record.json') if (store/'networks').exists() else []:
            for node in json.loads(record.read_text()).get('nodes',[]):
                pid=node.get('pid',0)
                if pid:
                    command_line=subprocess.run(['ps','-p',str(pid),'-o','command='],capture_output=True,text=True)
                    if str(runtime) in command_line.stdout:
                        try:os.kill(pid,15)
                        except ProcessLookupError:pass
        server.terminate()
        try:server.wait(timeout=10)
        except subprocess.TimeoutExpired:server.kill();server.wait()
        log.close()

if __name__=='__main__':main()
