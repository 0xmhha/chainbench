<script>
 // A destructive action asked in place: what will happen, the button that
 // does it, and the way back. It replaces window.confirm so the question sits
 // beside what it acts on and reads in the page's own type.
 let {label,message,confirmLabel,busy=false,onconfirm,oncancel}=$props()
</script>
<div class="confirm" role="group" aria-label={label}>
 <p>{message}</p>
 <button class="danger" disabled={busy} onclick={onconfirm}>{confirmLabel}</button>
 <button disabled={busy} onclick={oncancel}>돌아가기</button>
</div>
<style>
 .confirm { margin-top: var(--space-3); padding: var(--space-3) 0; border-top: 2px solid var(--danger); border-bottom: 2px solid var(--danger); }
 .confirm p { margin: 0 0 var(--space-2); max-width: 80ch; }
 .confirm button { margin-right: var(--space-2); }
</style>
