# Move crabhook/api/ and buf config to top-level sdk_types/, rename tool_io to sdk_types

## Context

The proto definitions and buf configuration currently live under `crabhook/api/` and `crabhook/buf.*` with proto package `tool_io.v1`. Moving them to a top-level `sdk_types/` directory (keeping `api/` structure) and renaming the proto package from `tool_io` to `sdk_types` better reflects their purpose as shared SDK type definitions.

The generated Go code is **not imported anywhere**, so this is safe.

## Target Structure

```
sdk_types/
├── api/
│   ├── schema/proto/sdk_types/v1/   (was tool_io/v1, 8 proto files)
│   └── gen/proto/go/sdk_types/v1/   (regenerated)
├── buf.yaml
└── buf.gen.yaml
```

## Steps

### 1. Create directory and move files with git

```bash
mkdir -p sdk_types/api/schema/proto/sdk_types/v1
git mv crabhook/api/schema/proto/tool_io/v1/*.proto sdk_types/api/schema/proto/sdk_types/v1/
git mv crabhook/buf.yaml sdk_types/buf.yaml
git mv crabhook/buf.gen.yaml sdk_types/buf.gen.yaml
```

### 2. Remove old generated code

```bash
rm -rf crabhook/api/
```

### 3. Update buf.yaml

No change needed — module path `api/schema/proto` stays the same.

### 4. Update buf.gen.yaml

Update `go_package_prefix` only:
- `github.com/ngicks/crabswarm/crabhook/api/gen/proto/go` → `github.com/ngicks/crabswarm/sdk_types/api/gen/proto/go`

Output paths (`api/gen/proto/go`) stay the same.

### 5. Rename proto package in all 8 .proto files

In every proto file under `sdk_types/api/schema/proto/sdk_types/v1/`:
- `package tool_io.v1;` → `package sdk_types.v1;`
- All imports: `import "tool_io/v1/..."` → `import "sdk_types/v1/..."`

### 6. Regenerate proto code

```bash
cd sdk_types && buf lint && buf generate
```

### 7. Leave crabhook/ as-is

Only `.claude/` remains — leave it intact per user preference.

## Verification

1. `buf lint` passes in `sdk_types/`
2. `buf generate` succeeds in `sdk_types/`
3. `go build ./...` passes from repo root
4. No references to `crabhook/api` remain in source (except plan docs)
