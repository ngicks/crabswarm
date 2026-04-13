# Claude SDK Interface Conversion

This document defines the conversion rules for reimplementing or updating Claude SDK-facing interfaces in this repository.

## Purpose

- Treat the Claude Agent SDK TypeScript interfaces as the source of truth.
- Keep a reusable rule set for LLM-authored updates to Go types and proto schema.
- Stop and ask when the Claude docs are ambiguous instead of inventing policy.

## Workflow

1. Read this document before interface-related work.
2. If any rule is unclear or the Claude docs are underspecified, ask the user one question at a time.
3. After each clarified decision, update this document so the rule remains reusable.

## Source Of Truth And Scope

- Source of truth: the TypeScript interfaces documented at <https://platform.claude.com/docs/en/agent-sdk/typescript>. When local proto or Go shapes differ, the TypeScript SDK shape wins.
- Ignore existing local Go types, proto schema, generated protobuf output, repository history, and other repository files when defining the SDK surface, except where repository-specific placement or build wiring is required.
- This is a hard rule, not a suggestion: existing local SDK-facing code may be consulted only after the SDK surface has already been derived from the TypeScript docs, and then only for integration wiring such as package placement, imports, callers, or build regeneration.
- Do not use existing local definitions, deleted files, generated files, git history, or prior implementations as the basis, template, scaffold, checklist, or starting draft for SDK type design.
- If the TypeScript docs and the old local implementation disagree, the old local implementation is wrong for the purpose of the conversion and must be ignored.
- Scope: implement the full Claude Agent SDK TypeScript type surface for `sdk_types/v1/`, not only the subset already used by this repository.

## Placement And Generation Boundary

- Handwritten SDK-shaped types and conversion code live under `pkg/api/types/sdk_types/v1/`.
- Keep the first-pass implementation in a single file within `pkg/api/types/sdk_types/v1/`, and place both the handwritten SDK-shaped types and all proto conversion logic in that file.
- Keep the first-pass proto schema in a single file under `pkg/api/schema/proto/sdk_types/v1/`. Do not split the SDK proto definitions across multiple files unless the user explicitly asks for that refactor later.
- Updating the proto schema and regenerating generated code is in scope. Do not treat existing proto definitions as fixed constraints.
- The handwritten Go file and the proto schema are authored and updated directly by the LLM as normal source files.
- Regenerate generated outputs after source edits using `buf generate`.

## Naming And Documentation

- Preserve TypeScript aliases as named Go aliases or named Go types instead of inlining primitives everywhere.
- Use the TypeScript type name as-is where possible, uppercasing the first letter so the Go type is exported.
- Use normal Go field naming conventions, and preserve the documented wire names in JSON struct tags.
- Reflect relevant comments from the Claude SDK docs in the handwritten Go types.
- Include a link to the Claude SDK documentation source for each generated handwritten type.
- Carry the same SDK-derived comments and source URLs into the proto schema as well.

## Type Mapping

- Represent TypeScript interface extension or embedding with Go struct embedding.
- Map TypeScript arrays to `[]T`.
- Map `Record<string, T>`-style maps to `map[string]T`.
- If a TypeScript `unknown` or `any` is effectively `Record<string, unknown>`, use `map[string]any`.
- Otherwise represent `unknown` or `any` as `json.RawMessage`.
- Use `time.Time` for timestamp or date-like fields by default.
- If custom time formatting is required, wrap it in a dedicated type such as `type CustomTime struct { t time.Time }`.

## Optional And Nullable Fields

- Represent optional TypeScript fields as pointer fields in Go and tag them with `omitzero`.
- Represent nullable TypeScript fields such as `T | null` as pointer fields in Go without `omitzero`.
- Assume the target Claude interfaces do not require optional-plus-nullable tri-state fields unless the docs prove otherwise.

## JSON Shape

- Use struct tags to match the documented Claude TypeScript JSON keys.
- Do not strip documented discriminators such as `type`; keep the original data shape by storing discriminator fields explicitly on concrete variant structs.
- Enforce discriminator correctness in `MarshalJSON` and `UnmarshalJSON` so a concrete type cannot round-trip with a conflicting discriminator value.
- Define custom JSON methods when needed rather than on every type.
- In practice, expect many types to need custom JSON methods because union-typed fields are common.

## Unions And String Literals

- Model each TypeScript union as a Go interface with a private marker method `interface { <unionTypeName>() }`.
- Every union variant must implement that interface.
- Define `unmarshal<UnionTypeName>(...)` helpers to decode union variants and use those helpers from enclosing `UnmarshalJSON` implementations.
- Treat unions as open-value.
- Always define an unknown variant type for each union so JSON decoding can preserve unsupported variants instead of failing.
- Store both the discriminator value and the original `json.RawMessage` on unknown union variants.
- Preserve round-trip conversion for unknown union variants.
- Generated proto conversion must also preserve unknown union variants by defining corresponding unknown proto variants.
- This applies to every union, not only selected high-value unions.
- Do not omit unknown proto variants just because protobuf `oneof` is inconvenient or because the current repository does not yet consume that path.
- Each proto unknown variant must be designed intentionally to preserve round-trip data. At minimum it must carry the discriminator value when the union is discriminator-based, and the original payload in a form that can be converted back without inventing or dropping fields.
- Proto union design is not complete until the unknown variant path exists and the handwritten Go conversion code maps to and from it.
- Represent string-literal unions and enum-like fields as named Go string types, for example `type <UnionTypeName> string`.
- Define string-literal union and enum constants in a `const (...)` block.
- Form constant names by prefixing the union or type name to the PascalCase form of the literal value. Example: `"string-content"` becomes `<UnionTypeName>StringContent`.

## Execution Discipline

- First derive the target SDK shape from the TypeScript docs. Only after that may you inspect repository code for integration points.
- Do not mechanically merge, rename, or consolidate old local SDK files into the new implementation unless the resulting content has already been validated against the TypeScript docs field-by-field.
- If you find yourself reusing old repository SDK code because it is faster, stop. Re-derive the type from the docs instead.

## Proto Conversion API

- Implement bi-directional proto conversion primarily as methods on the handwritten Go types.
- Also define package-level helpers for proto-to-Go conversion so callers can convert generated proto values into handwritten Go types easily.

## Ambiguity Rule

- If the Claude SDK docs are ambiguous or underspecified for a type or field, stop and ask the user instead of inferring.
