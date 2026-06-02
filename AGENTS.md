# MiniPost Agent Instructions

These instructions apply to all automated coding work in this repository.

## Work Routine

- Before changing code, write a TODO list and split the request into concrete development tasks.
- Read the relevant code first, then make narrowly scoped changes that match the existing project style.
- Keep every task small enough to review and verify independently.
- Do not leave unrelated refactors, formatting churn, or generated noise in the diff.

## Logging

- When adding or changing runtime behavior, add useful logs at important success and failure points.
- Use the repository's existing logging helpers and style instead of ad hoc `fmt.Println`, `console.log`, or temporary debug output.
- Logs should explain what happened and include safe context, but must not expose secrets, tokens, cookies, passwords, or request bodies that may contain private data.

## Tests

- Add or update tests for every code change that affects behavior.
- Cover the primary path, important edge cases, and regressions related to the change.
- Run the focused tests first, then the broader project checks before finishing.

## Build And Launch

- After code changes, build a fresh app instance instead of relying on an old build artifact.
- Before launching the new instance, close all existing MiniPost processes from previous runs.
- Start the freshly built instance and perform a quick smoke check for the touched workflow.
- Keep exactly one MiniPost instance running after launch; if multiple instances are detected, close the older ones and leave only the new instance.
- If a full desktop launch is not possible in the current environment, record the reason and run the closest available verification command.

## Git

- Check `git status` before and after changes.
- Do not revert user changes unless the user explicitly asks.
- Commit after each completed modification set, once tests and build checks pass.
- Use concise commit messages that describe the user-visible change.
