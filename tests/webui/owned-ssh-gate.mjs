import fs from 'node:fs'
import assert from 'node:assert/strict'
import {execFileSync} from 'node:child_process'

// Identify the SSH connection holding this fixture's paused gate command by
// walking from the gate's own PID up to this fixture's sshd. Nothing outside
// that ancestry is ever signalled; a changed or foreign chain is refused.
export function gatedConnectionAncestors(f){
 const pid=Number(fs.readFileSync(f.runtime+'/ssh/gate-copied','utf8').trim().split(/\s+/)[0])
 assert.ok(Number.isSafeInteger(pid)&&pid>1)
 assert.ok(execFileSync('ps',['-p',String(pid),'-o','command='],{encoding:'utf8'}).includes(f.sshGatePath),'gate process is outside this fixture')
 const ancestors=[];let current=pid
 for(let i=0;i<16;i++){
  const text=execFileSync('ps',['-p',String(current),'-o','ppid=','-o','comm='],{encoding:'utf8'}).trim(),match=/^(\d+)\s+(.+)$/.exec(text)
  assert.ok(match,'owned SSH ancestry unavailable');ancestors.push({pid:current,parent:Number(match[1]),command:match[2]})
  if(current===f.ssh.pid)break
  current=Number(match[1]);assert.ok(current>1,'gated command does not descend from the fixture SSH daemon')
 }
 assert.equal(ancestors.at(-1).pid,f.ssh.pid)
 assert.ok(execFileSync('ps',['-p',String(f.ssh.pid),'-o','command='],{encoding:'utf8'}).includes(f.runtime+'/ssh/sshd_config'),'SSH root belongs to another task')
 return ancestors
}

export async function ownedGatedConnection(f){
 const first=gatedConnectionAncestors(f)
 await new Promise(r=>setTimeout(r,100))
 const second=gatedConnectionAncestors(f)
 assert.deepEqual(second,first,'owned SSH connection identity changed')
 const connection=first.slice(1,-1).find(v=>/(^|\/)sshd(?:-session|-auth)?(?:$|:)/.test(v.command))
 assert.ok(connection,'no owned SSH connection ancestor; refusing to signal anything')
 return connection
}
