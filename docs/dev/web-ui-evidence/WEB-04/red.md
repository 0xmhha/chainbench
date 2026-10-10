# WEB-04 RED

(plugins, external manifests)

- No product RED recorded.
- Harness (not product): two WEB-04 attempts were refused by the existing disk minimum, which the sources identify as an environment failure. Source: PR ("Two WEB-04 attempts were refused by the existing disk minimum"), HO l.345-346.
- Harness (not product): stale WEB-03/04/10 receipts after the SPA rebuild. Source: HO l.437.

---

Sources: `docs/dev/web-ui-worklist.md` stage table (WL, row named in Korean in quotes; the commit that added each row was confirmed with `git log -p -- docs/dev/web-ui-worklist.md`), `docs/dev/web-ui-claude-handoff.md` (HO, line numbers), PR #443 body (PR), and `git log main..HEAD` commit messages (CM). RED log files are under `/private/tmp/` as the sources name them; they were not re-opened here.

Labels:
- **RED**: product failure observed before the fix.
- **Defect (CM)**: product defect stated in a commit message without a named RED log.
- **Runner RED**: the old `verify.sh` refused the criterion. This is verification tooling, not product behaviour.
- **Harness (not product)**: test, fixture or environment failure that the sources explicitly exclude from product RED.
