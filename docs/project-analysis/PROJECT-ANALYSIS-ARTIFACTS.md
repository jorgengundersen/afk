# PROJECT-ANALYSIS artifact pipeline spec

This document defines the artifact contract for the post-v1 project analysis workflow.

Canonical tracked assets:
- `docs/project-analysis/PROJECT-ANALYSIS.md`
- `docs/project-analysis/PROJECT-ANALYSIS-ARTIFACTS.md`

Run-scoped analysis outputs are temporary and belong under `tmp/project-analysis/<run-id>/`.

The workflow has three phases:
1. exploratory audit passes
2. synthesis into canonical AI-friendly artifacts
3. generation of a human-facing HTML slide deck

## Run layout

```text
tmp/project-analysis/<run-id>/
  00-manifest.md
  01-baseline.md
  02-audit-map.md
  findings/
    A4-product-drift.md
    A5-design-drift.md
    A6-architecture-drift.md
    A7-standards-conformance.md
    A8-test-quality.md
    A9-code-health.md
    A10-operational-readiness.md
    A11-security-privacy.md
    A12-performance-scalability.md
    A13-accessibility-docs-change-safety.md
  final/
    project-analysis-report.md
    project-analysis-findings.jsonl
    remediation-planning-brief.md
  deck/
    project-analysis.html
```

---

## 1. Manifest schema

### File
`tmp/project-analysis/<run-id>/00-manifest.md`

### Purpose
Declares the run and gives all agents a shared baseline.

### Template

```md
---
run_id: <run-id>
project: <project-name>
baseline: <commit-sha-or-branch>
generated_at: <ISO-8601 timestamp>
status: in_progress
---

# Analysis run manifest

## Baseline
- Commit: `<sha>`
- Branch or ref: `<ref>`
- Analysis date: `<date>`

## Active specs
- `specs/index.html`
- `specs/prd-afk.html`
- `specs/design-principles.html`
- `specs/architecture.html`
- `specs/coding-testing-standards.html`

## Planned audit tasks
- A4 product drift
- A5 design drift
- A6 architecture drift
- A7 standards conformance
- A8 test quality
- A9 code health
- A10 operational readiness
- A11 security/privacy
- A12 performance/scalability
- A13 accessibility/docs/change-safety

## Output locations
- Findings: `tmp/project-analysis/<run-id>/findings/`
- Final: `tmp/project-analysis/<run-id>/final/`
- Deck: `tmp/project-analysis/<run-id>/deck/`

## Notes
- Analysis only
- No issue filing
- No `ralph` label unless explicitly instructed
```

---

## 2. Exploratory findings schema

### File pattern
`tmp/project-analysis/<run-id>/findings/<task-id>-<slug>.md`

### Purpose
One scoped audit pass by one loop.

### Template

```md
---
task_id: A6
task_title: Audit architecture drift vs architecture spec
mode: explore
run_id: <run-id>
baseline: <commit-sha>
specs_read:
  - specs/architecture.html
code_areas:
  - <path-or-module>
test_areas:
  - <path-or-suite>
status: complete
---

# Scope reviewed
- <specific modules, files, flows>
- <specific tests or suites>

# Audit method
- <brief method summary>

# Findings

## A6-F1
- Type: Drift
- Severity: High
- Confidence: High
- Summary: <one-sentence finding>
- Why it matters:
  - <impact>
- Spec references:
  - `specs/architecture.html §<section>`
- Code references:
  - `<path>`
  - `<path>`
- Test references:
  - `<path>`
  - `Missing`
- Recommended action: Code fix

## A6-F2
- Type: Quality risk
- Severity: Medium
- Confidence: Medium
- Summary: <...>
- Why it matters:
  - <...>
- Spec references:
  - `specs/architecture.html §<section>` | `None`
- Code references:
  - `<path>`
- Test references:
  - `Missing`
- Recommended action: Follow-up investigation

# Open questions
- <question>
- <question>

# Coverage notes
## Reviewed
- <what was covered>

## Not reviewed
- <what was intentionally out of scope or not reached>

# Summary
- Total findings: <n>
- Critical: <n>
- High: <n>
- Medium: <n>
- Low: <n>
```

### Rules
- exactly one task per file
- stable IDs like `A6-F1`, `A6-F2`
- no cross-task synthesis
- no implementation plan
- no issue decomposition

---

## 3. Canonical final markdown report schema

### File
`tmp/project-analysis/<run-id>/final/project-analysis-report.md`

### Purpose
Primary human/AI-readable synthesis.

### Template

```md
---
run_id: <run-id>
mode: synthesize
baseline: <commit-sha>
generated_at: <ISO-8601 timestamp>
source_findings:
  - findings/A4-product-drift.md
  - findings/A5-design-drift.md
  - findings/A6-architecture-drift.md
  - findings/A7-standards-conformance.md
  - findings/A8-test-quality.md
  - findings/A9-code-health.md
  - findings/A10-operational-readiness.md
  - findings/A11-security-privacy.md
  - findings/A12-performance-scalability.md
  - findings/A13-accessibility-docs-change-safety.md
---

# Executive summary
- <3-7 bullets>

# Scope and method
- baseline reviewed
- specs reviewed
- audit tasks completed
- important limits or gaps

# Overall assessment
## Spec fidelity
## Architecture fidelity
## Test quality
## Code quality
## Operational quality
## Security/privacy
## Performance/scalability
## Accessibility/docs/change-safety

# Key findings by category
## Product drift
## Design/UX drift
## Architecture drift
## Standards conformance
## Test quality
## Code health
## Operational readiness
## Security/privacy
## Performance/scalability
## Accessibility/docs/change-safety

# Cross-cutting themes
## Theme 1
- related finding IDs
- explanation
- impact

## Theme 2
- related finding IDs
- explanation
- impact

# Drift register
| Finding ID | Type | Severity | Summary | Recommended action |
|---|---|---|---|---|

# Risk register
| Finding ID | Area | Severity | Summary | Why it matters |
|---|---|---|---|---|

# Spec ambiguities and gaps
| Finding ID | Spec reference | Issue | Recommended next step |
|---|---|---|---|

# Prioritized remediation opportunities
## Fix now
- <grouped items>

## Fix soon
- <grouped items>

## Later
- <grouped items>

## Spec clarification first
- <grouped items>

# Candidate work decomposition notes
## Likely epics
- <cluster>

## Likely leaf-task clusters
- <cluster>

## Dependency observations
- <what blocks what>

# Coverage limits and open questions
- <limits>
- <unknowns>

# Appendix: finding provenance
| Canonical ID | Source finding IDs | Source files |
|---|---|---|
```

This is the main report a future planning agent should read first.

---

## 4. Normalized findings JSONL schema

### File
`tmp/project-analysis/<run-id>/final/project-analysis-findings.jsonl`

### Purpose
Machine-friendly canonical finding index.

### Example object

```json
{
  "canonical_id": "PA-001",
  "source_ids": ["A6-F1", "A9-F2"],
  "run_id": "2026-05-15-v1-audit",
  "baseline": "<commit-sha>",
  "category": "architecture-drift",
  "type": "Drift",
  "severity": "High",
  "confidence": "High",
  "title": "UI layer reaches into persistence responsibilities",
  "summary": "Implementation bypasses intended boundary between UI and persistence layers.",
  "why_it_matters": [
    "Increases coupling",
    "Raises change risk for future feature work"
  ],
  "spec_refs": ["specs/architecture.html §3.2"],
  "code_refs": ["src/..."],
  "test_refs": ["tests/..."],
  "recommended_action": "Code fix",
  "theme_tags": ["boundaries", "coupling", "change-risk"],
  "followup_kind": "refactor",
  "priority_bucket": "fix-soon",
  "provenance_files": [
    "tmp/project-analysis/<run-id>/findings/A6-architecture-drift.md",
    "tmp/project-analysis/<run-id>/findings/A9-code-health.md"
  ],
  "open_questions": []
}
```

### Controlled values

`category`:
- `product-drift`
- `design-drift`
- `architecture-drift`
- `standards-conformance`
- `test-quality`
- `code-health`
- `operational-readiness`
- `security-privacy`
- `performance-scalability`
- `accessibility-docs-change-safety`

`type`:
- `Drift`
- `Missing`
- `Unspecified behavior`
- `Spec ambiguity`
- `Quality risk`

`recommended_action`:
- `Code fix`
- `Test fix`
- `Spec fix`
- `Both`
- `Follow-up investigation`

`followup_kind`:
- `bug-fix`
- `test-improvement`
- `refactor`
- `spec-clarification`
- `investigation`
- `operational-hardening`
- `performance-work`
- `security-hardening`
- `accessibility-fix`
- `documentation-fix`

`priority_bucket`:
- `fix-now`
- `fix-soon`
- `later`
- `spec-clarification-first`

---

## 5. Remediation planning brief schema

### File
`tmp/project-analysis/<run-id>/final/remediation-planning-brief.md`

### Purpose
Planner-ready handoff for a later agent to turn findings into an improvement plan or issue graph.

### Template

```md
---
run_id: <run-id>
baseline: <commit-sha>
input_report: final/project-analysis-report.md
input_findings: final/project-analysis-findings.jsonl
---

# Remediation planning brief

## Purpose
This brief translates the project analysis into planning-ready improvement clusters.

## Planning constraints
- Preserve active spec alignment
- Separate code fixes from spec clarifications
- Prefer small leaf tasks with observable outcomes
- Keep tasks parallel unless true blockers exist
- Do not create separate "write tests" tasks; combine tests with behavior changes where appropriate

## Improvement clusters

### Cluster 1: <name>
- Problem summary:
- Relevant canonical findings:
  - PA-001
  - PA-004
- Why these belong together:
- Likely work type:
  - refactor / bug fixes / test improvements / spec clarifications
- Suggested sequencing:
- Potential blockers:
- Human decisions needed:

### Cluster 2: <name>
...

## Recommended epic candidates
- <epic candidate>
- <why it is an epic>
- <included findings>

## Recommended standalone task candidates
- <task candidate>
- <included findings>

## Spec clarification candidates
- <questions needing product/engineering alignment before implementation>

## Suggested sequencing
1. <first wave>
2. <second wave>
3. <third wave>

## Notes for planning agent
- areas likely to parallelize well
- areas that should stay grouped
- risky decompositions to avoid
```

---

## 6. HTML slide deck schema

### File
`tmp/project-analysis/<run-id>/deck/project-analysis.html`

### Purpose
Human-facing summary and presentation artifact.

### Requirements
- single HTML file
- self-contained
- inline CSS/JS only
- usable without build tools
- keyboard navigable
- print/PDF friendly
- consistent with synthesized canonical report
- summary-first, appendix-backed

### Suggested slide structure
1. Title
2. Executive summary
3. Scope and method
4. Overall assessment scorecard
5. Spec fidelity
6. UX/design fidelity
7. Architecture fidelity
8. Standards conformance
9. Test quality
10. Code health
11. Failure-path and operational readiness
12. Security/privacy
13. Performance/scalability
14. Accessibility/docs/change-safety
15. Cross-cutting themes
16. Prioritized remediation opportunities
17. Spec clarification needs
18. Appendix: finding index / evidence summary

### Important rule
The deck should be derived from the canonical report and findings index, not be the primary source of truth.

---

## Invocation templates

### Explore mode

```text
Use @docs/project-analysis/PROJECT-ANALYSIS.md.

Mode: explore
Run ID: <run-id>
Baseline: <commit-sha>
Task ID: A6
Specs to read:
- specs/architecture.html

Code areas:
- <paths>

Test areas:
- <paths>

Output path:
- tmp/project-analysis/<run-id>/findings/A6-architecture-drift.md

Constraints:
- analysis only
- do not modify code or tests
- do not file issues
- do not add the ralph label
- write exactly one findings artifact
```

### Synthesize mode

```text
Use @docs/project-analysis/PROJECT-ANALYSIS.md.

Mode: synthesize
Run ID: <run-id>
Baseline: <commit-sha>

Input paths:
- tmp/project-analysis/<run-id>/findings/A4-product-drift.md
- tmp/project-analysis/<run-id>/findings/A5-design-drift.md
- tmp/project-analysis/<run-id>/findings/A6-architecture-drift.md
- tmp/project-analysis/<run-id>/findings/A7-standards-conformance.md
- tmp/project-analysis/<run-id>/findings/A8-test-quality.md
- tmp/project-analysis/<run-id>/findings/A9-code-health.md
- tmp/project-analysis/<run-id>/findings/A10-operational-readiness.md
- tmp/project-analysis/<run-id>/findings/A11-security-privacy.md
- tmp/project-analysis/<run-id>/findings/A12-performance-scalability.md
- tmp/project-analysis/<run-id>/findings/A13-accessibility-docs-change-safety.md

Output paths:
- tmp/project-analysis/<run-id>/final/project-analysis-report.md
- tmp/project-analysis/<run-id>/final/project-analysis-findings.jsonl
- tmp/project-analysis/<run-id>/final/remediation-planning-brief.md

Constraints:
- synthesis only
- no issue filing
- no labels
- preserve provenance
```

### Deck mode

```text
Use @docs/project-analysis/PROJECT-ANALYSIS.md.

Mode: deck
Run ID: <run-id>
Baseline: <commit-sha>

Input paths:
- tmp/project-analysis/<run-id>/final/project-analysis-report.md
- tmp/project-analysis/<run-id>/final/project-analysis-findings.jsonl
- tmp/project-analysis/<run-id>/final/remediation-planning-brief.md

Output path:
- tmp/project-analysis/<run-id>/deck/project-analysis.html

Constraints:
- human-facing summary deck
- single self-contained HTML file
- no external dependencies
- no new claims beyond synthesized artifacts
```
