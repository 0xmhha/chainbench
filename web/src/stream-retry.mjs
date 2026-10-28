// How the event stream reconnects: each failure in a row doubles the wait
// from two seconds up to thirty, so a server that is down is not asked again
// every two seconds by every open tab.
const FIRST = 2000, CEILING = 30000

export function retryDelay(failures) {
 return Math.min(FIRST * 2 ** failures, CEILING)
}

// A fetch that never reached the server rejects with a TypeError, which the
// shared client turns into an unreachable error; either is said in the
// screen's own words, with when it is tried again.
export function unreachableMessage(failure, delay) {
 if (failure instanceof TypeError || failure?.unreachable) return `서버에 연결할 수 없습니다. ${Math.round(delay / 1000)}초 뒤 다시 시도합니다.`
 return failure.message
}
