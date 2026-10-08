import test from 'node:test'
import assert from 'node:assert/strict'
import {snapshotState, applyJobChange} from '../../web/src/job-observation.mjs'
const cursor = v => 'a'.repeat(32)+':'+v
const snapshot = (v=4) => ({version:v,cursor:cursor(v),jobs:[{id:'one',state:'running'}],networks:[],observedAt:'2026-10-09T00:00:00Z'})
test('a missing committed version never turns an old job into a current result',()=>{
 const state=snapshotState(null,snapshot())
 const gap=applyJobChange(state,{type:'job_changed',version:6,cursor:cursor(6),job:{id:'one',state:'succeeded'}})
 assert.equal(gap.resync,true);assert.equal(gap.reason,'version_gap');assert.equal(gap.state.jobs[0].state,'running');assert.equal(gap.state.cursor,cursor(4))
 const restored=snapshotState(gap.state,snapshot(6))
 assert.equal(restored.version,6)
})
test('consecutive public changes restore jobs and old replies cannot roll back state',()=>{
 const state=snapshotState(null,snapshot())
 const changed=applyJobChange(state,{type:'job_changed',version:5,cursor:cursor(5),job:{id:'one',state:'succeeded'}})
 assert.equal(changed.resync,false);assert.equal(changed.state.jobs[0].state,'succeeded')
 assert.equal(snapshotState(changed.state,snapshot(4)),changed.state)
 assert.equal(applyJobChange(changed.state,{type:'job_changed',version:4,cursor:cursor(4),job:{id:'one',state:'failed'}}).state,changed.state)
})
test('heartbeats detect gaps without acknowledging undelivered changes',()=>{
 const state=snapshotState(null,snapshot())
 assert.equal(applyJobChange(state,{type:'observation',version:4,cursor:cursor(4)}).state.cursor,cursor(4))
 const missed=applyJobChange(state,{type:'observation',version:5,cursor:cursor(5)})
 assert.equal(missed.resync,true);assert.equal(missed.state.cursor,cursor(4))
})
test('server generations and explicit resync require a new snapshot',()=>{
 const state=snapshotState(null,snapshot())
 assert.equal(applyJobChange(state,{type:'version_changed',version:5,cursor:'b'.repeat(32)+':5'}).reason,'server_restarted')
 assert.equal(applyJobChange(state,{type:'resync_required',version:4,cursor:cursor(4),reason:'cursor_expired'}).reason,'cursor_expired')
 assert.equal(snapshotState(state,{...snapshot(),cursor:'b'.repeat(32)+':4'}).cursor,'b'.repeat(32)+':4')
})
test('malformed or inconsistent positions never become a trusted snapshot',()=>{
 for(const value of [{...snapshot(),cursor:cursor(5)},{...snapshot(),version:Infinity},{...snapshot(),jobs:null},{...snapshot(),networks:null}])assert.throws(()=>snapshotState(null,value))
 const state=snapshotState(null,snapshot())
 assert.equal(applyJobChange(state,{type:'job_changed',version:5,cursor:cursor(5)}).resync,true)
})
