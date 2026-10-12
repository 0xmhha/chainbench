import { chromium } from 'playwright'

// Keep the public BrowserServer handle so teardown owns the exact process it
// launched. Continuous SSE requests must not prolong graceful browser shutdown.
export async function launchOwnedBrowser() {
  const server = await chromium.launchServer({ headless: true, channel: process.env.WEBUI_BROWSER_CHANNEL ?? 'chrome', host: '127.0.0.1' })
  try {
    const browser = await chromium.connect(server.wsEndpoint())
    return { browser, stop: async () => {
      // A dead Chrome can leave its client close acknowledgement pending.
      // Bound graceful disposal; the exact process and listener still terminate.
      async function settle(promise) {
        let timer
        try {
          await Promise.race([promise.catch(() => {}), new Promise(resolve => { timer=setTimeout(resolve,2000) })])
        } finally { clearTimeout(timer) }
      }
      try {
        await settle(Promise.all(browser.contexts().map(context => context.request.dispose())))
        await settle(browser.close())
        // Only pinned Playwright test tooling uses this listener cleanup hook.
        await server._disconnectForTest()
      } finally {
        const child=server.process()
        const killed=server.kill()
        let timer, exited
        try {
          await new Promise((resolve,reject) => {
            if(child.exitCode!==null||child.signalCode!==null) return resolve()
            exited=resolve; child.once('exit',exited)
            timer=setTimeout(()=>reject(new Error('owned Chrome did not exit')),5000)
          })
        } finally { clearTimeout(timer); if(exited) child.removeListener('exit',exited) }
        // Chrome's detached updater/crashpad can inherit stderr. Only close
        // this fixture's pipe ends after its exact Chrome process has exited.
        for(const stream of child.stdio) stream?.destroy?.()
        await killed
      }
    }, process: server.process() }
  } catch (error) {
    await server.kill()
    throw error
  }
}
