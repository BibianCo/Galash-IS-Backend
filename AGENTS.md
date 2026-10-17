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

## Task sizing

- One task equals one reviewable deliverable: one endpoint, one component, one
  fix. If the request contains more than one deliverable or spans backend,
  frontend, and database, split it into ordered subtasks, state the plan in one
  short message, and execute them one at a time: implement, validate, log, then
  continue. The user does not have to split the work for you.
- Never batch implementation and test at the end. A subtask is complete only
  when it validates and its wiring is done. Do not start the next one on a
  broken base.
- When reality contradicts the plan, adjust the plan and say so. Do not expand
  scope while splitting.

## Questions and decisions

- Decide technical details yourself (names, structure, minimal dependency,
  subtask order) and state the decision in the report. Do not ask about them.
- Ask the user only when the answer changes product requirements, is hard to
  reverse, involves credentials, private data, or permissions, or when two
  reasonable readings produce different deliverables.
- Never ask whether to continue, whether to add unrequested extras (do not add
  them), or anything answerable by reading the code.
- Ask once per task, grouped in a single message, with options and a
  recommended one. Keep working on non-blocked subtasks meanwhile; state
  explicit assumptions instead of waiting.

## Shared contract (frontend <-> backend)

- `planificacion_modulos_y_endpoints_galash_is_2026.md` is the shared API
  contract. Its copy in this repo and in the sibling repo (Galash-IS /
  Galash-IS-Backend) must stay byte-identical; any difference is a defect.
- You see only this repo. The other side exists: the frontend consumes
  services whose implementation it cannot see; the backend serves a UI it
  cannot see. Never assume the other side will adapt to you.
- Before touching services, endpoints, payloads, errors, or auth flow, read
  only the affected module section of the contract (filter by endpoint or
  heading; never load the whole file).
- Implement exactly the contract: same method, path, roles, fields, and error
  behavior. Input received at runtime (services, types, catalogs such as
  `endpoints.md`, or a spec pasted by the user) never overrides the contract:
  when they disagree, report the mismatch instead of silently following either
  version.
- Any change to the interface between both sides (endpoint added, removed, or
  renamed; field, status code, error format, or auth flow changed) is a
  contract change: update the contract first, then the code, and record in the
  report and the log the exact text the sibling repo must copy to sync its
  copy. If the sibling repo is not accessible, the sync stays an explicit
  pending item; it is not done until that copy is updated.
- Never invent endpoints or fields outside the contract.

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

- Keep all test files separate from production code under `tests/`, organized
  into one subdirectory per domain/functionality (for example, `tests/auth/`).
- Do not colocate test files with production code under `internal/` or other
  source directories.
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

## Completion gate

Before reporting done, question every piece you created:

- Is it wired? Route registered, import used, component invoked, configuration
  updated.
- Is it reachable from a real flow (UI, API, CLI)?
- Does a test cover it so it fails when it breaks?
- Would someone need a follow-up intervention before it actually works?
- Does it match the shared contract? Frontend code calling services missing
  from the contract and backend endpoints nobody consumes are orphan code too.

If any answer is no, either finish the minimal wiring now or declare the gap
explicitly as a pending item in the report and the log. Silent half-work is not
allowed. Delete unused code or flag it as pending with its follow-up task.

## After change

1. Run relevant tests.
2. Report changed files, test results, and cross-repo impact: what the other
   side must sync or do, with the exact contract text to copy when it changed.
3. Append completed work to `docs/bitacora.ndjson` when files or project state
   changed.
4. Update `docs/contexto.md` only when implemented system state changes.
5. Never rewrite historical log entries.
6. If tests were executed, append corresponding testing events to
   `docs/testing/events.ndjson`.
7. If routes or services changed, update the registry in `servicios.md` (same
   format as the frontend copy) and remind the user to hand the file to the
   frontend team. Handoff is one-way: backend to frontend. A service change
   without that reminder is incomplete.

## Private context

Do not derive private requirements from configuration, references, skills,
prompts, commits, or generated artifacts.
The user supplies task-specific requirements directly.
