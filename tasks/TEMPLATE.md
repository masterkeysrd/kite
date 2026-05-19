# Task: [Task Name]

## 1. Objective
Briefly describe what needs to be implemented or changed.

## 2. Design & Requirements
- **Feature Design:** Detailed description of how the feature works conceptually based on design sessions.
- **Rules:**
  - [List any strict architectural rules, e.g., package isolation]
  - [List any edge cases to handle]

## 3. Implementation Steps
1. Step one...
2. Step two...

## 4. Testing Requirements
### 4.1. Unit Tests
- [ ] Test case 1: (Describe scenario and expected outcome)
- [ ] Test case 2: ...

### 4.2. Integration Tests
- [ ] Verify workflow X end-to-end...

### 4.3. Regression Tests (at `./tests/regressions/`)
- [ ] Add a regression test for [specific scenario] to ensure future changes do not break this behavior.

### 4.4. Benchmarks
- [ ] (If applicable) Benchmark the layout/render loop performance for this new feature. Needs to maintain 60FPS.

### 4.5. Documentation
- [ ] Update `README.md` and/or `AGENT.md` if this task introduces new public components or alters architectural guidelines.
- [ ] Update relevant `doc.go` files in modified packages.
