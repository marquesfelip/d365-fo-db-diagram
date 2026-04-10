# Copilot Instructions — d365-fo-db-diagram

## Project Purpose

This project automates D365 F&O database diagram generation for developers. It is a **multi-component pipeline**:

```
[D365 XML files] → (1) Go XML reader → (2) Relational DB → (3) Diagram visualizer
```

1. **XML Reader** (`structure-reader/` — current Go code): parses AxTable XML metadata files to extract tables, fields, ExtendedDataTypes, and table relations.
2. **Storage** *(planned)*: persists parsed data into a relational database so it can be queried and diffed.
3. **Diagram** *(planned)*: reads from the DB and renders a visual ER diagram to help D365 F&O developers understand and navigate the data model.

The Go CLI tool is **component 1 only**. Its sole responsibility is reading XML files, structuring the data, and sending it to the database (or stdout while the DB layer is not yet wired up).

## Repository Layout

```
structure-reader/       # Go application (module root)
  main.go               # Entry point: directory walking, concurrency orchestration, stdout output
  entity/entity.go      # XML struct definitions: AxTable, Descriptor, TableFieldInfo
  go.mod / go.sum       # Module: github.com/marquesfelip/d365-fo-db-diagram
temp/
  PackageLocalDirectory/ # D365 F&O package export — INPUT (not committed, place files here)
    <Package>/
      Descriptor/        # Module descriptor XMLs (.xml → AxModelInfo)
      <ModuleName>/
        AxTable/         # One XML per table (AxTable root element)
  resultado.txt          # Example output snapshot
```

## Build & Run

```bash
# From the project root
cd structure-reader

# Download dependencies
go mod download

# Run directly (results to stdout, progress to stderr)
go run main.go > ../temp/resultado.txt

# Build binary
go build -o ../d365-structure-reader .
```

> The program **must be run from the `structure-reader/` directory** (it resolves `temp/PackageLocalDirectory` relative to the working directory). Place the D365 package export under `temp/PackageLocalDirectory/` before running.

## Output Format

Each record is printed to **stdout**:
```
tabela: <TableName>, campo: <FieldName>, ExtendedDataType: <EDT>
```
Progress and errors go to **stderr** (colored, using `github.com/fatih/color`).

## Architecture & Key Patterns

### XML Reader (Go — `structure-reader/`)
- **Concurrency**: AxTable XML files are parsed in parallel using `golang.org/x/sync/errgroup` with a pool of 8 workers (`maxWorkers = 8`). Shared results are protected by `sync.Mutex`.
- **Progress tracking**: An `atomic.Int64` counter (`processed`) is updated by workers; a background goroutine ticks every second to print progress to stderr.
- **Entity structs** live in `entity/entity.go`. All XML parsing uses `encoding/xml` with struct tags. Add new D365 metadata types here.

### Data model extracted from XML
Each AxTable XML contains:
- **Table metadata**: Name, TableGroup, TableType, PrimaryIndex, SaveDataPerCompany
- **Fields** (`AxTableField`): Name, ExtendedDataType — mapped to DB columns
- **Relations** (`AxTableRelation`): Name, RelatedTable, Cardinality, RelatedTableCardinality, RelationshipType — the FK graph for the ER diagram. **Struct exists (`entity.AxTableRelation`) but extraction is not yet wired up — this is the next priority.**

### Storage layer *(planned — component 2)*
- The Go reader will write structured records to a relational database instead of stdout.
- Database engine TBD. Design the schema to store: tables, fields, relations, and package/module provenance.
- Keep DB writes behind an interface/port so the XML reader is not coupled to any specific driver.

### Diagram layer *(planned — component 3)*
- Reads the relational DB and renders a visual ER diagram.
- Technology TBD (could be a separate service, web UI, or generated Mermaid/DOT files).

## Conventions

- **Language**: code comments and variable names use **Brazilian Portuguese** (e.g., `processarPasta`, `resultado`, `erro`). Keep new code consistent.
- **Error handling**: errors are logged to stderr with the `errColor` printer and execution continues — the tool is best-effort (partial results are better than none).
- **Stdout purity**: while the DB layer is absent, only structured records go to stdout so the output can be piped or redirected cleanly.

## Key Dependencies

| Package | Purpose |
|---|---|
| `github.com/fatih/color` | Colored stderr output |
| `golang.org/x/sync/errgroup` | Bounded concurrent goroutine pool |

## Adding New D365 Metadata Types

1. Add struct(s) to `entity/entity.go` with `xml:"..."` tags matching the XML element names.
2. Add a parser function in `main.go` following the pattern of `readAxTableXML`.
3. Integrate it in `processDescriptorFolder` or a new sibling function.

## Roadmap — What's Not Here Yet

| Item | Notes |
|---|---|
| Relation extraction | `entity.AxTableRelation` struct is defined; parsing and output not wired up yet — **next step** |
| Database writer | Component 2; schema and driver TBD |
| Diagram renderer | Component 3; depends on DB layer |
| Tests | No tests yet — use the `go-test-strategy` agent to design a suite |
