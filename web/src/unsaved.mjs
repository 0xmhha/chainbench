// warnWhenUnsaved asks the browser to confirm a reload or a closed tab while
// isDirty holds. Moving between screens keeps every form, so only leaving the
// page can lose an edit. It returns the function that stops the warning.
export function warnWhenUnsaved(isDirty, target = globalThis) {
 const hold = event => { if (isDirty()) event.preventDefault() }
 target.addEventListener('beforeunload', hold)
 return () => target.removeEventListener('beforeunload', hold)
}
