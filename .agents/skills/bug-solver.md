---
apiVersion: warp/v1alpha1
kind: Skill
metadata:
  name: bug-solver
  description: A structured, test-driven approach to identifying, reproducing, and fixing bugs with regression and unit tests.
  displayName: Bug Solver
---

# Bug Solver Skill

This skill provides a structured, test-driven methodology for identifying, reproducing, and fixing bugs within the Kite framework. Following this process ensures that bugs are resolved at their root cause and that future regressions are prevented.

## Structured Method for Bug Resolution

When assigned a bug-related task, follow these steps in order:

### 1. Investigation & Initial Analysis
*   Analyze the reported issue and locate the relevant components in the codebase.
*   Trace the data flow and logic paths associated with the bug.
*   Identify whether the issue is structural (DOM), layout-driven, or related to the rendering/event pipeline.

### 2. Regression Testing (Reproduction)
*   **Goal**: Create a failing test case that mirrors the user's reported scenario.
*   **Action**: Write a high-level integration or regression test (typically in `tests/regressions/`) that fails in the current codebase.
    *   **Tip**: Use `devtools/testenv` to ergonomically simulate the user's interaction (typing, clicking) in a headless environment. See the `kite-testing` skill for details.
*   **Verification**: Run the test to confirm it fails with the expected symptom. This serves as your "Red" state in TDD.

### 3. Unit Testing
*   **Goal**: Isolate the specific low-level logic causing the bug.
*   **Action**: Identify the exact function or method at fault and write focused unit tests for it.
*   **Breadth**: Cover edge cases related to the bug (e.g., empty inputs, zero offsets, boundary conditions).
*   **Verification**: Ensure these unit tests also fail before any code changes are made.

### 4. Implementation (The Fix)
*   **Goal**: Resolve the bug while maintaining architectural integrity.
*   **Action**: Apply the minimal necessary changes to fix the logic identified in Step 3.
*   **Constraint**: Do not modify application code (examples/demos) unless explicitly instructed; the fix should live in the framework logic.

### 5. Verification & Validation
*   **Goal**: Confirm the fix and ensure no collateral damage.
*   **Action**: 
    1. Run the newly created regression and unit tests. They must now pass ("Green" state).
    2. Ensure the regression test is integrated into the permanent test suite (e.g., `tests/regressions/`). **Regression tests are the mandatory final artifact of every bugfix.**
    3. Run the entire test suite (`go test ./...`) to ensure no other part of the system is broken.
    4. Manually verify the fix by running relevant examples if applicable.

## Guidelines
*   **No Guessing**: Always verify assumptions by reading code or running targeted probe tests.
*   **Permanent Regressions**: Every bugfix must conclude with a permanent regression test added to the codebase to prevent the issue from re-emerging.
*   **Test Quality**: Write clear, descriptive test names and include comments explaining what the test is verifying and why.
*   **Documentation**: If the fix involves a significant architectural change or touches a core invariant, update relevant ADRs or package documentation.
*   **Cleanup**: Remove any temporary "repro" files created during investigation before finishing the task.
