# Contributing: Configuration Changes

The configuration surface — the `option/` structs, the generated JSON Schema and
the documentation — is a single, versioned, user-visible product. Every option
field, inbound/outbound/service type, rule item and shared option struct follows
the policy below.

## Single source of truth

- Each configuration field is declared exactly once, as a Go struct field in
  `option/`, with a `snake_case` `json` tag. There is no second definition to
  keep in sync.
- `docs/schema.json` is **generated**, never hand-edited. Regenerate it with
  `make schema`, which runs `sing-box schema -o docs/schema.json` (`Makefile`,
  `cmd/sing-box/cmd_schema.go`). The schema is produced by reflecting over the
  `option` structs and their `DescribeSchema` implementations
  (`schema/generate.go`, `schema/generator.go`).
- Unknown fields are hard-rejected: `Options.UnmarshalJSONContext` calls
  `decoder.DisallowUnknownFields()` (`option/options.go`), and nested structs
  use `json.UnmarshalDisallowUnknownFields` / `UnmarshalContextDisallowUnknownFields`
  (for example `option/direct.go`, `option/dns.go`, `option/rule_action.go`).
  Do not add permissive "ignore unknown" paths.

## Uniformity

A new field must look like the fields already next to it.

- JSON tag names are `snake_case`; `,omitempty` marks optional fields and is
  omitted for required ones (`option/options.go`).
- Add the schema tags consumed by `schema/generator.go` where applicable:
  - `enum:"a,b,c"` for a closed set of strings;
  - `examples:"..."` for sample values (`option/options.go`, `option/tls.go`);
  - `reference:"some_type"` when a string references another declared type
    (`option/group.go`, `option/dns.go`);
  - `schema:"omit"` for retired fields that must keep parsing but disappear from
    the schema (`schema/generator.go` rejects any other `schema` tag value).
- Polymorphic types (a `type` discriminator selecting between option structs)
  follow one pattern:
  1. an unexported `_Type` struct holding the shared fields, including
     `` Type string `json:"type" enum:"..."` ``, plus one field per variant
     tagged `json:"-"`;
  2. an exported alias `type X _Type`;
  3. `MarshalJSON` / `UnmarshalJSON` that switch on `Type` and combine the base
     struct with the selected variant using `badjson.MarshallObjects` /
     `badjson.UnmarshallExcluded`;
  4. `DescribeSchema` returning `schema.DiscriminatedUnion(...)`.

  The reference implementations are `option/hysteria2.go` (`Hysteria2Obfs`,
  `Hysteria2Masquerade`) and `option/v2ray_transport.go`
  (`V2RayTransportOptions`). Mirror them exactly.

## Adding a field

Adding a field is normally a single-file change plus schema regeneration and a
documentation section. The checklist:

1. **Declare** the field in the relevant `option/*.go` struct with a
   `snake_case` `json` tag and `,omitempty` where appropriate, plus
   `enum`/`examples`/`reference` tags when applicable.
2. **Wire** it: constructors return `(*T, error)` and never panic; validate in
   the struct's check function or the consuming component using `E.New(...)` /
   `E.Cause(err, ...)`.
3. **Regenerate the schema**: `make schema` (updates `docs/schema.json`).
4. **Document it**: add the field to the matching page under
   `docs/configuration/`.
5. **Format**: `make fmt` for Go and `make fmt_docs` if you added `:material-`
   markers to a docs page.

## Backward compatibility

Removing or renaming a field is never done silently.

- **Removal requires a deprecation cycle.** Add a `deprecated.Note` to
  `experimental/deprecated/constants.go` with `Name`, `Description`,
  `DeprecatedVersion`, `ScheduledVersion`, `EnvName` and, where one exists,
  `MigrationLink`; add the note to the exported `Options` slice. Report it from
  the code that consumes the option with `deprecated.Report(ctx, note)`
  (`experimental/deprecated/manager.go`). Mark the retired struct field
  `schema:"omit"` and add a `// Deprecated:` comment, so it keeps parsing while
  leaving the schema.
- **Deprecations are reported at startup.** `experimental/deprecated/stderr.go`
  logs a warning for each note reported. For an *impending* note — one at or
  within one minor release of its `ScheduledVersion` (`Note.Impending()`) — it
  logs an error, and when the note has an `EnvName` it aborts startup unless the
  escape hatch is set: `ENABLE_DEPRECATED_<EnvName>=true`.
- **Rename without breaking.** A renamed key keeps parsing its old spelling for
  the duration of the cycle. Keep a field whose `json` name is the old spelling
  (named e.g. `Deprecated_<Name>`) with `schema:"omit"`, and handle it in the
  runtime — see `option/rule.go` / `option/rule_dns.go` and their consumers such
  as `route/rule/rule_dns.go`. There are no silent renames.
- **Record it in the docs.** Every deprecation gets an entry in
  `docs/deprecated.md`, grouped by version, and — for user-migratable features —
  a recipe in `docs/migration.md` that `MigrationLink` points to.
- **Never loosen unknown-field handling.** Unknown fields stay hard-rejected.

## Format

- YAML is the documented configuration format; JSON is fully supported with the
  identical schema and validation rules (`docs/configuration/index.md`).
- The tooling is `sing-box format` (with `-w` to rewrite files in place),
  `sing-box merge` and `sing-box check` (`cmd/sing-box/cmd_format.go`,
  `cmd_merge.go`, `cmd_check.go`). `check` builds the configuration to validate
  it.

## One commit

Code, the regenerated `docs/schema.json`, documentation and any
deprecation/migration entries land in the **same commit**. A schema change
without its docs, or a deprecation without its migration recipe, is incomplete.

## Review checklist

- [ ] Field declared once in `option/` with a `snake_case` `json` tag.
- [ ] `,omitempty` and required fields match surrounding conventions.
- [ ] `enum` / `examples` / `reference` tags added where applicable.
- [ ] Polymorphic types follow the `_Type` + `Marshal`/`Unmarshal` +
      `DescribeSchema` pattern (`option/hysteria2.go`,
      `option/v2ray_transport.go`).
- [ ] `docs/schema.json` regenerated with `make schema`; not hand-edited.
- [ ] Matching page under `docs/configuration/` updated.
- [ ] Removals carry a `deprecated.Note`, `deprecated.Report`, `schema:"omit"`,
      a `docs/deprecated.md` entry, a `docs/migration.md` recipe and the
      `ENABLE_DEPRECATED_<ENV>` escape hatch.
- [ ] Renames keep parsing the old spelling during the deprecation cycle.
- [ ] Unknown fields remain hard-rejected.
- [ ] `sing-box check` passes for a representative configuration.
