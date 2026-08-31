# Status

Current state: **not started** — plan drafted with automatic decisions
(user directive to skip questions); IDEA.md gate not yet confirmed by the
user.

## Checklist

- [ ] Step 1 — resolver carries the name: `TeamInfo.Name` + label constants +
      derivation in `CmdmanCompose.Resolve`, unit tests
      (D1 "Name field filled by CmdmanCompose.Resolve",
      D2 "`<command>-<scale-index>`, degrading gracefully",
      D4 "label values used verbatim")
- [ ] Step 2 — `Service.Join` prefers `req.Name` → `info.Name` →
      `defaultName(token)`; update `defaultName` doc comment; service tests
      (D1; D3 "clear rejection at join time" untouched;
      D5 "no migration for already-stored names")
- [ ] Step 3 — e2e: default-named member listed as `<command>-<scale-index>`
      (goal criterion "covered by ... an e2e join test")

Next action: user reviews IDEA.md gate + automatic decisions, then step 1.
