import subprocess,json,time,shutil,tempfile,urllib.request,socket
from pathlib import Path
out=Path('/Users/0xtopaz/work/github/0xmhha/chainbench/docs/dev/architecture/stablenet-boundary-audit')
binary='/Users/0xtopaz/work/github/0xmhha/chain/go-stablenet/build/bin/gstable'
tx='0x0413b32b1eba2f3aff315b76a5b8e99e9a2bed7b31a5d364ea5282765b4d69e8'
addr='0x000000000000000000000000000000000000dEaD'
results=[]
for i in range(1,6):
 d=Path(tempfile.mkdtemp(prefix='sba-',dir='/tmp'));shutil.copytree(f'/tmp/cbe2e3643186464/node{i}',d/'data')
 with socket.socket() as s:s.bind(('127.0.0.1',0));port=s.getsockname()[1]
 args=[binary,'--datadir',str(d/'data'),'--http','--http.addr','127.0.0.1','--http.port',str(port),'--http.api','eth,net,web3','--port','0','--maxpeers','0','--nodiscover','--ipcdisable','--authrpc.port','0']
 with (out/f'restored-node{i}.log').open('w') as log:
  p=subprocess.Popen(args,stdout=log,stderr=subprocess.STDOUT)
  def rpc(method,params=[]):
   data=json.dumps(dict(jsonrpc='2.0',id=1,method=method,params=params)).encode()
   with urllib.request.urlopen(urllib.request.Request(f'http://127.0.0.1:{port}',data,{'Content-Type':'application/json'}),timeout=5) as r:return json.load(r)
  try:
   for _ in range(60):
    try: head=rpc('eth_blockNumber');break
    except Exception:
     if p.poll() is not None:raise RuntimeError('node exited')
     time.sleep(.5)
   rec={'node':i,'argv':args,'head':head}
   for method,params,key in [('eth_getTransactionReceipt',[tx],'receipt'),('eth_getTransactionByHash',[tx],'transaction'),('eth_getBlockByNumber',['latest',False],'latest'),('eth_getBalance',[addr,'latest'],'balance_latest'),('eth_getBalance',[addr,'0x1'],'balance_block1'),('eth_syncing',[],'syncing')]:rec[key]=rpc(method,params)
   results.append(rec)
  except Exception as e:results.append({'node':i,'error':str(e)})
  finally:
   p.terminate()
   try:p.wait(timeout=15)
   except subprocess.TimeoutExpired:p.kill();p.wait()
 (out/'restored-state.json').write_text(json.dumps(results,indent=2)+'\n')
 print(i,results[-1].get('balance_latest',results[-1]),flush=True)
