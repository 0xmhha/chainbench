import os, sys, json, hashlib, subprocess, time, signal
from pathlib import Path
from datetime import datetime, timezone

root=Path('/Users/0xtopaz/work/github/0xmhha/chainbench')
stamp=datetime.now(timezone.utc).strftime('%Y%m%dT%H%M%SZ')
out=root/'docs/dev/architecture/baselines'/stamp
out.mkdir(parents=True, exist_ok=False)
Path('/tmp/chainbench-baseline-path').write_text(str(out))
env=os.environ.copy()
env['GOCACHE']='/tmp/chainbench-baseline-gocache'
env['GOTOOLCHAIN']='local'
env['GOROOT']='/Users/0xtopaz/.gvm/pkgsets/go1.25.2/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.25.13.darwin-arm64'
env['PATH']=env['GOROOT']+'/bin:'+env['PATH']
env['GOPROXY']='off'
env['GOSUMDB']='off'
tmp=Path('/tmp')/('cbb-'+stamp)
tmp.mkdir()
env['TMPDIR']=str(tmp)
results=[]
def run(name,args,timeout=600,extra=None):
    e=env.copy(); e.update(extra or {})
    start=time.time()
    with (out/(name+'.log')).open('w') as f:
        p=subprocess.Popen(args,cwd=root,env=e,stdout=f,stderr=subprocess.STDOUT,start_new_session=True)
        try: rc=p.wait(timeout=timeout); status='PASS' if rc==0 else 'FAIL'
        except subprocess.TimeoutExpired:
            os.killpg(p.pid,signal.SIGTERM)
            try: p.wait(timeout=10)
            except subprocess.TimeoutExpired: os.killpg(p.pid,signal.SIGKILL); p.wait()
            rc=p.returncode; status='TIMEOUT'
    r=dict(name=name,argv=args,cwd=str(root),environment_overrides={k:e[k] for k in ['GOCACHE','GOTOOLCHAIN','GOPROXY','GOSUMDB','TMPDIR']},extra_environment=extra or {},timeout_seconds=timeout,started_at=datetime.fromtimestamp(start,timezone.utc).isoformat(),elapsed_seconds=round(time.time()-start,2),exit_code=rc,status=status,log=name+'.log')
    results.append(r)
    (out/'commands.json').write_text(json.dumps(results,ensure_ascii=False,indent=2)+'\n')
    print(name,status,r['elapsed_seconds'],flush=True)
    return rc

repos=[root]+[root.parent/'chain'/n for n in ['go-wemix','go-wbft','go-stablenet']]
snap=[]
for repo in repos:
    def git(*a): return subprocess.check_output(['git','-C',str(repo),*a])
    files=git('ls-files','-z').decode().split('\0')
    hashes={f:hashlib.sha256((repo/f).read_bytes()).hexdigest() if (repo/f).is_file() else None for f in files if f}
    snap.append(dict(root=str(repo),commit=git('rev-parse','HEAD').decode().strip(),status=git('status','--porcelain').decode(),tracked_content_sha256=hashlib.sha256(json.dumps(hashes,sort_keys=True).encode()).hexdigest(),tracked_files=hashes))
(out/'source-snapshots.json').write_text(json.dumps(snap,indent=2)+'\n')
bins={n:str(root.parent/'chain'/n/'build/bin'/b) for n,b in [('go-wemix','gwemix'),('go-wbft','gwemix'),('go-stablenet','gstable')]}
(out/'binaries.json').write_text(json.dumps({n:dict(path=b,sha256=hashlib.sha256(Path(b).read_bytes()).hexdigest()) for n,b in bins.items()},indent=2)+'\n')
for n,b in bins.items():
    run(n+'-version',[b,'version'],30)
    run(n+'-build-info',['go','version','-m',b],30)
run('go-environment',['go','env','GOVERSION','GOOS','GOARCH','CGO_ENABLED','GOMOD'],30)
run('unit',['go','test','-count=1','-json','-timeout=8m','./...'],900)
if run('build-cli',['go','build','-o',str(tmp/'chainbench'),'./cmd/chainbench'],300)==0:
    for name,args in [('help',['--help']),('chains',['chains']),('capabilities',['capabilities'])]:
        run('cli-'+name,[str(tmp/'chainbench'),*args],30)
extra={'WBFT_BIN':bins['go-wbft'],'GSTABLE_BIN':bins['go-stablenet'],'WEMIX_BIN':bins['go-wemix'],'CHAINBENCH':str(tmp/'chainbench')}
for name,test in [('wbft','TestE2E_WbftChain'),('stablenet','TestE2E_StablenetChain')]:
    run('e2e-'+name,['go','test','-count=1','-json','-tags','e2e','-timeout=6m','-run','^'+test+'$','./tests/e2e/'],420,extra)
extra={'CHAINBENCH_E2E_FROM_BIN':bins['go-wemix'],'CHAINBENCH_E2E_TO_BIN':bins['go-wbft'],'CHAINBENCH_E2E_TEMPLATE':str(root.parent/'chain/go-wemix/wemix/scripts/genesis-template.json')}
run('e2e-wemix-handoff',['go','test','-count=1','-json','-tags','e2e','-timeout=6m','-run','^TestUpgradeRunE2E$','./cmd/chainbench/'],420,extra)
print('ARTIFACT',out,flush=True)
