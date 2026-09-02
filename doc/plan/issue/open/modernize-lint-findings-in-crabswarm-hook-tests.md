---
tags: hook lint
---

# Modernize lint findings in crabswarm/hook tests (2026-09-02)

A repo-wide `golangci-lint run` surfaces pre-existing `modernize`
findings in untouched files: `crabswarm/hook/audit_test.go:90` and
`:103` want `errors.As` replaced with `errors.AsType`.

Follow-up: apply the two mechanical replacements.
