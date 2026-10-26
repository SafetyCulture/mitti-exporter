# Data Feed Coverage Audit — developer.mitti.com vs Mitti Exporter

**Date:** 2026-10-13
**Worktree:** `docs-audit/api-coverage` (based on `origin/main` @ `ca560d5`)
**Docs snapshot:** developer.mitti.com, OpenAPI `v1.348.74` (pages updated 2026-08-11)
**Sources:** [Data Feeds reference](https://developer.mitti.com/reference/data-feeds), [llms.txt index](https://developer.mitti.com/llms.txt), exporter code at `cli/pkg/internal/feed/` + `cli/pkg/api/exporter_api.go`

## Executive summary

*The findings below are the original audit, written before the feeds were implemented. All 17 gaps are now closed; see the status note under this summary and the matrix in section 3.*

The customer complaints were confirmed. The exporter was **17 documented feed endpoints behind** across **7 feature families**, and the family with active complaints — **Scheduling** — is the most serious: the exporter still exports schedules from the **legacy** `thepubservice` feeds, while the docs now label those "(legacy)" and direct orgs on the new Schedules experience to a **new Scheduling feed service** (`/scheduling/v1/feed/*`) that the exporter does not call at all.

---

> **Status (2026-10-14):** all 17 missing feeds are now implemented in the CLI (`cli/pkg/internal/feed/feed_row.go` plus one file per family) and added to the desktop app's table pickers. Paths marked "inferred" in the original audit were verified against the OpenAPI definitions and corrected below. All new feeds are full-refresh (no incremental sync yet).

## 1. Scheduling — the active complaint (HIGH)

### What the exporter does today

All three schedule tables come from the legacy endpoints (see `feed_schedule.go`, `feed_schedule_assignee.go`, `feed_schedule_occurrence.go`):

| Exporter table | Endpoint used | Docs status |
|---|---|---|
| `schedules` | `/feed/schedules` | **(legacy)** `ThePubService_FeedSchedules` |
| `schedule_assignees` | `/feed/schedule_assignees` | **(legacy)** `ThePubService_FeedScheduleAssignees` |
| `schedule_occurrences` | `/feed/schedule_occurrences` | **(legacy)** `ThePubService_FeedScheduleOccurrences` |

The docs' legacy pages now carry an explicit callout:

> 📘 For organizations on the new Schedules experience, use the [Data feed for Schedules](https://developer.mitti.com/reference/schedulingfeedservice_feedschedules) for new builds.

### What the exporter is missing

**New Scheduling feed service (V2) — zero coverage:**

| Endpoint | Operation ID | Notes |
|---|---|---|
| `/scheduling/v1/feed/schedules` | `SchedulingFeedService_FeedSchedules` | limit max **1000** (legacy: 500), `next_page_token` pagination, `template` filter |
| `/scheduling/v1/feed/schedule_assignees` | `SchedulingFeedService_FeedScheduleAssignees` | new `completion_rule`; richer assignee `type` incl. `location`, `location_intersection` |
| `/scheduling/v1/feed/schedule_occurrences` | `SchedulingFeedService_FeedScheduleOccurrences` | `start_date`/`end_date` apply to **`due_time`**; richer statuses |

**Schema/data loss vs the legacy rows the exporter writes (`Schedule` struct):**

- `title` (legacy has `description` only), `created_at`, `created_by` (vs `creator_user_id`)
- `completion_rule` (`any`/`all` enum) vs legacy boolean `all_must_complete`
- `timezone_type` (`CUSTOM_TIMEZONE`/`SITE_TIMEZONE`) — new
- `is_late_submit_allowed` vs `can_late_submit` (naming change)
- `site_ids` **(array, multi-site schedules)** vs legacy single `site_id`
- `asset_ids` **(array, multi-asset schedules)** vs legacy single `asset_id`
- `grace_period_duration`, `site_based_assignment_enabled` — new
- Ending rules: `ending_rule_type` (`never`/`end_date`/`end_after_repeats`), `ending_rule_end_date`, `ending_rule_end_after_repeats` — new (legacy only has `from_date`/`to_date`)
- `assignees`, `intersections` (site+group intersection-based assignment) — new
- `schedule_type` (`USERS`/`SITES`/`ASSETS`) — new
- Occurrences: `WONT_DO` status + `wont_do_note`, `assignee_from`, `completion_rule`, `audit_id` linkage documented explicitly; occurrence `occurrence_status` enum is richer (`TODO`, `IN_PROGRESS`, `COMPLETED`, `LATE`, `OVERDUE`, `MISSED`, `WONT_DO`)

### Impact

For orgs migrated to the new Schedules experience, the legacy feeds return incomplete/stale schedule data (or nothing), so `schedules`, `schedule_assignees`, and `schedule_occurrences` exports are unreliable — exactly matching "we are missing data feeds for Scheduling" complaints.

### Implementation notes (if we add the new feeds)

- Pagination: new feeds return `metadata.next_page` (full path) **plus** `next_page_token`. `DrainFeed` already follows `metadata.next_page`, so it should mostly work — **but** `s12.common.v1.FeedMetadataSimple` has **no `remaining_records`** field. The exporter's progress/status reporting (`status.UpdateStatus(f.Name(), resp.Metadata.RemainingRecords, ...)`) and `HasRemainingInformation()` assume it exists, so new-feed implementations must handle its absence (return `false` from `HasRemainingInformation()`).
- Query params: `show_active`/`show_finished`/`show_paused` are **booleans** on the new service (legacy: string enums); the exporter currently passes neither.
- The new `schedules` response also has `total_records` at the top level.
- ID formats differ (`scheduleitem_{id}` etc.) — consider separate table names (e.g. `new_schedules`) rather than overwriting legacy tables, since orgs not yet migrated still need legacy feeds.

---

## 2. Other documented feed families with ZERO exporter coverage (before this change)

Verified absent — no references anywhere in the exporter codebase (grep for `investigation|contractor|credential|maintenance_plan|maintenance_program|action_field|issue_relation` matches nothing but a comment):

| # | Family | Endpoints (documented) | Docs links |
|---|---|---|---|
| 1 | **Action fields** | `/feed/action_fields` (`FeedService_FeedActionFields`) — plus a second, similarly-named page `feedservice_feedactionfields2` | [action fields](https://developer.mitti.com/reference/feedservice_feedactionfields) |
| 2 | **Issue relations** | `/feed/issue_relations` (`ThePubService_FeedIssueRelationships`) — links issues to inspections/assets/actions (`ISSUE-INSPECTION`, `ISSUE-ACTION`) | [issue relations](https://developer.mitti.com/reference/thepubservice_feedissuerelationships) |
| 3 | **Investigations** | `/incidents/v1/feed/investigations` + investigation fields, investigation detail fields, investigation relationships (4 endpoints) | [investigations](https://developer.mitti.com/reference/feedservice_feedinvestigations) |
| 4 | **Contractor companies** | `/companies/v1/feed/companies` + company documents + company members (3 endpoints); supports `modified_after` incremental sync | [contractor companies](https://developer.mitti.com/reference/feedservice_feedcontractorcompanies) |
| 5 | **Credentials** | credential types + credentials — docs list **two pairs** (`feedservice_credentialtypes`/`feedservice_credentials` and `feedservice_feeddocumenttypes`/`feedservice_feeduserdocuments`) | [credentials](https://developer.mitti.com/reference/feedservice_feeduserdocuments) |
| 6 | **Maintenance** | `/assets/v1/feed/maintenance_plans` + maintenance programs + assets maintenance details (3 endpoints) | [maintenance plans](https://developer.mitti.com/reference/feedservice_feedmaintenanceplans) |
| 7 | **New Scheduling** | (covered in section 1) | — |

### Per-family impact

- **Action fields** — customers using custom fields on actions get no field values in exports. The `actions`/`action_assignees`/`action_timeline_items` tables don't carry them; there is a dedicated feed for exactly this.
- **Issue relations** — no way to join `issues` to the inspections/actions/assets they came from in exported data.
- **Investigations** — the whole Incidents/Investigations module is invisible to the exporter.
- **Companies/contractors** — contractor management data (incl. compliance stats inputs) is invisible.
- **Credentials** — credential/licence tracking data is invisible.
- **Maintenance** — exporter has an `assets` table but none of the maintenance programs/plans/asset maintenance detail feeds.

### Docs-side duplication (resolved)

The reference lists what look like two generations of the same feed under different operation IDs:
- `FeedService_FeedActionFields` **and** `FeedService_FeedActionFields2`
- `FeedService_CredentialTypes` **and** `FeedService_FeedDocumentTypes`
- `FeedService_Credentials` **and** `FeedService_FeedUserDocuments`

**Resolved:** `feeddocumenttypes`/`feeduserdocuments` are stub pages with no spec; `credentialtypes`/`credentials` carry the full OpenAPI definition (`/credentials/v1/feed/credential-types`, `/credentials/v1/feed/credentials`) and are what the exporter calls. The two action-field pages are the same schema at `/feed/action_fields` and `/tasks/v1/feed/action_fields`; the exporter uses `/tasks/v1/feed/action_fields` because `/feed/action_fields` timed out (504) against a real org.

---

## 3. Full coverage matrix (documented data feeds vs exporter)

| Documented feed | Endpoint | Exporter table | Status |
|---|---|---|---|
| Inspections | `/feed/inspections` | `inspections` | ✅ |
| Inspection items | `/feed/inspection_items` | `inspection_items` | ✅ |
| Templates | `/feed/templates` | `templates` | ✅ |
| Template permissions | `/feed/template_permissions` | `template_permissions` | ✅ |
| Sites | `/feed/sites` | `sites` | ✅ |
| Site members | `/feed/site_members` | `site_members` | ✅ |
| Groups | `/feed/groups` | `groups` | ✅ |
| Group users | `/feed/group_users` | `group_users` | ✅ |
| Users | `/feed/users` | `users` | ✅ |
| Actions | `/feed/actions` | `actions` | ✅ |
| Action assignees | `/feed/action_assignees` | `action_assignees` | ✅ |
| Action timeline items | `/feed/action_timeline_items` | `action_timeline_items` | ✅ |
| Issues | `/feed/issues` | `issues` | ✅ |
| Issue assignees | `/feed/issue_assignees` | `issue_assignees` | ✅ |
| Issue timeline items | `/feed/issue_timeline_items` | `issue_timeline_items` | ✅ |
| Assets | `/feed/assets` | `assets` | ✅ |
| Course progress | `/training/v1/feed/training-course-progress` | `training_course_progress` | ✅ |
| Activity log events | `/accounts/history/v2/feed/activity_log_events` | `account_histories` | ✅ |
| Schedules (legacy) | `/feed/schedules` | `schedules` | ⚠️ legacy-only |
| Schedule assignees (legacy) | `/feed/schedule_assignees` | `schedule_assignees` | ⚠️ legacy-only |
| Schedule occurrences (legacy) | `/feed/schedule_occurrences` | `schedule_occurrences` | ⚠️ legacy-only |
| **New Schedules** | `/scheduling/v1/feed/schedules` | `new_schedules` | ✅ added |
| **New Schedule Assignees** | `/scheduling/v1/feed/schedule_assignees` | `new_schedule_assignees` | ✅ added |
| **New Schedule Occurrences** | `/scheduling/v1/feed/schedule_occurrences` | `new_schedule_occurrences` | ✅ added |
| **Action fields** | `/tasks/v1/feed/action_fields` (also documented at `/feed/action_fields`, identical schema, but that path returned 504 Gateway Timeout on every attempt against a real org on 2026-10-14) | `action_fields` | ✅ added |
| **Issue relations** | `/feed/issue_relations` | `issue_relations` | ✅ added |
| **Investigations** | `/incidents/v1/feed/investigations` | `investigations` | ✅ added |
| **Investigation fields** | `/incidents/v1/feed/investigation_fields` | `investigation_fields` | ✅ added |
| **Investigation detail fields** | `/incidents/v1/feed/investigation_detail_fields` | `investigation_detail_fields` | ✅ added |
| **Investigation relationships** | `/incidents/v1/feed/investigation_relationships` | `investigation_relationships` | ✅ added |
| **Contractor companies** | `/companies/v1/feed/companies` | `contractor_companies` | ✅ added |
| **Contractor company documents** | `/companies/v1/feed/companies/documents` | `contractor_company_documents` | ✅ added |
| **Contractor company members** | `/companies/v1/feed/companies/members` | `contractor_company_members` | ✅ added |
| **Credential types** | `/credentials/v1/feed/credential-types` | `credential_types` | ✅ added |
| **Credentials** | `/credentials/v1/feed/credentials` | `credentials` | ✅ added |
| **Assets maintenance details** | `/assets/v1/feed/assets_maintenance` | `assets_maintenance` | ✅ added |
| **Maintenance plans** | `/assets/v1/feed/maintenance_plans` | `maintenance_plans` | ✅ added |
| **Maintenance programs** | `/assets/v1/feed/maintenance_programs` | `maintenance_programs` | ✅ added |
| **Lone Worker jobs** | `/loneworker/v1/feed/jobs` | `lone_worker_jobs` | ✅ added (incremental via `modified_after`) |

*(SHEQSY feeds — employees, departments, department employees, activities, shifts — are separate and not documented under Data Feeds; out of scope for this audit.)*

## 4. Minor observations

- Git history: schedule feed files were last touched only by the rebrand (FG-7681) and monorepo consolidation (FG-5091) — no functional work on scheduling export in a long time.
- The docs "understanding-the-data" page lists fewer exporter tables than the CLI actually has (it omits `site_members`, `issue_assignees`, `training_course_progress`, `account_histories`) — a smaller docs-side gap worth fixing while we're in there.
- The legacy `/feed/schedules` doc schema doesn't even include `asset_id`, though the exporter's `Schedule` struct has an `AssetID` column — the legacy feed is already drifting from its own docs, which reinforces moving to the V2 service.
- `ScheduleOccurrenceFeed.ResumeDownload` is marked `// EXPERIMENTAL: we don't have modified_at from the backend` — the new V2 occurrences feed's `start_date`/`end_date` filtering on `due_time` may allow a cleaner resume strategy.

## 5. Recommended priority order (as originally proposed; all items now done)

1. **New Scheduling feeds** (3 endpoints) — active customer complaints; docs already steer new builds here. Decide: additive new tables vs replacing legacy (recommend additive while migration is ongoing).
2. **Action fields** — completes the Actions story customers already assume exists.
3. **Issue relations** — single flat feed, enables joins customers ask for.
4. **Investigations family** (4 endpoints) — whole module missing.
5. **Contractor companies family** (3 endpoints).
6. **Maintenance family** (3 endpoints) — pairs naturally with the existing `assets` feed.
7. **Credentials** (confirm canonical pair first).

---

*Audit performed by comparing the documented Data Feeds reference (verified page-by-page for scheduling, action fields, issue relations, investigations, contractor companies, maintenance plans) against exporter feed registrations in `cli/pkg/internal/feed/feed_exporter.go` and every `InitialURL` in the feed package.*
