import { chromium } from 'playwright'

// Keep the public BrowserServer handle so teardown owns the exact process it
// launched. Continuous SSE requests must not prolong graceful browser shutdown.
export async function launchOwnedBrowser() {
  const server = await chromium.launchServer({ headless: true, channel: process.env.WEBUI_BROWSER_CHANNEL ?? 'chrome', host: '127.0.0.1' })
  try {
    const browser = await chromium.connect(server.wsEndpoint())
    return { browser, stop: async () => {
      // BrowserServer runs request agents in this Node process. Dispose them
      // before dropping the remote connection so their sockets cannot outlive it.
      try {
        await Promise.all(browser.contexts().map(context => context.request.dispose()))
        await browser.close()
      } finally {
        await server.kill()
      }
    }, process: server.process() }
  } catch (error) {
    await server.kill()
    throw error
  }
}
