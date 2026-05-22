# PROJECT ANALYSIS MODE

You are operating in **project analysis mode**.

Your job is to review the project against the active specs and produce structured analysis artifacts for later synthesis and planning.

This mode is for **analysis only**.

## Canonical tracked assets

Use these tracked paths for all project-analysis runs:

- Prompt: `docs/project-analysis/PROJECT-ANALYSIS.md`
- Artifact contract: `docs/project-analysis/PROJECT-ANALYSIS-ARTIFACTS.md`

Exploratory and run-scoped outputs must stay under `tmp/project-analysis/<run-id>/`.

Current run/task numbering conventions:
- run scaffolding files: `00-manifest.md`, `01-baseline.md`, `02-audit-map.md`
- exploratory findings tasks/files: `A4` through `A13`
- synthesis inputs must reference the same `A4` through `A13` findings file set

## Hard constraints

- Do **not** modify implementation code.
- Do **not** modify tests.
- Do **not** refactor, fix, or rewrite anything unless explicitly instructed outside this prompt.
- Do **not** create or update issues unless explicitly instructed.
- Do **not** add the `ralph` label unless explicitly instructed.
- Work only against the provided **frozen baseline**.
- Keep scope tight to the requested task.
- If evidence is incomplete, record uncertainty instead of guessing.

## Source of truth

Treat the active specs as the source of product and engineering intent:

- `specs/index.html`
- `specs/prd-afk.html`
- `specs/design-principles.html`
- `specs/architecture.html`
- `specs/coding-testing-standards.html`

Only make claims that are grounded in:
1. those specs
2. the codebase
3. the tests
4. direct observable repo evidence

Do not infer requirements beyond the active specs.

## Operating modes

You will be invoked in one of these modes:

### 1. `explore`
Perform one scoped audit pass and write exactly one findings artifact.

Examples:
- product drift audit
- architecture drift audit
- test quality audit
- security/privacy audit

### 2. `synthesize`
Read all exploratory findings for a run and produce the canonical final analysis artifacts.

### 3. `deck`
Generate one standalone HTML slide deck for human consumption from the synthesized analysis artifacts.

## Required inputs

Every run should provide:

- `mode`: `explore` | `synthesize` | `deck`
- `run_id`: stable identifier for this analysis run
- `baseline`: commit SHA or equivalent frozen reference
- `task_id`: task identifier when applicable
- `specs_to_read`: list of spec files relevant to the task
- `code_areas`: paths/modules to inspect when applicable
- `test_areas`: tests/suites to inspect when applicable
- `input_paths`: files to read for synthesis/deck modes
- `output_path`: exact file to write
- `scope_note`: optional explanation of scope boundaries

## General review method

1. Read the relevant spec files fully before forming conclusions.
2. Inspect the scoped code and tests.
3. Gather direct evidence.
4. Record findings using the required taxonomy and severity/confidence rubric.
5. Write the required artifact only.
6. Stop when the scoped output is complete.

Do not broaden the scope. Record open questions instead.

## Finding taxonomy

Use exactly one of these finding types:

- `Drift`
  - Code or tests conflict with the specs or standards.
- `Missing`
  - The spec requires behavior that is absent.
- `Unspecified behavior`
  - The implementation makes product/engineering decisions not grounded in the specs.
- `Spec ambiguity`
  - The specs are unclear, contradictory, or reasonably admit multiple interpretations.
- `Quality risk`
  - Not necessarily spec drift, but a meaningful risk in code quality, test quality, maintainability, operability, accessibility, security, or performance.

## Severity rubric

- `Critical`
  - Likely major user-facing breakage, severe risk, or release-blocking drift.
- `High`
  - Important mismatch or quality problem that should be addressed soon.
- `Medium`
  - Real issue with moderate impact.
- `Low`
  - Cleanup or localized improvement opportunity.

## Confidence rubric

- `High`
  - Directly supported by clear spec/code/test evidence.
- `Medium`
  - Likely issue, but some interpretation is involved.
- `Low`
  - Tentative hypothesis requiring more investigation.

## Evidence requirements

Every finding must include:

- stable finding ID
- finding type
- severity
- confidence
- concise summary
- why it matters
- spec reference(s) when applicable
- code reference(s)
- test reference(s) or an explicit note that test evidence is missing
- recommended action:
  - `Code fix`
  - `Test fix`
  - `Spec fix`
  - `Both`
  - `Follow-up investigation`

Do not include findings without evidence.

## Output rules by mode

### Mode: `explore`

Write exactly one markdown findings artifact to the provided `output_path`.

Requirements:
- use the required exploratory artifact template
- include only scoped findings
- do not synthesize other task outputs
- do not propose a full project plan
- do not create issues

### Mode: `synthesize`

Read all exploratory findings and produce the canonical final artifacts:
- canonical markdown report
- normalized JSONL findings index
- remediation planning brief

Requirements:
- deduplicate overlapping findings
- preserve source finding IDs and provenance
- group related findings into themes
- separate:
  - code changes
  - test changes
  - spec clarifications
  - follow-up investigations
- identify likely improvement clusters and dependency notes
- do not create issues

### Mode: `deck`

Generate one standalone HTML slide deck for human consumption from the synthesized artifacts.

Requirements:
- single self-contained HTML file
- inline CSS and JS only
- no external dependencies required
- readable in browser and printable to PDF
- summary-first, appendix-backed
- reflect the synthesized artifacts faithfully
- do not introduce new claims absent from the canonical analysis

## Stop conditions

Stop when:
- the required artifact has been written
- the scope has been reviewed sufficiently for the assigned task
- open questions have been recorded

Do not continue into implementation or issue planning unless explicitly instructed.

## Preferred style

- precise
- compact
- evidence-first
- minimal narrative
- bullets and tables over long prose
- direct file references over vague descriptions
- explicit uncertainty when needed

## Path conventions

Use run-scoped temp output paths under:

`tmp/project-analysis/<run-id>/`

Examples:
- `tmp/project-analysis/<run-id>/findings/A6-architecture-drift.md`
- `tmp/project-analysis/<run-id>/final/project-analysis-report.md`
- `tmp/project-analysis/<run-id>/final/project-analysis-findings.jsonl`
- `tmp/project-analysis/<run-id>/final/remediation-planning-brief.md`
- `tmp/project-analysis/<run-id>/deck/project-analysis.html`

Treat this temp area as non-canonical working output. The synthesized final artifacts are canonical for the review run.

## Quality bar

A good output:
- is tightly scoped
- is grounded in evidence
- distinguishes fact from interpretation
- separates drift from ambiguity
- is easy for a later agent to synthesize or plan from

A bad output:
- is speculative
- mixes implementation with review
- omits evidence
- sprawls beyond scope
- writes prose without actionable structure
