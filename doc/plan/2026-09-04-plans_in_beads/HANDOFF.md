# Handoff

Work found while planning that this plan does not cover.

## Out-of-scope discovery — Playwright and the nix browsers disagree on the revision

`web/package.json` pins `@playwright/test` 1.61.0 and `web/playwright.config.ts`
documents that it expects chromium-1228, but `PLAYWRIGHT_BROWSERS_PATH` now
holds `chromium-1234` and `chromium_headless_shell-1234`. Playwright refuses
to launch (`Executable doesn't exist at .../chromium_headless_shell-1228/...`),
so `pnpm e2e` cannot run in this environment, and step 6's Playwright
verification depends on it. Found 2026-09-06 while driving the mock headless.

Follow-up: align the `@playwright/test` pin with the revision the nix side
provides (the user's call, per the web base preference rule: the nix side
leads, never `playwright install`). Not part of this plan. Until then
`pnpm e2e` fails at browser launch on this host; the specs pass with a
config that sets `launchOptions.executablePath` to the 1234 headless
shell and `FONTCONFIG_FILE` to a fontconfig naming a nix font directory
(the Ark combobox aborts Chromium's Skia font code without one). The
repository's `playwright.config.ts` carries neither, so the fontconfig
need is a second thing to settle when the pin is aligned.

## Out-of-scope discovery — the Stop hook ignores `stop_hook_active`

`hooks/issues-mermaid-lint/hooks/hook.json` runs `crabswarm hook exec
'crabswarm issues lint'` on every Stop. A blocking Stop hook fires again on
the agent's next turn, and the harness marks that turn with
`stop_hook_active: true` in the hook input so hooks can avoid a loop. The
template does not read it, so a broken diagram the agent cannot repair
re-blocks every turn. Found 2026-09-06 during the Stop dry run of step 3.

Follow-up: decide the loop policy (block once, then allow with the
findings as context is the obvious one) and, if a `hook exec` template
cannot express it, add a flag to `crabswarm hook exec` rather than a
shell wrapper, per the repository's hook rule.

## Out-of-scope discovery — `strict: true` in a mermaid-lint config

`mermaid-lint --format json` reports each rule's `severity` as `error` or
`warn` regardless of `strict`, while its exit status under a repository
config with `strict: true` fails on `warn` too. `crabswarm issues lint`
turns only `error`-severity rules into findings, so such a repository's
file hook and issue hook disagree on warnings. Found 2026-09-06 while
closing the parity gap for error-severity rules.

Follow-up: read the effective config's `strict` (or honour `--strict`) in
`crabswarm/issues/mermaidlint` and promote `warn` findings when set.
