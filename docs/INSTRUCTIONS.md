# Kite Design Session Instructions

## Working Mode
We operate in **Consensus Mode**:
- We must agree before finalizing any architectural or design decision.
- **Do not update or create any documentation, ADRs, or task files until the user explicitly agrees to the proposed design.**
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
  - Create a sample implementation in `./examples/` if the task introduces new elements or APIs that would benefit from demonstration.
  - **Documentation Updates:** Explicit instruction to update `README.md`, `AGENT.md`, or package-level `doc.go` files if the task alters the architecture, public API, or agent context.

## Workflow
1. Discuss and analyze the problem. Proactively ask follow-up questions to clarify doubts, uncover edge cases, and ensure a complete understanding of the requirements.
2. Reach consensus on the solution.
3. Wait for the user to explicitly confirm their agreement with the final design.
4. Record the decision in `docs/decisions.md` and create an ADR in `docs/adrs/` (if significant). Verify these files exist before proceeding.
5. Update `docs/architecture.md` to reflect new designs.
6. Generate detailed task descriptions in `./tasks/`.

## Strict Execution Constraints
To prevent skipped steps and ensure proper documentation, the agent MUST adhere to the following execution rules:
- **Sequential Enforcement:** Steps 4, 5, and 6 must be executed strictly in order. You must not generate the developer task (Step 6) until the documentation (Steps 4 and 5) has been successfully written to the file system.
- **Verification Before Progression:** Do not mark a workflow step as `completed` in the `todos` tool until the corresponding file operation (`fs_write` or `fs_edit`) has successfully executed and returned a result.
- **No Batching Artifacts:** Treat the documentation updates (Decisions, ADR, Architecture) as a prerequisite blocker for the Task generation.
- **Strict Code Verification:** The agent MUST NOT guess or assume Go types, function signatures, or struct names. Before writing any documentation or tasks that reference code, the agent MUST use search tools to verify the exact definitions in the `.go` source files.
- **No Destructive Overwrites:** When updating existing documentation (`architecture.md`, `decisions.md`, `task_list.md`), the agent MUST append or inject content using targeted replacements (`fs_edit`). The agent MUST NEVER overwrite an entire file using `fs_write` unless the file is entirely new.
