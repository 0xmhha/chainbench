# WEB-10 RED

(roles, secrets)

- RED: node private keys in shared presets and cases were accepted, the import confirmation was not rechecked, and the browser found an editor bypass (`chainbench-web-document-{secrets,import-recheck,editor}-red.log`). Fixed in `26207819` (fix(web): refuse node private keys in shared configurations). Source: WL "공유 문서의 노드 개인키 거절".
- RED: legacy documents and previews with node keys were not quarantined, and historical output was not redacted (`chainbench-web-{document-quarantine,quarantine-redaction,import-quarantine-redaction}-red.log`). Fixed in `64ee7671` (fix(web): quarantine legacy node keys before serving configurations). Source: WL "구형 문서/미리보기의 노드 개인키 격리".
- RED: a keystore was shared through the asset API, and a storage error exposed paths. Fixed in `50e4e52c`. Source: WL "업로드 바이너리 실행".
- RED: the plan response and pinning of key snapshots failed every predefined criterion. Fixed in `6c98916c`. Source: WL "키 자료 고정".
- RED: `TestAccountKeyCredentialStaysPrivateAndIsNoSSHLogin` and related tests failed before implementation. Fixed in `a9bf05ed`. Source: WL "Web attach 실행".
- RED: refused changes showed only generic text. Fixed in `32aa8140` with redacted reasons for 400/409/422 only. Source: WL "검증 오류의 구체적 이유 표시".
- Harness (not product): stale WEB-03/04/10 receipts after the SPA rebuild. Source: HO l.437.

---

Sources: `docs/dev/web-ui-worklist.md` stage table (WL, row named in Korean in quotes; the commit that added each row was confirmed with `git log -p -- docs/dev/web-ui-worklist.md`), `docs/dev/web-ui-claude-handoff.md` (HO, line numbers), PR #443 body (PR), and `git log main..HEAD` commit messages (CM). RED log files are under `/private/tmp/` as the sources name them; they were not re-opened here.

Labels:
- **RED**: product failure observed before the fix.
- **Defect (CM)**: product defect stated in a commit message without a named RED log.
- **Runner RED**: the old `verify.sh` refused the criterion. This is verification tooling, not product behaviour.
- **Harness (not product)**: test, fixture or environment failure that the sources explicitly exclude from product RED.
