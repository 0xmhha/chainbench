<script>
  // The dashboard SPA consumes the Go server's contract: the SSE stream at
  // /events (obs.Event JSON) and the run list at /api/runs (obs.RunRecord JSON).
  let events = $state([])
  let runs = $state([])
  let connected = $state(false)

  async function loadRuns() {
    try {
      const r = await fetch('/api/runs')
      runs = (await r.json()) ?? []
    } catch {
      runs = []
    }
  }

  $effect(() => {
    loadRuns()
    const es = new EventSource('/events')
    es.onopen = () => (connected = true)
    es.onerror = () => (connected = false)
    es.onmessage = (m) => {
      try {
        const e = JSON.parse(m.data)
        events = [e, ...events].slice(0, 200)
        if (e.kind === 'result') loadRuns()
      } catch {
        // ignore malformed frames
      }
    }
    return () => es.close()
  })

  const fmtTime = (t) => (t ? new Date(t).toLocaleTimeString() : '')
  const runOk = (s) => s === 'success' || s === 'pass'
</script>

<main>
  <header>
    <h1>chainbench</h1>
    <span class="conn" class:on={connected}>{connected ? 'live' : 'disconnected'}</span>
  </header>

  <section>
    <h2>Runs</h2>
    {#if runs.length === 0}
      <p class="empty">no runs yet</p>
    {:else}
      <table>
        <thead>
          <tr><th>id</th><th>phase</th><th>chain</th><th>network</th><th>status</th></tr>
        </thead>
        <tbody>
          {#each runs as r (r.id)}
            <tr>
              <td>{r.id}</td>
              <td>{r.phase}</td>
              <td>{r.chain}</td>
              <td>{r.network}</td>
              <td class:ok={runOk(r.status)} class:bad={!runOk(r.status)}>{r.status}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    {/if}
  </section>

  <section>
    <h2>Events</h2>
    {#if events.length === 0}
      <p class="empty">waiting for events…</p>
    {:else}
      <ul class="events">
        {#each events as e, i (i)}
          <li>
            <span class="t">{fmtTime(e.time)}</span>
            <span class="phase">{e.phase}</span>
            <span class="kind kind-{e.kind}">{e.kind}</span>
            {#if e.network}<span class="net">{e.network}</span>{/if}
            {#if e.node}<span class="node">node{e.node}</span>{/if}
            <span class="msg">{e.message}</span>
          </li>
        {/each}
      </ul>
    {/if}
  </section>
</main>

<style>
  :global(body) {
    margin: 0;
    font: 14px/1.4 ui-monospace, SFMono-Regular, Menlo, monospace;
    background: #0e1116;
    color: #d7dde5;
  }
  main { max-width: 960px; margin: 0 auto; padding: 1.5rem; }
  header { display: flex; align-items: center; gap: 0.75rem; }
  h1 { font-size: 1.3rem; margin: 0; }
  h2 { font-size: 1rem; margin: 1.5rem 0 0.5rem; color: #9aa7b4; }
  .conn { font-size: 0.8rem; padding: 0.1rem 0.5rem; border-radius: 0.75rem; background: #4a1f1f; color: #ff9a9a; }
  .conn.on { background: #14351f; color: #6ee7a0; }
  .empty { color: #6b7683; }
  table { width: 100%; border-collapse: collapse; }
  th, td { text-align: left; padding: 0.3rem 0.6rem; border-bottom: 1px solid #222833; }
  th { color: #6b7683; font-weight: 600; }
  td.ok { color: #6ee7a0; }
  td.bad { color: #ff9a9a; }
  ul.events { list-style: none; margin: 0; padding: 0; }
  ul.events li { display: flex; gap: 0.5rem; padding: 0.2rem 0; border-bottom: 1px solid #1a1f27; white-space: nowrap; overflow: hidden; }
  .t { color: #6b7683; }
  .phase { color: #7aa2f7; }
  .kind { color: #9aa7b4; }
  .kind-error { color: #ff9a9a; }
  .kind-result { color: #6ee7a0; }
  .net, .node { color: #b48ead; }
  .msg { color: #d7dde5; overflow: hidden; text-overflow: ellipsis; }
</style>
