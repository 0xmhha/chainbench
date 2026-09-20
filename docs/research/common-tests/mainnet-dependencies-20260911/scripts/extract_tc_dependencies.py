#!/usr/bin/env python3
"""Fresh extraction of per-test mainnet dependencies from tests/tc/**/*.json
(current working tree). Classifies every field of env and steps into the
dependency categories the task names: accounts, rpc, contracts, chain_id,
forks, fees, topology, binary. Values are kept (they are test fixtures, not
secrets) except anything that looks like a private key."""
import json, glob, os, re, hashlib, sys, collections

ROOT = '/Users/wm-it-25_0220/Work/github/chainbench'
OUT = sys.argv[1]
ADDR = re.compile(r'^0x[0-9a-fA-F]{40}$')
HEX32 = re.compile(r'^0x[0-9a-fA-F]{64}$')
ROLE = re.compile(r'^(node\d+|bp\d+|en\d+|pn\d+|faucet|producer|validator)$')
SYSTEM_CONTRACTS = {
    '0x0000000000000000000000000000000000001000': 'NativeCoinAdapter (stablenet)',
    '0x0000000000000000000000000000000000001001': 'GovValidator (stablenet)',
    '0x0000000000000000000000000000000000001002': 'GovMasterMinter (stablenet)',
    '0x0000000000000000000000000000000000001003': 'GovMinter (stablenet)',
    '0x0000000000000000000000000000000000001004': 'GovCouncil (stablenet)',
    '0x0000000000000000000000000000000000000100': 'P256VERIFY precompile (RIP-7212)',
}
NS_CHAIN = {'istanbul': 'wbft-family only (go-wbft, go-stablenet)', 'wemix': 'wemix namespace (go-wemix, go-wbft)',
            'eth': 'standard', 'net': 'standard', 'txpool': 'standard', 'admin': 'standard (admin, usually not public)',
            'web3': 'standard', 'debug': 'standard (debug, usually not public)'}
FEE_KEYS = {'gas', 'gasPrice', 'maxFeePerGas', 'maxPriorityFeePerGas', 'value', 'fillPercent', 'amount'}
ACCOUNT_KEYS = {'from', 'feePayerKey', 'senderKey', 'authorityKey', 'deployer', 'delegate'}
FAULT_VERBS = {'stopNode', 'startNode', 'restartNode', 'swapNode', 'partition', 'healPartition', 'readNodeLog', 'load'}
FORK_VERBS = {'sendSetCode', 'signAuthorization'}

def vkind(v):
    if isinstance(v, str):
        if v.startswith('$'): return 'binding'
        if ROLE.match(v): return 'role'
        if ADDR.match(v): return 'literal-address'
        if HEX32.match(v): return 'literal-hex32'
        if re.match(r'^0x[0-9a-fA-F]+$', v): return 'literal-hex'
        if re.match(r'^-?\d+$', v): return 'literal-number'
        return 'literal-string'
    if isinstance(v, bool): return 'literal-bool'
    if isinstance(v, (int, float)): return 'literal-number'
    if isinstance(v, list): return 'list'
    if isinstance(v, dict): return 'object'
    return 'null'

def walk(obj, path):
    if isinstance(obj, dict):
        for k, v in obj.items():
            yield from walk(v, f'{path}.{k}' if path else k)
    elif isinstance(obj, list):
        for i, v in enumerate(obj):
            yield from walk(v, f'{path}[{i}]')
    else:
        yield path, obj

def analyze(f):
    raw = open(f, 'rb').read()
    d = json.loads(raw)
    rel = os.path.relpath(f, ROOT)
    rec = {'file': rel, 'sha256': hashlib.sha256(raw).hexdigest(), 'id': d.get('id'), 'description': d.get('description', ''),
           'applicableChains': d.get('applicableChains'), 'requires': d.get('requires', []),
           'source_dir_chain': rel.split('/')[2] if rel.startswith('tests/tc/go-') else 'shared'}
    env = d.get('env')
    rec['env'] = {'chain': env.get('chain'), 'binaries': env.get('binaries'), 'topology': env.get('topology'),
                  'keys': env.get('keys'), 'capabilities': env.get('capabilities'), 'genesis': env.get('genesis'),
                  'hardforks': env.get('hardforks'), 'launch': env.get('launch'), 'config': env.get('config'),
                  'accounts': env.get('accounts'), 'upgrade': env.get('upgrade')} if isinstance(env, dict) else {'ref': env}
    deps = collections.defaultdict(list)
    def add(cat, field, value, meaning):
        k = vkind(value)
        if k == 'literal-hex32' and 'Key' in field:
            value = '<redacted-32-byte-hex>'
        deps[cat].append({'field': field, 'value_kind': k, 'value': value if not isinstance(value, (dict, list)) else json.dumps(value, ensure_ascii=False)[:200], 'meaning': meaning})
    # env-level
    if isinstance(env, dict):
        add('chain_id', 'env.chain', env.get('chain'), 'client family selection (fixes manifest chain_id/network_id defaults)')
        for k, v in (env.get('binaries') or {}).items(): add('binary', f'env.binaries.{k}', v, 'binary name/path resolved by the composer')
        for p, v in walk(env.get('topology') or {}, 'env.topology'): add('topology', p, v, 'node count/role layout')
        for p, v in walk(env.get('keys') or {}, 'env.keys'): add('accounts', p, v, 'node identity key source (validators/producers and node-signed sender)')
        for p, v in walk(env.get('accounts') or {}, 'env.accounts'): add('accounts', p, v, 'declared test account funded at run time')
        for p, v in walk(env.get('hardforks') or {}, 'env.hardforks'): add('forks', p, v, 'fork activation height override')
        g = env.get('genesis') or {}
        for p, v in walk(g, 'env.genesis'):
            cat = 'forks' if re.search(r'(Block|Time|fork|anzeon|boho|croissant|brioche|pangyo|applepie|istanbul)', p, re.I) else \
                  'chain_id' if 'chainId' in p else 'contracts' if re.search(r'alloc|0x0{30,}', p) else 'topology'
            add(cat, p, v, 'genesis overlay/set value')
        for p, v in walk(env.get('launch') or {}, 'env.launch'): add('topology', p, v, 'node launch flag')
        for p, v in walk(env.get('config') or {}, 'env.config'): add('topology', p, v, 'node config knob')
        for p, v in walk(env.get('upgrade') or {}, 'env.upgrade'): add('binary', p, v, 'mixed-binary handoff declaration')
        for c in env.get('capabilities') or []: add('rpc', 'env.capabilities[]', c, 'capability the env claims to provide')
    for c in d.get('requires', []): add('rpc', 'requires[]', c, 'capability the case needs (rpc/ws/process/consensus)')
    if d.get('applicableChains'): add('chain_id', 'applicableChains', d['applicableChains'], 'chain allowlist filter (not a support proof)')
    # steps
    verbs = collections.Counter(); methods = collections.Counter()
    for i, s in enumerate(d.get('steps', [])):
        pre = f'steps[{i}]'
        verb = s.get('do') or ('expect:' + str(s.get('expect')))
        verbs[verb] += 1
        if s.get('do') in FAULT_VERBS: add('topology', f'{pre}.do', s['do'], 'process/peer control (needs owned nodes)')
        if s.get('do') in FORK_VERBS: add('forks', f'{pre}.do', s['do'], 'EIP-7702 set-code path (fork gated; unsupported on go-wemix)')
        if s.get('do') == 'swapNode' and 'binary' in s: add('binary', f'{pre}.binary', s['binary'], 'binary swapped into a node')
        if s.get('do') in ('deployContract', 'registerContract', 'faucet'): add('contracts' if s['do'] != 'faucet' else 'accounts', f'{pre}.do', s['do'], 'node-signed asset helper')
        if s.get('expect') in ('chainId',): add('chain_id', f'{pre}.is', s.get('is'), 'expected eth_chainId value')
        if s.get('expect') in ('baseFee', 'gasPrice', 'estimateGas') or s.get('source') in ('baseFee', 'gasPrice'):
            add('fees', f'{pre}.{"expect" if "expect" in s else "source"}', s.get('expect') or s.get('source'), 'fee observation whose expected value is chain-policy dependent')
        if s.get('expect') in ('metric',): add('rpc', f'{pre}.expect', 'metric', 'Prometheus metrics endpoint (launch --metrics)')
        if s.get('expect') in ('wsSubscribe', 'wsCollected') or s.get('do') == 'wsOpen': add('rpc', f'{pre}.{"do" if "do" in s else "expect"}', s.get('do') or s.get('expect'), 'WebSocket endpoint')
        if s.get('expect') in ('peerCount',): add('topology', f'{pre}.expect', 'peerCount', 'peer graph shape')
        for k, v in s.items():
            fld = f'{pre}.{k}'
            if k == 'method':
                methods[v] += 1
                ns = v.split('_')[0]
                cat = 'forks' if ns in ('istanbul', 'wemix') else 'rpc'
                add(cat, fld, v, f'RPC method; namespace {ns}: {NS_CHAIN.get(ns, "?")}')
            elif k in ('on', 'onEach'):
                for x in (v if isinstance(v, list) else [v]): add('rpc', fld, x, 'target node role (endpoint selection)')
            elif k in ACCOUNT_KEYS:
                add('accounts', fld, v, 'signer/account reference' + (' (hardcoded address literal)' if vkind(v) == 'literal-address' else ''))
            elif k == 'to':
                kind = vkind(v)
                if kind == 'literal-address':
                    if v.lower() in SYSTEM_CONTRACTS: add('contracts', fld, v, 'SYSTEM CONTRACT: ' + SYSTEM_CONTRACTS[v.lower()])
                    elif re.match(r'^0x0{28,}', v): add('contracts', fld, v, 'reserved low address (precompile/system range) literal')
                    else: add('accounts', fld, v, 'hardcoded recipient address literal')
                elif kind == 'binding': add('contracts', fld, v, 'address bound from an earlier step (deployed contract or created account)')
                else: add('accounts', fld, v, 'recipient by role/label')
            elif k == 'address':
                kind = vkind(v)
                if kind == 'literal-address' and v.lower() in SYSTEM_CONTRACTS: add('contracts', fld, v, 'SYSTEM CONTRACT: ' + SYSTEM_CONTRACTS[v.lower()])
                elif kind == 'literal-address': add('accounts', fld, v, 'hardcoded address literal')
                else: add('accounts', fld, v, 'address by role/label/binding')
            elif k == 'params' and isinstance(v, list):
                for j, p in enumerate(v):
                    pk = vkind(p)
                    if pk == 'literal-address':
                        if p.lower() in SYSTEM_CONTRACTS: add('contracts', f'{fld}[{j}]', p, 'SYSTEM CONTRACT: ' + SYSTEM_CONTRACTS[p.lower()])
                        else: add('accounts', f'{fld}[{j}]', p, 'hardcoded address literal in RPC params')
                    elif isinstance(p, dict):
                        for pp, pv in walk(p, f'{fld}[{j}]'):
                            if vkind(pv) == 'literal-address':
                                add('contracts' if pv.lower() in SYSTEM_CONTRACTS else 'accounts', pp, pv, ('SYSTEM CONTRACT: ' + SYSTEM_CONTRACTS[pv.lower()]) if pv.lower() in SYSTEM_CONTRACTS else 'hardcoded address literal in RPC params')
                            elif pp.endswith('.from') or pp.endswith('.to'): add('accounts', pp, pv, 'account reference in RPC params')
                            elif re.search(r'gas|fee|value', pp, re.I): add('fees', pp, pv, 'fee/gas input in RPC params')
            elif k in FEE_KEYS:
                add('fees', fld, v, 'fee/gas/value input literal' if vkind(v).startswith('literal') else 'fee/gas/value bound from a step')
            elif k in ('data', 'bytecode', 'selector'):
                add('contracts', fld, (v[:24] + '…') if isinstance(v, str) and len(v) > 24 else v, 'calldata/bytecode fixture (EVM revision dependent)')
            elif k in ('accessList',): add('fees', fld, json.dumps(v)[:120], 'EIP-2930 access list input')
            elif k in ('timeout', 'pollInterval', 'target', 'blocks', 'wait', 'within', 'maxAdvance', 'maxSeconds', 'maxBytes', 'count'):
                add('topology', fld, v, 'timing/block-count constant (block period dependent)')
            elif k == 'reason': add('fees', fld, v, 'expected rejection reason (client error string; may differ per client)')
            elif k == 'genesisOverlay': add('forks', fld, json.dumps(v)[:200], 'per-step genesis overlay (swapNode)')
    rec['verbs'] = dict(verbs); rec['methods'] = dict(methods)
    rec['dependencies'] = {c: deps.get(c, []) for c in ('accounts', 'rpc', 'contracts', 'chain_id', 'forks', 'fees', 'topology', 'binary')}
    rec['dependency_counts'] = {c: len(rec['dependencies'][c]) for c in rec['dependencies']}
    rec['flags'] = {
        'hardcoded_address_literals': sorted({x['value'] for c in ('accounts',) for x in rec['dependencies'][c] if x['value_kind'] == 'literal-address'}),
        'system_contracts': sorted({x['value'] for x in rec['dependencies']['contracts'] if str(x['meaning']).startswith('SYSTEM CONTRACT')}),
        'consensus_rpc_namespaces': sorted({m.split('_')[0] for m in methods if m.split('_')[0] in ('istanbul', 'wemix')}),
        'uses_ws': any(v in ('wsOpen', 'expect:wsSubscribe', 'expect:wsCollected') for v in verbs),
        'uses_process_control': any(v in FAULT_VERBS for v in verbs),
        'uses_setcode': any(v in FORK_VERBS for v in verbs),
        'uses_node_signed_helpers': any(v in ('deployContract', 'registerContract', 'faucet', 'load') for v in verbs),
        'expects_chain_id': any(v == 'expect:chainId' for v in verbs),
        'fee_expectation': any(v in ('expect:baseFee', 'expect:gasPrice', 'expect:estimateGas') or x for v in verbs for x in [False]) or any(s.get('source') in ('baseFee', 'gasPrice') for s in d.get('steps', [])),
        'genesis_customized': bool(isinstance(env, dict) and (env.get('genesis') or env.get('hardforks'))),
        'hooks': bool(d.get('hooks')),
    }
    return rec

files = sorted(glob.glob(os.path.join(ROOT, 'tests/tc/**/*.json'), recursive=True))
recs = [analyze(f) for f in files]
json.dump(recs, open(os.path.join(OUT, 'analyses', 'tc-dependencies.json'), 'w'), indent=1, ensure_ascii=False)
print('files', len(recs))
agg = collections.Counter()
for r in recs:
    for c, n in r['dependency_counts'].items(): agg[c] += n
print('dependency rows', dict(agg))
print('hardcoded addr literal files', sum(1 for r in recs if r['flags']['hardcoded_address_literals']))
print('system contract files', sum(1 for r in recs if r['flags']['system_contracts']))
print('consensus ns files', collections.Counter(tuple(r['flags']['consensus_rpc_namespaces']) for r in recs))
print('process control', sum(1 for r in recs if r['flags']['uses_process_control']), 'ws', sum(1 for r in recs if r['flags']['uses_ws']), 'setcode', sum(1 for r in recs if r['flags']['uses_setcode']), 'node-signed helpers', sum(1 for r in recs if r['flags']['uses_node_signed_helpers']), 'genesis custom', sum(1 for r in recs if r['flags']['genesis_customized']))
print('env.chain', collections.Counter(r['env'].get('chain') for r in recs))
print('applicableChains', collections.Counter(r['applicableChains'] for r in recs))
print('requires', collections.Counter(tuple(r['requires']) for r in recs))
addrs = collections.Counter(a for r in recs for a in r['flags']['hardcoded_address_literals'])
print('addr literals', addrs.most_common(20))
