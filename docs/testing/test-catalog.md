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
|T026|module-3-registration|`TestCreateRejectsMissingRequiredField`|Rejects a registration request when a required field is blank.|`tests/registration/service_test.go`|—|
|T027|module-3-registration|`TestCreateReturnsOpaqueTrackingToken`|Creates a Radicada request and returns a token different from its persisted hash.|`tests/registration/service_test.go`|—|
|T028|module-3-registration|`POST /registration-requests`|Persists a valid public registration against the active PostgreSQL schema.|Manual API smoke check|—|
|T029|module-3-registration|`PATCH /admin/registration-requests/{id}/status`|Allows an Admin to transition Radicada to En revisión and records the responsible user.|Manual API smoke check|—|
|T030|seedbeds|`TestCreateRejectsMissingRequiredFields`|Rejects seedbed without name, responsible or valid contact email.|`tests/seedbed/service_test.go`|PT-SEM-01|
|T031|seedbeds|`TestCreateRequiresActiveTeacherAsResponsible`|Rejects student, admin, inactive or missing responsible.|`tests/seedbed/service_test.go`|PT-SEM-01|
|T032|seedbeds|`TestCreateStartsAsDraftWithTrimmedData`|Creates seedbed in `Borrador` with trimmed data.|`tests/seedbed/service_test.go`|PT-SEM-01|
|T033|seedbeds|`TestVisibilityAllowsResponsibleOrAdminAndKnownStates`|Publishes, hides and inactivates by responsible or admin; rejects others and unknown states.|`tests/seedbed/service_test.go`|PT-SEM-01|
|T034|seedbeds|`TestAddMemberAcceptsOnlyActiveStudentsOrTeachers`|Links only active students or teachers, by responsible only.|`tests/seedbed/service_test.go`|PT-SEM-03|
|T035|seedbeds|`TestRemoveMemberRequiresResponsibleOrAdmin`|Rejects withdrawal by student; admin withdraws.|`tests/seedbed/service_test.go`|PT-SEM-03|
|T036|seedbeds|`TestListBoundsPagination`|Defaults page to 1 and clamps pageSize to 1..100.|`tests/seedbed/service_test.go`|—|
|T051|module-4-research|`TestActivityRequiresResponsibleTeacherOrAdmin`|Activity create/update only by responsible teacher or admin; rejects empty title and unknown status.|`tests/research/service_test.go`|PT-SEM-04|
|T052|module-4-research|`TestProjectProgressMustBeBetween0And100`|Project progress accepts 0..100 and minimum states only.|`tests/research/service_test.go`|PT-SEM-05|
|T053|module-4-research|`TestProjectUpdateRecordsActorAndChangedFields`|PATCH /projects/{id} records actor and each changed relevant field.|`tests/research/service_test.go`|PT-SEM-05|
|T054|module-4-research|`TestProgressOnlyByResponsibleOrAdmin`|Progress update rejected for non-responsible teacher and student.|`tests/research/service_test.go`|PT-SEM-07|
|T055|module-4-research|`TestProposalCreateAlwaysStartsRadicada`|New proposal is always Radicada regardless of requested status.|`tests/research/service_test.go`|PT-SEM-06|
|T056|module-4-research|`TestProposalAuthorCannotApproveOrReview`|Proposal author edits content only; reviewer sets status/observations; manual Convertida rejected.|`tests/research/service_test.go`|PT-SEM-06|
|T057|module-4-research|`TestProposalDetailIncludesHistory`|Proposal detail returns review history; stranger is forbidden.|`tests/research/service_test.go`|PT-SEM-06|
|T058|module-4-research|`TestConvertRequiresApprovedProposalAndReviewer`|Conversion requires approved proposal and teacher/admin.|`tests/research/service_test.go`|PT-SEM-06|
|T059|module-4-research|`TestResearchJSONUsesCamelCase`|Module 4 payloads serialize with camelCase keys.|`tests/research/service_test.go`|PT-SEM-04, PT-SEM-05, PT-SEM-06, PT-SEM-07|

## HTTP/API tests

|T ID|Domain|Test name|Purpose|Test file|Existing matrix ID(s)|
|---|---|---|---|---|---|
|T016|user-management|`TestAdminMembersRejectsAuthenticatedNonAdmin`|Rejects administrative search by authenticated non-admin.|`tests/auth/http_test.go`|PT-TR-01|
|T020|user-management|`TestPublicMembersRequiresQueryAndOmitsPrivateFields`|Requires search query and excludes private profile fields.|`tests/auth/http_test.go`|PT-INT-02, TDB-T09|
|T021|user-management|`TestAdminMembersRequiresSessionAndReturnsAdminFields`|Requires session and returns restricted fields to admin.|`tests/auth/http_test.go`|PT-TR-01, TDB-T09|
|T022|user-management|`TestUpdateRoleAcceptsHeaderAndPathTargets`|Accepts selected UID from header or resource path.|`tests/auth/http_test.go`|PT-TR-01|
|T023|user-management|`TestUpdateRoleRejectsUnauthenticatedAndConflictingTargets`|Rejects missing session and conflicting target UIDs.|`tests/auth/http_test.go`|PT-TR-01|
|T037|seedbeds|`TestCreateSeedbedRequiresTeacherOrAdminSession`|`POST /seedbeds` returns 401/403/400 and 201 with `id`, `Borrador`, `responsibleTeacherId`.|`tests/seedbed/http_test.go`|PT-SEM-01|
|T038|seedbeds|`TestListSeedbedsRestrictsPrivateViewToAdmin`|`includeInactive=true` only for admin.|`tests/seedbed/http_test.go`|PT-SEM-02|
|T039|seedbeds|`TestGetSeedbedRespectsVisibility`|Hidden seedbed visible only to responsible or admin.|`tests/seedbed/http_test.go`|PT-SEM-01, PT-SEM-02|
|T040|seedbeds|`TestSeedbedSpaceHidesPrivateData`|Public space omits responsible email; hidden space only for managers.|`tests/seedbed/http_test.go`|PT-SEM-02|
|T041|seedbeds|`TestUpdateSeedbedRequiresResponsibleOrAdmin`|`PATCH /seedbeds/{id}` rejects anonymous and non-responsible.|`tests/seedbed/http_test.go`|PT-SEM-01|
|T042|seedbeds|`TestSetSeedbedVisibilityValidatesActorAndState`|Visibility change validates actor and state.|`tests/seedbed/http_test.go`|PT-SEM-01|
|T043|seedbeds|`TestSeedbedMemberHistoryIsRestricted`|Withdrawal history only for responsible or admin; hidden members not exposed.|`tests/seedbed/http_test.go`|PT-SEM-02, PT-SEM-03|
|T044|seedbeds|`TestLinkAndWithdrawMemberRequireResponsibleOrAdmin`|Link and withdraw member require responsible or admin.|`tests/seedbed/http_test.go`|PT-SEM-03|

## Database integration tests

|T ID|Domain|Test name|Purpose|Test file|Existing matrix ID(s)|
|---|---|---|---|---|---|
|T024|user-management|`TestSearchReturnsOnlyActiveMatchingUsers`|Searches name/email and excludes inactive users.|`tests/auth/user_repository_test.go`|PT-INT-02, TDB-T12|
|T025|user-management|`TestUpdateRoleUpdatesActiveUserAndRejectsInactiveOrMissingUID`|Updates active user and rejects inactive or missing UID.|`tests/auth/user_repository_test.go`|PT-TR-01, TDB-T05|
|T045|seedbeds|`TestLiveCreateRejectsDuplicateName`|Running API rejects duplicate seedbed name with 409.|`tests/seedbed/live_api_test.go`|PT-SEM-01|
|T046|seedbeds|`TestLiveUnknownSeedbedReturnsNotFound`|Running API returns 404 for unknown seedbed on detail, space and members.|`tests/seedbed/live_api_test.go`|PT-SEM-02|
|T047|seedbeds|`TestLiveVisibilityControlsPublicAccess`|Publish, hide and inactivate change public detail and list; admin sees inactive.|`tests/seedbed/live_api_test.go`|PT-SEM-01, PT-SEM-02|
|T048|seedbeds|`TestLiveHiddenMembersAnonymousSingleError`|Hidden seedbed members without session returns one 401 error body.|`tests/seedbed/live_api_test.go`|PT-SEM-02|
|T049|seedbeds|`TestLiveWithdrawKeepsHistory`|Withdrawal removes current member, keeps history with `leftAt`; non-member withdrawal is 404.|`tests/seedbed/live_api_test.go`|PT-SEM-03|
|T060|module-4-research|`TestProjectUpdatesKeepAuthorAndDateHistory`|Project update and progress persist history with author and date atomically; invalid progress leaves no history.|`tests/research/repository_test.go`|PT-SEM-05, PT-SEM-07, TDB-T07|
|T061|module-4-research|`TestConvertProposalKeepsHistoryAndLinksProject`|Conversion creates Radicado project, links proposal.projectId, keeps proposal history and attachments.|`tests/research/repository_test.go`|PT-SEM-06, TDB-T07|
|T062|module-4-research|`TestProposalWithoutSeedbedIsStored`|Proposal without seedbed or optional lists is stored.|`tests/research/repository_test.go`|PT-SEM-06|
|T063|module-4-research|`TestActivityWithoutOptionalListsIsStored`|Activity without responsibles or evidence is stored with empty lists.|`tests/research/repository_test.go`|PT-SEM-04|

For a new test type, add a table with the same columns. Keep domain as a column
in every table so tests remain filterable by functionality.
