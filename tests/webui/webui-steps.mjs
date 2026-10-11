// Shared browser steps for the Web UI proofs.

// runReviewedPlan clicks the reviewed plan's run button. An accepted job opens
// its own page; the step waits for it and goes back, so the job form (which
// keeps its choices per screen) continues where it was. A refused start leaves
// the page as it is for the proof to inspect. Returns the job id, or null when
// the start was refused, together with the start response.
export async function runReviewedPlan(page) {
 const started = page.waitForResponse(r => r.request().method() === 'POST' && new URL(r.url()).pathname === '/api/v1/jobs', { timeout: 60000 })
 await page.getByRole('button', { name: '검토한 계획 실행', exact: true }).click()
 const response = await started
 if (response.status() !== 202) return { id: null, response }
 const { id } = await response.json()
 await page.waitForURL(u => u.pathname === '/jobs/' + id, { timeout: 30000 })
 await page.getByTestId('job-continues').waitFor({ timeout: 15000 })
 await page.goBack()
 await page.getByRole('button', { name: '실행 계획 확인', exact: true }).waitFor({ timeout: 30000 })
 return { id, response }
}

// clickRun is runReviewedPlan for proofs that wait on the start response
// themselves: it clicks, follows an accepted job to its page and back, and
// returns the start response.
export async function clickRun(page) {
 return (await runReviewedPlan(page)).response
}
