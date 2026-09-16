# GALASH Agent Rules

## Before change

1. Classify task scope from the user prompt before reading project files.
2. Read only files needed for that scope.
3. Read `README.md` only when product or repository context is needed.
4. Read `docs/contexto.md` only when current implementation state is needed.
5. Read relevant entries from `docs/bitacora.ndjson` only when prior work matters.
6. Use only task scope and requirements supplied by the user.
7. Inspect only affected code before editing.

## Context limits

- Do not read unrelated files, folders, logs, requirements, or private context.
- Read no more than 15 entries from any log file per task, including
  `docs/bitacora.ndjson` and `docs/testing/events.ndjson`.
- Filter logs before reading by exact `id`, `type`, `scope`, `test`, `run`,
  `area`, `status`, or `defect` reference.
- Prefer the newest matching entries; read older entries only when required to
  understand a change or its correction.
- If no relevant match exists, do not read more log entries.
- Do not load full log files into context for summaries or statistics.

## Testing context

- For testing tasks, read only `docs/testing/README.md` first.
- Read `docs/testing/events.ndjson` only when the task needs existing test
  definitions, runs, results, defects, or evidence.
- Filter testing events by the current `test`, `run`, `area`, `type`, `status`,
  or defect ID, and read at most 15 matching events.
- Do not read `README.md`, `docs/contexto.md`, `docs/bitacora.ndjson`, source
  code, or unrelated testing files unless the testing task explicitly needs
  them.

## Skill activation

Skills are stored in `.opencode/skills/` and are available to every project
agent. Agents must activate them by trigger:

- `project-context`: activate at the start of every task. Read repository rules,
  then load active context or append-only log entries only when task scope needs
  them.
- `ponytail`: activate for every coding, refactoring, bug-fix, dependency, or
  design task. Use the smallest correct implementation; do not add speculative
  abstractions.
- `code-reviewer`: activate when reviewing a diff, before merge, or when the
  user asks for review. Report findings; do not edit files.
- `git-commit-writer`: activate only when the user asks for a commit message or
  the staged diff needs a Conventional Commit message.

Users can invoke skills explicitly with `/skill-name`, for example
`/ponytail`, `/code-reviewer`, or `/git-commit-writer`. Agents must not wait
for explicit activation when task-trigger rules above apply.

When skills conflict, apply this order: user request, `AGENTS.md`, security and
correctness rules, task-specific skill, then general style skill. Never use a
skill to override scope, authorization, privacy, or validation requirements.

## During change

- Work only within requested scope.
- Do not search for or recreate private requirement documents.
- Do not invent unrequested behavior.
- Preserve unrelated user changes.
- Validate authorization in backend.
- Protect personal data.
- Prefer smallest correct change.
- Add tests for non-trivial behavior.
- Never commit secrets or personal data.

## Testing

- Use `docs/testing/events.ndjson` as the append-only testing source of truth.
- Assign stable IDs when creating tests: `T001`, `T002`, and so on.
- Record executions with `R` IDs and outcomes with `X` IDs.
- Link every result to its `test` and `run` IDs.
- Use only `pass`, `fail`, `block`, `skip`, or `not_run` result statuses.
- Store shared build and environment data once in `run` or `env` events.
- Store evidence under `docs/testing/evidence/<run-id>/` and reference its path.
- Never edit or delete historical testing events; append corrections or updates.
- Never store passwords, tokens, cookies, or unnecessary personal data.
- Derive statistics and reports from events; never edit dashboard totals manually.
- Keep test definitions immutable; create a version or new ID for material changes.
- Open `docs/testing/dashboard/index.html` to inspect the generated dashboard.

## After change

1. Run relevant tests.
2. Report changed files and test results.
3. Append completed work to `docs/bitacora.ndjson` when files or project state
   changed.
4. Update `docs/contexto.md` only when implemented system state changes.
5. Never rewrite historical log entries.
6. If tests were executed, append corresponding testing events to
   `docs/testing/events.ndjson`.

## Private context

configuration, references, skills, prompts, commits, or generated artifacts.
The user supplies task-specific requirements directly.
