# Automated test catalog

Catalog is an opt-in index of implemented automated tests, not a test-result
ledger. `events.ndjson` remains the source of truth for test definitions, runs,
and outcomes. Consult and update this catalog only when the user asks to create
or add tests. For other work, use existing event-log rules without catalog work.

## When adding tests

1. Read `docs/testing/README.md`, this guide, and the relevant catalog table.
2. Inspect matching test definitions in `docs/testing/events.ndjson` to choose
   the next unused stable `T` ID. Filter by relevant area and read no more than
   15 matching entries.
3. Add the test under `tests/<domain>/`, following `AGENTS.md`.
4. Append one catalog row in the table matching the test type. Use the same `T`
   ID in the catalog and the `test` event. Add existing matrix IDs only when
   they directly cover the test; use references such as `PT-*` or `TDB-*`
   without creating new matrix IDs.
5. Append the test-definition event. When tests run, append linked `R` and `X`
   events as required by `AGENTS.md`. Do not record results in this catalog.

Classify isolated application/domain logic with test doubles as **unit**;
route and handler checks using `httptest` as **HTTP/API**; tests against a real
database or external adapter as **database integration**. Keep one table per
type and always record domain in its own column.

IDs are never reused. Keep existing rows stable; material test changes get a
new version or ID, in keeping with `docs/testing/README.md`.
If separate immutable `T` definitions cover different behaviors in one Go test
function, keep one catalog row per `T` ID and repeat the function name.

## Unit tests

|T ID|Domain|Test name|Purpose|Test file|Existing matrix ID(s)|
|---|---|---|---|---|---|
|T015|user-management|`TestUpdateUserRoleRequiresAdminAndUsesTargetUID`|Uses selected UID and rejects self-role changes.|`tests/auth/service_test.go`|PT-TR-01|
|T017|user-management|`TestUpdateUserRoleRequiresAdminAndUsesTargetUID`|Accepts canonical `profesor` role and case-insensitive admin actor.|`tests/auth/service_test.go`|PT-TR-01|
|T018|user-management|`TestUpdateUserRoleRejectsNonAdminAndInvalidRequests`|Rejects non-admin actor, missing UID, and unsupported role.|`tests/auth/service_test.go`|PT-TR-01|
|T019|user-management|`TestSearchUsersRequiresQueryAndBoundsLimit`|Requires nonblank query and clamps result limit.|`tests/auth/service_test.go`|PT-INT-02|

## HTTP/API tests

|T ID|Domain|Test name|Purpose|Test file|Existing matrix ID(s)|
|---|---|---|---|---|---|
|T016|user-management|`TestAdminMembersRejectsAuthenticatedNonAdmin`|Rejects administrative search by authenticated non-admin.|`tests/auth/http_test.go`|PT-TR-01|
|T020|user-management|`TestPublicMembersRequiresQueryAndOmitsPrivateFields`|Requires search query and excludes private profile fields.|`tests/auth/http_test.go`|PT-INT-02, TDB-T09|
|T021|user-management|`TestAdminMembersRequiresSessionAndReturnsAdminFields`|Requires session and returns restricted fields to admin.|`tests/auth/http_test.go`|PT-TR-01, TDB-T09|
|T022|user-management|`TestUpdateRoleAcceptsHeaderAndPathTargets`|Accepts selected UID from header or resource path.|`tests/auth/http_test.go`|PT-TR-01|
|T023|user-management|`TestUpdateRoleRejectsUnauthenticatedAndConflictingTargets`|Rejects missing session and conflicting target UIDs.|`tests/auth/http_test.go`|PT-TR-01|

## Database integration tests

|T ID|Domain|Test name|Purpose|Test file|Existing matrix ID(s)|
|---|---|---|---|---|---|
|T024|user-management|`TestSearchReturnsOnlyActiveMatchingUsers`|Searches name/email and excludes inactive users.|`tests/auth/user_repository_test.go`|PT-INT-02, TDB-T12|
|T025|user-management|`TestUpdateRoleUpdatesActiveUserAndRejectsInactiveOrMissingUID`|Updates active user and rejects inactive or missing UID.|`tests/auth/user_repository_test.go`|PT-TR-01, TDB-T05|

For a new test type, add a table with the same columns. Keep domain as a column
in every table so tests remain filterable by functionality.
