"""Prepare isolated native chain assets and record independent source provenance."""
from runtime_storage import clone_file
from fixtures.native_selection import selected_binaries
import json
from pathlib import Path
import subprocess
from evidence_web04 import digest

def prepare(runtime,output):
    assets=[]
    provenance=[]
    for selected in selected_binaries():
        chain=selected['chain']
        source=Path(selected['binary'])
        version=selected['version']
        commit=selected['embeddedCommit']
        config=selected['sourceConfig']
        destination=runtime/(chain+'-node');clone_file(source,destination)
        if digest(destination)!=selected['sha256']:
            raise RuntimeError('selected native binary changed while copying: '+chain)
        assets.append({'id':chain,'chain':chain,'path':str(destination),'sha256':digest(destination),'commit':commit})
        help_text=subprocess.check_output([str(destination),'--help'],text=True)
        (output/(chain+'-version.txt')).write_text(version)
        (output/(chain+'-help.txt')).write_text(help_text)
        provenance.append({'chain':chain,'sha256':digest(destination),'version':version,'commit':commit,'helpDigest':digest(output/(chain+'-help.txt')),'sourceConfigDigest':__import__('hashlib').sha256(config.encode()).hexdigest(),'identitySource':selected['identitySource'],'buildInfoSHA256':__import__('hashlib').sha256(selected['buildInfo'].encode()).hexdigest() if selected['buildInfo'] else None})
    wrong={**assets[2],'id':'wrong-wbft','chain':'wbft','commit':assets[2]['commit']}
    stale={**assets[1],'id':'stale-wbft','sha256':'0'*64}
    assets.extend([wrong,stale])
    (runtime/'assets.json').write_text(json.dumps(assets))
    return provenance
