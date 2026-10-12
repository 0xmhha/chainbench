"""Prepare isolated native chain assets and record independent source provenance."""
import glob
import json
from pathlib import Path
import re
import shutil
import subprocess
from evidence_web04 import digest

def prepare(runtime,output):
    assets=[]
    provenance=[]
    for chain in ('stablenet','wbft','wemix'):
        matches=glob.glob('/Users/0xtopaz/work/github/0xmhha/chain/go-'+chain+'/build/bin/'+('gstable' if chain=='stablenet' else 'gwemix'))
        if len(matches)!=1:
            raise RuntimeError('native chain binary not found: '+chain)
        source=Path(matches[0]);repository=source.parents[2]
        version=subprocess.check_output([str(source),'version'],text=True)
        commit=re.search(r'Git Commit: ([0-9a-f]{40})',version).group(1)
        # Confirm identity from the source repository object, not the executable basename.
        subprocess.check_call(['git','-C',str(repository),'cat-file','-e',commit+'^{commit}'])
        config=subprocess.check_output(['git','-C',str(repository),'show',commit+':params/config.go'],text=True)
        marker={'stablenet':'Anzeon','wbft':'Croissant','wemix':'Wemix'}[chain]
        if marker not in config:
            raise RuntimeError('binary source commit does not carry expected chain engine: '+chain)
        destination=runtime/(chain+'-node');shutil.copy2(source,destination)
        assets.append({'id':chain,'chain':chain,'path':str(destination),'sha256':digest(destination),'commit':commit})
        help_text=subprocess.check_output([str(destination),'--help'],text=True)
        (output/(chain+'-version.txt')).write_text(version)
        (output/(chain+'-help.txt')).write_text(help_text)
        provenance.append({'chain':chain,'sha256':digest(destination),'version':version,'commit':commit,'helpDigest':digest(output/(chain+'-help.txt')),'sourceConfigDigest':__import__('hashlib').sha256(config.encode()).hexdigest()})
    wrong={**assets[2],'id':'wrong-wbft','chain':'wbft','commit':assets[2]['commit']}
    stale={**assets[1],'id':'stale-wbft','sha256':'0'*64}
    assets.extend([wrong,stale])
    (runtime/'assets.json').write_text(json.dumps(assets))
    return provenance
