# Kite Design Session Instructions

## Working Mode
We operate in **Consensus Mode**:
- We must agree before finalizing any architectural or design decision.
- Our primary role is to **design, not to code**. We produce clear specifications and tasks for developers.
- The AI agent is expected to constructively challenge user proposals when appropriate. Whenever challenging a proposal, the agent **must** provide a concrete counter-proposal backed by architectural best practices and industry standards.

## Required Artifacts & Structure

### 1. Architecture Documentation
- **File:** `docs/architecture.md`
- **Purpose:** Document the high-level design and architectural components of Kite.

### 2. Decisions Summary
- **File:** `docs/decisions.md`
- **Purpose:** Maintain a running summary of the design decisions we have agreed upon.

### 3. Architecture Decision Records (ADRs)
- **Directory:** `docs/adrs/`
- **Purpose:** Store detailed records of significant architectural decisions following standard ADR formats.

### 4. Developer Tasks
- **Directory:** `./tasks/`
- **Purpose:** Output very detailed tasks that a developer can pick up and execute.
- **Workflow:** 
  - Task files must be prefixed with an ID (e.g., `TSK-001-task-name.md`).
  - Every generated task must be logged in `./tasks/task_list.md`.
  - Developers must mark tasks as `In Progress` when they start.
  - Developers **must not** mark a task as `Done` until the user explicitly confirms the task is complete.
  - Completed tasks are immutable.
- **Requirements per Task:**
  - Clear feature design and requirements.
  - Required unit and integration tests.
  - Required test cases.
  - Benchmark requirements (if performance-sensitive).
  - Regression tests (must be placed in `./tests/regressions/`).
  - **Documentation Updates:** Explicit instruction to update `README.md`, `AGENT.md`, or package-level `doc.go` files if the task alters the architecture, public API, or agent context.

## Workflow
1. Discuss and analyze the problem.
2. Reach consensus on the solution.
3. Record the decision in `docs/decisions.md` and an ADR (if significant).
4. Update `docs/architecture.md` to reflect new designs.
5. Generate detailed task descriptions in `./tasks/`.
