---
name: create-spec
description: Create a spec for a new module or feature. Use when scoping work before epic creation and implementation.
user-invocable: true
allowed-tools: Read, Write, Glob, Grep, Bash(bd *)
---

Create a spec for the given module, feature, or capability.

## What a spec is

A spec defines **what** to build and **why**. It is the contract that epics
decompose and implementing agents fulfill. A spec succeeds when an agent can
read it, build the right thing, and know when they're done — without asking
clarifying questions.

## Spec structure

Use this skeleton. Every section earns its place — omit nothing, add nothing.

```markdown
# <Module Name>

<One paragraph: what this module does and why it exists. No implementation
detail — a human should understand the purpose without reading further.>

## What this adds

<2-4 CLI examples showing user-visible behaviour. Concrete inputs and outputs.
This is the "demo" someone would run to verify it works.>

## Structure

<File paths. Where this code lives.>

## Contract

<Describe the public boundary in terms of behaviour and capabilities — what
this module accepts, returns, and guarantees. Name the package and its public
surface conceptually. Do NOT use language-specific code blocks for signatures;
the implementation agent owns concrete types, names, and function signatures.
Use code signatures only when another spec already references a specific type
and changing it would break a cross-module contract.>

### Behaviour rules

<Table or list of rules: given X, the module does Y. These define the contract
exhaustively. Frame as rules, not test cases — the implementing agent decides
how to test them.>

| Given | Then |
|-------|------|
| ...   | ...  |

### What this does NOT do

<Explicit boundaries. What belongs to other modules. What callers are
responsible for. Prevents scope creep during implementation.>

## Out of scope

<Future work that is intentionally deferred. Helps the implementing agent
avoid gold-plating.>

## Definition of done

<Observable outcomes. How to verify the module works. Keep to 3-5 bullet
points. Prefer "X passes" or "X produces Y" over vague statements.>
```

## Principles

**Own the contract, not the implementation.** Behaviour rules and capabilities
are spec territory. Concrete function signatures, type names, and code blocks
are not — the implementing agent owns those. Pseudocode, test tables, and
implementation steps are also out of scope. The implementing agent has TDD
discipline and the codebase — let them decide how to build and test.

**No caller wiring.** A spec describes its own module. How main.go or other
modules use it is their concern. "What main.go becomes" sections leak
implementation across boundaries.

**Behaviour rules, not test cases.** "When prompt is empty, return error" is
a rule. A test case table with setup/assertion columns signals the wrong
thing and over-constrains the implementing agent. The agent translates rules
into tests through TDD.

**Explicit boundaries prevent duplication.** The "What this does NOT do"
section is load-bearing. Without it, implementing agents add validation,
error handling, or features that belong to adjacent modules.

**Small surface, clear edges.** A spec that needs more than ~100 lines is
probably scoping too much. Apply the "and" test — if the module does X and Y,
consider two specs.

## Before writing

1. Read `specs/design-principles.md` — this is the guiding star for all specs.
   Every design decision in a spec should be traceable to these principles.
2. Read the overview spec (`specs/afk-overview.md`) for context.
3. Read adjacent specs to understand boundaries — what's already owned
   elsewhere.
4. Check existing code (`internal/`) to avoid speccing something that exists.

## After writing

1. Place the spec in `specs/<module-name>.md`.
2. Verify it against adjacent specs — no overlapping ownership.
3. The spec is ready for epic decomposition via `/create-epic`.
