import assert from 'node:assert/strict'
import {launchOwnedBrowser} from './owned-browser.mjs'
import http from 'node:http'
const server=http.createServer((req,res)=>{if(req.url==='/events'){res.writeHead(200,{'Content-Type':'text/event-stream'});res.write('data: ready\n\n')}else{res.end('<script>new EventSource("/events")</script>')}})
await new Promise(r=>server.listen(0,'127.0.0.1',r))
const owned=await launchOwnedBrowser(),context=await owned.browser.newContext(),page=await context.newPage()
const url='http://127.0.0.1:'+server.address().port
await page.goto(url)
await context.request.get(url)
if(process.argv.includes('--stalled-close')) owned.browser.close=()=>{process.kill(-owned.process.pid,'SIGKILL');return new Promise(()=>{})}
await owned.stop()
assert.ok(owned.process.exitCode!==null||owned.process.signalCode!==null,'owned Chrome process survived teardown')
server.closeAllConnections();await new Promise(r=>server.close(r))
console.log('Owned browser and SSE/request sockets stopped')
