# Requirements Document

## Introduction

The go-tipp application is currently hardcoded for a single tournament (Euro 2024). This feature introduces multi-tournament support, allowing the system to manage multiple events (e.g. FIFA World Cup 2026, Euro 2028) with their respective phases, matches, and leaderboards. An "active event" concept controls what users see by default, while historical event data remains accessible.

## Glossary

- **Event**: A tournament or competition (e.g. "Euro 2024", "FIFA World Cup 2026") stored in the `events` database table.
- **Event_Phase**: A stage within an Event (e.g. "Gruppenphase 1", "Achtelfinale") stored in the `event_phases` database table, replacing the current hardcoded phase definitions.
- **Active_Event**: The single Event currently marked as active, which determines the default view for all users.
- **Match**: A scheduled game between two teams, belonging to exactly one Event via a foreign key.
- **Tipp**: A user's prediction for a specific Match outcome.
- **Leaderboard**: A ranked list of users by points, scoped to a specific Event.
- **CLI_Tool**: The command-line application (`cmd/cli/main.go`) that fetches live results from the openligadb.de API.
- **Admin**: A user with elevated privileges who can manage events and trigger system operations.
- **Phase_Type**: A classification of an Event_Phase as either "group" or "knockout", used for scoring point calculations.

## Requirements

### Requirement 1: Events Table

**User Story:** As an admin, I want to store tournament metadata in a database table, so that the system can support multiple tournaments over time.

#### Acceptance Criteria

1. THE Database SHALL contain an `events` table with columns: `id` (integer, auto-increment primary key), `name` (varchar(255), not null), `slug` (varchar(255), unique, not null), `api_base_url` (varchar(255), not null), `is_active` (boolean, not null, default false), `created` (datetime, not null, default current timestamp).
2. WHEN a new Event is inserted with a `slug` value that already exists in the `events` table, THE Database SHALL reject the insert and return a uniqueness constraint violation error.
3. WHEN the events-table migration is applied, THE Database SHALL insert exactly one Event row with `name` = "Euro 2024", `slug` = "euro-2024", `api_base_url` = "https://api.openligadb.de/getmatchdata/em/2024", and `is_active` = true.

### Requirement 2: Event Phases Table

**User Story:** As an admin, I want event phases stored in the database rather than hardcoded in Go source, so that new tournaments can define their own phase structure without code changes.

#### Acceptance Criteria

1. THE Database SHALL contain an `event_phases` table with columns: `id` (auto-increment primary key), `event_id` (foreign key to `events.id`, not null), `number` (int, not null, minimum value 1), `title` (varchar(255), not null), `api_path` (varchar(512), not null), `phase_type` (varchar(50), not null), `start` (datetime, not null), `end` (datetime, not null).
2. WHEN an Event is deleted, THE Database SHALL cascade-delete all associated Event_Phase rows.
3. THE Database SHALL enforce a unique constraint on the combination of `event_id` and `number`.
4. THE `phase_type` column SHALL contain one of the values "phase_group" or "phase_ko".
5. THE Database SHALL contain Event_Phase rows matching the current seven hardcoded phases for Euro 2024 after migration, with `number` values 1 through 7, titles matching the existing `Phases` variable in `events.go`, and `phase_type` set to "phase_group" for phases 1–3 and "phase_ko" for phases 4–7.
6. IF the `start` value of an Event_Phase row is equal to or later than its `end` value, THEN THE Database SHALL reject the insert or update.
7. WHEN an Event_Phase row is deleted or its `number` is updated, THE Database SHALL maintain referential integrity with any `matches.event_phase` column referencing that phase.

### Requirement 3: Match-Event Association

**User Story:** As a developer, I want each match linked to an event, so that matches can be filtered and displayed per tournament.

#### Acceptance Criteria

1. THE Database SHALL add an `event_id` column (integer, foreign key to `events.id`, not null) to the `matches` table with a foreign key constraint that prevents deletion of an Event while associated matches exist.
2. WHEN the migration runs, THE Database SHALL set `event_id` on all existing match rows to the Euro 2024 event ID inserted by the events table migration.
3. THE Match model struct SHALL include an `EventID` field of type `int`, and all insert and query operations (including `Get`, `All`, `AllByDaterange`, and `GetByMetadata`) SHALL read and write the `event_id` column.
4. IF a match is inserted with an `event_id` that does not reference an existing row in the `events` table, THEN THE Database SHALL reject the insert with a foreign key constraint error.
5. WHEN a match query filters by `event_id`, THE Match model SHALL return only matches whose `event_id` matches the provided value, and SHALL return an empty result set if no matches exist for that event.

### Requirement 4: Active Event Selection

**User Story:** As an admin, I want to mark exactly one event as active, so that users see the current tournament by default.

#### Acceptance Criteria

1. WHEN an admin sets an Event as active, THE System SHALL atomically set `is_active = true` on that Event and `is_active = false` on all other Events within a single transaction.
2. THE System SHALL ensure exactly one Event is active at any time; there is no operation that allows deactivating an Event without simultaneously activating another.
3. IF no Event is marked as active during application startup, THEN THE System SHALL log an error message indicating no active event was found and terminate with a non-zero exit code.
4. WHEN a user requests the default tournament view without specifying an event, THE System SHALL display the Event where `is_active = true`.
5. WHEN an admin successfully sets an Event as active, THE System SHALL confirm the change by displaying a success notification to the admin.

### Requirement 5: Event-Scoped Match Queries

**User Story:** As a user, I want to see only matches from the active tournament, so that past tournament data does not clutter my view.

#### Acceptance Criteria

1. WHEN a user views the matches page, THE Web_Application SHALL display only matches belonging to the Active_Event.
2. WHEN a user views match details, THE Web_Application SHALL verify the match belongs to the Active_Event before rendering.
3. IF a user requests match details for a match that does not belong to the Active_Event, THEN THE Web_Application SHALL respond with a 404 Not Found page.
4. THE MatchModel `All` and `AllByDaterange` methods SHALL accept an `event_id` parameter and return only matches whose `event_id` matches the provided parameter.
5. THE Web_Application SHALL determine the Active_Event as the event record marked active in the database, and if no event is marked active, SHALL fall back to the most recently created event.

### Requirement 6: Event-Scoped Leaderboards

**User Story:** As a user, I want leaderboards to reflect only the current tournament, so that scores from past events do not mix with the active competition.

#### Acceptance Criteria

1. WHEN a user views the leaderboard, THE Web_Application SHALL compute each user's total points by summing only the `points` column from tipps whose associated match belongs to the Active_Event, excluding tipps linked to matches from any other event.
2. WHEN a user views the scoreboard chart, THE Web_Application SHALL include only matches that belong to the Active_Event when building the cumulative points-per-match dataset.
3. THE `GroupLeaderboard` and `GlobalLeaderboard` queries SHALL join tipps to matches and filter by the Active_Event identifier so that only tipps for matches within that event contribute to the ranking.
4. IF no matches in the Active_Event have finished results, THEN THE Web_Application SHALL display the leaderboard with zero points for every user rather than falling back to data from a previous event.
5. WHEN the Active_Event changes, THE Web_Application SHALL reset all displayed leaderboard rankings to reflect only the new Active_Event's match data from that point forward.

### Requirement 7: Event-Scoped Tipp Submission

**User Story:** As a user, I want to submit tipps only for matches in the active tournament, so that I cannot accidentally interact with archived events.

#### Acceptance Criteria

1. WHEN a user submits a tipp, THE Web_Application SHALL verify the target match belongs to the Active_Event before accepting the submission, in addition to existing match-level validations (match not started, both teams known).
2. IF a tipp submission targets a match not in the Active_Event, THEN THE Web_Application SHALL reject that tipp with an error message indicating the match does not belong to the active tournament.
3. WHEN a bulk tipp submission contains matches from both the Active_Event and a non-active event, THE Web_Application SHALL skip tipps for non-active event matches and continue processing the remaining valid tipps without aborting the entire request.

### Requirement 8: Event-Scoped Points Calculation

**User Story:** As a developer, I want the points update logic to use phase types from the database, so that scoring works correctly for any tournament structure.

#### Acceptance Criteria

1. WHEN the `UpdatePoints` method runs, THE TippModel SHALL join `matches` to `event_phases` on `matches.event_id = event_phases.event_id` AND `matches.event_phase = event_phases.number` to determine the `phase_type` for each match, instead of using hardcoded phase number ranges.
2. WHEN the `InferEventPhaseType` function is called with a Match, THE System SHALL query the `event_phases` table using the match's `event_id` and `event_phase` number to return the corresponding `phase_type` value ("phase_group" or "phase_ko").
3. IF a match has no corresponding row in the `event_phases` table, THEN THE System SHALL treat that match as scoring zero points and skip its point calculation.
4. THE points calculation SQL SHALL apply group-phase scoring rules when `phase_type` is "phase_group" and knockout-phase scoring rules when `phase_type` is "phase_ko", using the point values defined in the `PhasePointsMap` configuration.

### Requirement 9: CLI Tool Database-Driven Phases

**User Story:** As a developer, I want the CLI tool to read event phases from the database, so that it fetches results for the correct API endpoints without code changes per tournament.

#### Acceptance Criteria

1. WHEN the CLI_Tool starts, THE CLI_Tool SHALL query the Active_Event and its associated Event_Phases from the database ordered by Event_Phase `number` ascending.
2. WHEN constructing API URLs, THE CLI_Tool SHALL concatenate the Event `api_base_url` with the Event_Phase `api_path` using simple string concatenation (the `api_base_url` ends without a trailing slash and `api_path` begins with a leading slash, e.g. `https://api.openligadb.de` + `/getmatchdata/em/2024/1`).
3. WHEN the CLI_Tool determines the current phase, THE CLI_Tool SHALL select the Event_Phase whose `start` timestamp is less than or equal to the current time and whose `end` timestamp is greater than or equal to the current time.
4. IF no Active_Event exists in the database, THEN THE CLI_Tool SHALL exit with a non-zero exit code and print an error message indicating that no active event was found.
5. IF the Active_Event has no Event_Phases in the database, THEN THE CLI_Tool SHALL exit with a non-zero exit code and print an error message indicating that no phases are configured for the active event.
6. IF the current time does not fall within any Event_Phase `start` and `end` range, THEN THE CLI_Tool SHALL exit with a non-zero exit code and print a message indicating that no phase is currently active.

### Requirement 10: Historical Event Access

**User Story:** As a user, I want to optionally view data from past tournaments, so that I can revisit previous competition results.

#### Acceptance Criteria

1. WHERE an event selector is provided, THE Web_Application SHALL allow users to switch the displayed event via a URL query parameter (e.g. `?event=euro-2024`), where the event identifier is a lowercase alphanumeric slug of at most 64 characters.
2. IF no event query parameter is provided, THEN THE Web_Application SHALL display the currently active event by default.
3. WHEN a user selects a non-active event, THE Web_Application SHALL display that event's matches, tipps, and leaderboard with tipp input fields hidden or disabled so that no editing controls are available.
4. WHILE viewing a non-active event, THE Web_Application SHALL reject any tipp submission requests and not modify stored data.
5. IF a user provides an event query parameter that does not match any known event slug, THEN THE Web_Application SHALL respond with a 404 Not Found page.

### Requirement 11: Data Migration Integrity

**User Story:** As a developer, I want the migration to preserve all existing Euro 2024 data, so that no historical information is lost.

#### Acceptance Criteria

1. WHEN the migration runs, THE Database SHALL create the `events` table and insert a row for Euro 2024 with `name = 'Euro 2024'`, `slug = 'euro-2024'`, `api_base_url = 'https://api.openligadb.de/getmatchdata/em/2024'`, and `is_active = true`.
2. WHEN the migration runs, THE Database SHALL create the `event_phases` table and insert seven rows corresponding to the current hardcoded phase definitions (Gruppenphase 1, Gruppenphase 2, Gruppenphase 3, Achtelfinale, Viertelfinale, Halbfinale, Finale) with their respective start dates, end dates, and API paths.
3. WHEN the migration runs, THE Database SHALL add the `event_id` column to `matches` as a foreign key referencing `events(id)` and populate it with the Euro 2024 event ID for all existing rows.
4. IF the migration is rolled back, THEN THE Database SHALL drop the `event_phases` table, drop the `events` table, remove the `event_id` column from `matches`, and leave all pre-existing match rows unchanged.
5. IF the migration fails partway through, THEN THE Database SHALL not apply any partial changes, leaving the schema in its pre-migration state.

### Requirement 12: Admin Event Management

**User Story:** As an admin, I want to create new events and their phases through the admin interface, so that I can set up future tournaments.

#### Acceptance Criteria

1. WHEN an admin submits the event creation form with a valid name (1–100 characters), a unique slug (1–100 characters, lowercase alphanumeric and hyphens only), and a valid API base URL (1–255 characters), THE Web_Application SHALL insert a new Event row with those values and set `is_active` to false.
2. IF an admin submits the event creation form with a missing or invalid name, slug, or API base URL, THEN THE Web_Application SHALL reject the submission and display a validation error message indicating which fields are invalid.
3. IF an admin submits the event creation form with a slug that already exists, THEN THE Web_Application SHALL reject the submission and display a validation error message indicating the slug is already taken.
4. WHEN an admin adds phases to an event with a valid phase number (integer ≥ 1), title (1–100 characters), API path (1–255 characters), phase type (one of: "phase_group" or "phase_ko"), and start and end timestamps where start is before end, THE Web_Application SHALL insert Event_Phase rows associated with that event.
5. IF an admin adds a phase with missing or invalid fields, or with a start timestamp that is not before the end timestamp, THEN THE Web_Application SHALL reject the submission and display a validation error message indicating which fields are invalid.
6. WHEN an admin activates a different event, THE Web_Application SHALL set `is_active` to true on the selected event and set `is_active` to false on the previously active event, ensuring exactly one event is active at any time.
