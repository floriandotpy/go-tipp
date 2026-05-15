# Requirements Document

## Introduction

Admin Match Import enables administrators to fetch match data from an external API for a specific event phase, preview the data in a table, and selectively import matches into the database. This replaces the need to manually insert match records and provides duplicate detection with visual feedback.

## Glossary

- **Admin_System**: The server-side web application handling admin routes, protected by `requireAdminAuthentication` middleware
- **Import_Handler**: The HTTP handler responsible for the GET (preview) and POST (confirm import) operations at `/admin/phases/{phaseID}/import`
- **API_Client**: The shared `internal/api` package that fetches and parses match data from the external football data API
- **Preview_Table**: The server-rendered HTML table displaying fetched match data for admin review before import
- **Match_Record**: A row in the `matches` database table representing a single match
- **Duplicate_Match**: A match from the API response that already exists in the database, identified by date, team A, and team B via `GetByMetadata`
- **EventPhase**: A configured phase belonging to an event, containing an `ApiPath` used to construct the full API URL
- **Event**: A tournament/competition entity containing an `ApiBaseURL`

## Requirements

### Requirement 1: Import Button on Admin Page

**User Story:** As an admin, I want a "Spieldaten laden" button for each phase on the admin page, so that I can initiate the match import process for a specific phase.

#### Acceptance Criteria

1. WHEN the admin page is rendered, THE Admin_System SHALL display a "Spieldaten laden" button for each EventPhase in the phases table.
2. WHEN the admin clicks the "Spieldaten laden" button for a phase, THE Admin_System SHALL navigate to `GET /admin/phases/{phaseID}/import`.

### Requirement 2: Fetch and Preview Match Data

**User Story:** As an admin, I want to see a preview of match data fetched from the external API, so that I can review the data before importing it.

#### Acceptance Criteria

1. WHEN a GET request is made to `/admin/phases/{phaseID}/import`, THE Import_Handler SHALL retrieve the EventPhase and its parent Event from the database.
2. IF the phaseID does not correspond to a valid EventPhase, THEN THE Import_Handler SHALL return an HTTP 404 response.
3. WHEN the EventPhase and Event are retrieved, THE API_Client SHALL fetch match data from the URL constructed as `Event.ApiBaseURL + EventPhase.ApiPath`.
4. IF the API request fails or returns a non-200 status code, THEN THE Import_Handler SHALL display an error message to the admin.
5. WHEN match data is successfully fetched, THE Import_Handler SHALL render a Preview_Table containing one row per match with columns: date, team A, team B, and event phase number.

### Requirement 3: Duplicate Detection and Display

**User Story:** As an admin, I want to see which matches already exist in the database, so that I can avoid unintentional re-imports.

#### Acceptance Criteria

1. WHEN the Preview_Table is rendered, THE Import_Handler SHALL check each fetched match against the database using `GetByMetadata` with the match date, team A, and team B.
2. WHEN a Duplicate_Match is detected, THE Preview_Table SHALL display an "already exists" warning badge on that row.
3. WHEN a Duplicate_Match is detected, THE Preview_Table SHALL pre-deselect the checkbox for that row.
4. THE Preview_Table SHALL pre-select the checkbox for each match that does not already exist in the database.

### Requirement 4: Selective Import via Checkboxes

**User Story:** As an admin, I want to select or deselect individual matches before importing, so that I have full control over which matches are inserted.

#### Acceptance Criteria

1. THE Preview_Table SHALL display a checkbox for each match row allowing the admin to include or exclude the match from import.
2. WHEN a Duplicate_Match row is displayed, THE Admin_System SHALL allow the admin to manually re-select the checkbox to force re-import.
3. WHEN a non-duplicate match row is displayed, THE Admin_System SHALL allow the admin to deselect the checkbox to skip import.

### Requirement 5: Confirm and Insert Matches

**User Story:** As an admin, I want to submit the selected matches for import, so that they are inserted into the database.

#### Acceptance Criteria

1. WHEN the admin submits the import form, THE Import_Handler SHALL receive a POST request to `/admin/phases/{phaseID}/import` containing the selected match indices.
2. WHEN the POST request is processed, THE Import_Handler SHALL insert each selected match into the `matches` table with the fields: start (datetime), team_a, team_b, match_type (EventPhase phase_type), event_phase (EventPhase number), and event_id (Event ID).
3. WHEN all selected matches are successfully inserted, THE Admin_System SHALL redirect the admin to the admin page with a success flash message indicating the number of imported matches.
4. IF an error occurs during insertion, THEN THE Admin_System SHALL display an error message to the admin.

### Requirement 6: Shared API Package

**User Story:** As a developer, I want the API types and fetch logic extracted into a shared package, so that both the CLI tool and the web handler can reuse the same code.

#### Acceptance Criteria

1. THE API_Client SHALL be located in the `internal/api` package.
2. THE API_Client SHALL export the types `ApiMatch`, `ApiTeam`, `ApiResult`, and `ApiGoal`.
3. THE API_Client SHALL export a `FetchMatchData(url string) ([]ApiMatch, error)` function.
4. WHEN the shared package is created, THE CLI tool (`cmd/cli/main.go`) SHALL import types and `FetchMatchData` from `internal/api` instead of defining them locally.

### Requirement 7: Route Registration and Authentication

**User Story:** As an admin, I want the import routes to be protected by admin authentication, so that only authorized users can import match data.

#### Acceptance Criteria

1. THE Admin_System SHALL register `GET /admin/phases/{phaseID}/import` using the admin middleware chain.
2. THE Admin_System SHALL register `POST /admin/phases/{phaseID}/import` using the admin middleware chain.
3. IF a non-admin user attempts to access the import routes, THEN THE Admin_System SHALL deny access according to the existing `requireAdminAuthentication` middleware behavior.
