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
- Do not use existing local definitions, deleted files, generated files, git history, or prior implementations as the basis, template, scaffold, checklist, starting draft, or restoration source for SDK type design.
- Do not restore deleted proto or Go files from git as a shortcut. If a file needs to exist again, re-author it from the TypeScript docs and only then wire it into the repository.
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
- Every handwritten Go type must have its own doc comment that includes the exact Claude SDK reference URL for that specific type or section anchor. A single file-level URL is insufficient.
- Every exported handwritten Go type doc comment must begin with the type name in standard Go doc-comment form, for example `TypeName is ...`, so editor and linter tooling do not flag it.
- Every proto message, enum, and union-carrier message must also carry its own doc comment with the exact corresponding Claude SDK reference URL. A file-level URL is insufficient there as well.
- Use anchored `https://code.claude.com/docs/en/agent-sdk/typescript#...` URLs that identify the relevant section for the specific type, such as `#message-types`, `#sdkassistantmessage`, or the exact anchored section that defines that type.
- Do not use the bare top-level TypeScript docs URL as a per-type source comment.
- Do not infer anchors mechanically from Go/proto type names. Use the actual anchor used by the documentation heading or section, even when it is non-obvious, shared across multiple types, or tool-name based, for example `https://code.claude.com/docs/en/agent-sdk/typescript#read-2`.
- If multiple adjacent types come from the same exact SDK section, repeat that anchored source URL on each type instead of relying on shared context.
- Carry the same SDK-derived comments and source URLs into the proto schema as well.

## Type Mapping

- Represent TypeScript interface extension or embedding with Go struct embedding.
- Map TypeScript arrays to `[]T`.
- Map `Record<string, T>`-style maps to `map[string]T`.
- If a TypeScript `unknown` or `any` is effectively `Record<string, unknown>`, use `map[string]any`.
- Otherwise represent `unknown` or `any` as `json.RawMessage`.
- When the Claude SDK docs reference Anthropic SDK-owned payload types that are named but not defined on the page, keep them as named `json.RawMessage` wrappers in handwritten Go and as raw JSON carrier messages in proto for now; add doc comments linking to the source mention instead of inventing a local structural model.
- Use `time.Time` for timestamp or date-like fields by default.
- If custom time formatting is required, wrap it in a dedicated type such as `type CustomTime struct { t time.Time }`.

## Optional And Nullable Fields

- Represent optional TypeScript fields as pointer fields in Go and tag them with `omitzero`.
- Represent nullable TypeScript fields such as `T | null` as pointer fields in Go without `omitzero`.
- Do not add `omitempty` to handwritten Go JSON tags for this SDK conversion. Use `omitzero` alone when omission-on-empty is required by an optional field.
- Assume the target Claude interfaces do not require optional-plus-nullable tri-state fields unless the docs prove otherwise.

## JSON Shape

- Use struct tags to match the documented Claude TypeScript JSON keys.
- Do not strip documented discriminators such as `type`; keep the original data shape by storing discriminator fields explicitly on concrete variant structs.
- Enforce discriminator correctness in `MarshalJSON` and `UnmarshalJSON` so a concrete type cannot round-trip with a conflicting discriminator value.
- Define custom JSON methods when needed rather than on every type.
- In practice, almost every SDK-facing handwritten Go type should define `MarshalJSON` and `UnmarshalJSON` explicitly so the docs-derived JSON shape is enforced rather than assumed.
- Only omit custom JSON methods for a handwritten Go type when the type is a trivially safe alias or plain struct with no optional/nullable nuance, no union participation, no discriminator constraints, no raw-payload preservation, and no risk of Go's default encoder producing a docs-incompatible shape.
- If there is any doubt, implement the JSON methods.

## Unions And String Literals

- Model each TypeScript union as a Go struct wrapping a single unexported field that holds the active variant: `type <UnionTypeName> struct { value <UnionTypeName>_Value }`. The union is a struct, not an interface.
- Define the field's type as an exported interface carrying exactly one unexported marker method: `type <UnionTypeName>_Value interface { <unionTypeName>() }`. Export the interface so accessors can return it without linter complaints; keep the method unexported so only this package can add variants (the union stays sealed).
- Every union variant must implement `<UnionTypeName>_Value`. Store variants as pointers (`*ConcreteVariant`); the marker method itself may use a value receiver.
- Immediately after the value interface, add a compile-time implementation check block `var ( _ <UnionTypeName>_Value = (*ConcreteVariant)(nil) ... )` covering every concrete variant, including the unknown variant. Keep it adjacent so omissions fail to compile.
- Provide a constructor `func New<UnionTypeName>(v <UnionTypeName>_Value) <UnionTypeName>` that wraps a variant into the struct. Callers construct unions through this constructor, never by setting the unexported field.
- Provide accessors on the struct: `GetValue() <UnionTypeName>_Value` returns the active variant (nil when unset), and `Get<Variant>() (*ConcreteVariant, bool)` type-asserts each case. Form the getter name by stripping the union type name prefix from the variant when the variant carries it (`SystemPromptString` → `GetString`, `PermissionUpdateUnknown` → `GetUnknown`); otherwise use the full variant type name (`BashOutput` → `GetBashOutput`, `PostToolUseHookInput` → `GetPostToolUseHookInput`).
- Do not define free-floating union marshalers or unmarshalers. JSON (de)serialization lives on the struct: `func (o <UnionTypeName>) MarshalJSON() ([]byte, error)` (normally `return json.Marshal(o.value)`, since each variant carries its own discriminator-enforcing marshaling) and `func (o *<UnionTypeName>) UnmarshalJSON(data []byte) error` (the discriminator dispatch, storing `o.value = &variant`).
- When a union can only be decoded with extra context — for example the tool-name-dispatched `ToolInputSchemas` / `ToolOutputSchemas` — expose the decoder as a method that takes that context, e.g. `func (o *<UnionTypeName>) UnmarshalForTool(toolName string, data []byte) error`, instead of a free function. Enclosing `UnmarshalJSON` implementations call these methods on a local union value.
- A union whose TypeScript form is only a grouping marker (a sub-union that carries a second marker method and is never used as a field/parameter/return type, e.g. `SDKResultMessage`, `AgentOutput`, `FileReadOutput`) may remain a plain Go interface; the struct treatment applies to unions actually used as field/parameter/return types.
- Treat unions as open-value.
- Always define an unknown variant type for each union so JSON decoding can preserve unsupported variants instead of failing.
- Store both the discriminator value and the original `json.RawMessage` on unknown union variants.
- Preserve round-trip conversion for unknown union variants.
- Generated proto conversion must also preserve unknown union variants by defining corresponding unknown proto variants.
- This is mandatory for both handwritten Go unions and proto unions. Do not leave either side without its explicit unknown variant.
- This applies to every union, not only selected high-value unions.
- Do not omit unknown proto variants just because protobuf `oneof` is inconvenient or because the current repository does not yet consume that path.
- Each proto unknown variant must be designed intentionally to preserve round-trip data. At minimum it must carry the discriminator value when the union is discriminator-based, and the original payload in a form that can be converted back without inventing or dropping fields.
- Proto union design is not complete until the unknown variant path exists and the handwritten Go conversion code maps to and from it.
- Represent string-literal unions and enum-like fields as named Go string types, for example `type <UnionTypeName> string`.
- Define string-literal union and enum constants in a `const (...)` block.
- Form constant names by prefixing the union or type name to the PascalCase form of the literal value. Example: `"string-content"` becomes `<UnionTypeName>StringContent`.
- For Claude hook- and permission-related discriminators, fields such as `behavior: "allow" | "deny"` must not remain plain `string` fields in handwritten Go. Model them with a named Go string type and constants, and use those constants instead of raw string literals in normal code paths.
- Do not reuse one named literal type across different SDK fields just because the current literal set overlaps. If two fields can evolve independently in the TypeScript SDK, they must have separate named Go types and separate proto enums.

## Execution Discipline

- First derive the target SDK shape from the TypeScript docs. Only after that may you inspect repository code for integration points.
- Do not mechanically merge, rename, or consolidate old local SDK files into the new implementation unless the resulting content has already been validated against the TypeScript docs field-by-field.
- If you find yourself reusing old repository SDK code because it is faster, stop. Re-derive the type from the docs instead.
- If you catch yourself copying from an old proto, old Go file, generated `.pb.go`, or git history before the docs-derived shape is written down, stop and restart from the TypeScript docs.

## Proto Conversion API

- Implement bi-directional proto conversion primarily as methods on the handwritten Go types.
- Also define package-level helpers for proto-to-Go conversion so callers can convert generated proto values into handwritten Go types easily.
- Treat bi-directional conversion as part of the definition of each handwritten SDK-facing Go type, not as optional follow-up work.
- For each handwritten Go type that has a proto representation, provide conversion coverage in both directions between the handwritten type and the generated proto type.
- Prefer methods such as `ToProto()` and `FromProto(...)` on the handwritten Go types, plus package-level helpers for unions and convenience entrypoints.
- For the struct-wrapped unions, keep the package-level `<UnionTypeName>ToProto` / `<UnionTypeName>FromProto` helpers: `ToProto` type-switches on the active variant (`v.GetValue()` or the in-package `value` field), and `FromProto` wraps the decoded variant back into the struct with `New<UnionTypeName>(...)`, returning the zero `<UnionTypeName>{}` (not `nil`) on the empty/error paths.
- Unknown union variants must also participate in bi-directional conversion and preserve discriminator plus raw payload without loss.
- A type implementation is not complete until its JSON behavior and its proto-to-Go / Go-to-proto behavior are both defined.
- In proto conversion code, do not read generated protobuf struct fields directly.
- Read generated proto values through `Get...` accessors, union oneof getters, or protobuf reflection helpers for presence-sensitive optional fields.
- This rule exists to keep the conversion layer compatible if generated protobuf code switches to opaque APIs where exported struct fields are not available.
- When converting getter-returned scalar values back into pointer fields on handwritten Go types, use a small helper such as `func opt[T comparable](v T) *T` instead of direct generated-field access.
- `proto3 optional` scalar fields are presence-sensitive. Treat “absent” and “present with zero value” as distinct states during proto-to-Go conversion.
- This matters for exact JSON round-trip requirements. Examples: omitted vs `false`, omitted vs `0`, and omitted vs `""` must remain distinguishable when the handwritten Go type uses pointer fields.
- Therefore, when reconstructing handwritten Go pointer fields from generated proto values, first check field presence, then read the value with `GetXX()`. Do not reconstruct pointer fields by reading generated struct fields directly.
- Plain non-optional proto3 scalar fields do not preserve this distinction and should not be treated as tri-state.

## Ambiguity Rule

- If the Claude SDK docs are ambiguous or underspecified for a type or field, stop and ask the user instead of inferring.
