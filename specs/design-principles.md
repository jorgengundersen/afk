# Design Principles

Guiding star for this project. Defines _how_ things should be built, not
_what_ to build. Product specs define the what.

This codebase is developed primarily by AI agents. Humans steer and review.
Agents can't infer intent — implicit conventions become gaps where assumptions
compound. These principles exist to make the codebase self-documenting so
agents produce consistent, high-quality work.

## 1. Primitives and Composition

Build from small, composable units. Each unit has a clear contract
(inputs → outputs), a single responsibility, and is independently testable.

**Pure functions** are the foundation — no side effects, deterministic,
trivial to test. Prefer pure functions wherever possible.

**The "and" test:** if you need "and" to describe what a unit does, it's two
units. "Validate config" is one job. "Parse flags and validate config" is two.

**Primitives do. Orchestration decides.** A primitive transforms data. An
orchestrator sequences work and makes decisions. Keep them separate.

**Entry points are thin:** parse input, call composed units, surface results.
No business logic in entry points.

**Naming:** domain-first. Name for what it does, not how. Poor naming causes
duplication — an agent that can't find a function writes a new one.

## 2. Testing

The agent's primary self-verification mechanism. Weak tests mean agents can't
self-verify and humans must review everything.

**Black-box by default.** Test contracts, not internals. Litmus test: can you
refactor internals without breaking tests? If no, the tests are wrong.

**Test boundaries:**

- **Unit** (fast, the majority): known inputs, expected outputs. Test each
  unit's own contract, not the units it calls internally.
- **Integration** (fewer): verify that boundaries work — wrappers interact
  correctly with external systems.

**Tests are executable documentation.** An agent reading tests should learn
how a module works faster than reading comments. Write tests that demonstrate
the contract clearly.

## 3. Build vs Depend

AI-led development lowers the cost of building. Don't depend by default.

**Decision procedure:**

1. Does the standard library cover it? Use it.
2. Is correctness hard-won or security-critical? Depend.
3. How much of the library would you actually use? If 10%, build the 10%.
4. What's the transitive dependency cost?
5. Can an agent implement it in a sitting? Then build.

**When in doubt, own it.** Every dependency is code you don't control.

## 4. Error Handling

**Fail fast, fail loudly.** If something is wrong, surface it immediately.
Silent failures propagate and surface far from the cause.

**Errors are part of the contract.** Not just "given X return Y" but "given X,
fail this specific way."

**Principles:**

1. Created at the failure point with sufficient context.
2. Wrapped as they propagate — each layer adds context.
3. Handled at entry points. Internal code propagates.
4. Propagate or handle, never both. Don't log an error and also return it.
5. Translate at wrapper boundaries — external errors don't leak into domain
   code.

**User-facing errors must be actionable.** Tell the user what's wrong and what
they can do about it.

## 5. Dependency Isolation

External dependencies enter through wrappers that translate the external world
into your own types and contracts. Domain code never sees dependency-specific
types or behavior.

**The hard rule:** dependency types do not cross into domain code.

**Why:** if dependency types are woven through domain code, switching
implementations means rewriting the domain. Isolation means:

- Domain depends on your contracts, not the dependency's API
- Replacing an implementation = rewriting one wrapper
- Testing domain code doesn't require the real dependency
- Dependency API changes have blast radius of one wrapper

**Wrappers are thin** — translate, don't decide. Logic stays in domain code.

**Extract interfaces when needed,** not upfront. The constraint is "dependency
types never appear in domain code." When domain code is about to import a
dependency, that's the signal to introduce a wrapper.

## 6. State and Configuration

**Push state to edges.** Core logic is stateless. Pure functions in, pure
functions out. State lives at boundaries.

**Sane defaults, full override.** Software should work out of the box.

**Only expose what the user has a reason to change.** Not every internal value
is configuration. If you can't explain to a user why they'd change a value,
it's code, not config.

## 7. Observability

**Structured events** not ad-hoc strings. Machine-parseable, consistent
fields: what happened, when, what context.

**Observe at edges:** pure functions are silent. Instrument where inputs
arrive, external systems are called, and errors are handled.

**Surface problems.** If something affects the user's workflow, they must
know immediately.

## 8. Incremental Complexity

Add abstraction only when the cost of not having it exceeds the cost of
introducing it.

- Start concrete. Introduce abstraction when a real need emerges.
- Three similar lines of code is better than a premature abstraction.
- Build what specs define, not a framework that could do anything.
- Prefer reversible decisions. Function before package.
- Never compromise dependency isolation in the name of concreteness.

## 9. Codebase is the Prompt

Every file an agent reads is context. Vague conventions produce vague output.
Inconsistent examples produce inconsistent output.

- **Signatures and types are self-documenting.** If a function requires
  explanation beyond its signature and a brief doc comment, the API is too
  clever.
- **Test files are executable examples.** Tests show how a module is intended
  to be used.
- **Consistent patterns across modules.** Knowledge of one module should
  transfer to the next.
- **Specs are part of the codebase.** They exist in the repo so agents read
  them as context.

**When the codebase sends conflicting signals:** follow the majority pattern,
file the inconsistency as a bug.

**Context degrades. Maintain it.** Conventions drift from reality. Stale
context is actively harmful — agents follow outdated patterns with full
confidence.

## Applying These Principles

**Common tensions:**

1. **Isolation (5) vs. incremental complexity (8).** Isolation wins. A wrapper
   feels like premature abstraction with only one implementation, but isolation
   is structural — the cost of not having it compounds silently.

2. **One job (1) vs. practical convenience.** If splitting a unit produces a
   piece with no independently testable contract, the split isn't earning its
   keep.

3. **Build vs. depend (3) vs. time.** When the standard library covers 90% of
   the need, build the last 10%. When it covers 10%, evaluate carefully.

**Decision checklist:**

1. One nameable job? If it needs "and," split it.
2. Can this be pure? If yes, make it one.
3. Build or depend? Ownership cost < dependency cost?
4. What's the blast radius? Change here breaks there = wrong boundary.
5. Error contract explicit? Caller knows how this fails?
6. Dependency isolated? Swappable without touching domain?
7. Would an agent find this? Discoverable name, clear contract?
8. Simplest thing that works? Abstraction not earning its keep = remove it.
