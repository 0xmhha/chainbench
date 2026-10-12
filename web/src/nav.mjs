// go opens another screen without reloading. App listens for popstate, so a
// pushed path is handled the same way as the browser's back and forward.
export function go(path) {
 history.pushState({}, '', path)
 dispatchEvent(new PopStateEvent('popstate'))
}

// follow turns a plain left click on a link into go(); other clicks (new tab,
// modifier keys) keep the browser's behaviour.
export function follow(event, path) {
 if (event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) return
 event.preventDefault()
 go(path)
}
