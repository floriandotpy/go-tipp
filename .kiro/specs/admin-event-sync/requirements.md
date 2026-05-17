# Requirements Document

## Introduction

This feature replaces the existing per-phase manual import workflow with a single "Daten synchronisieren" button per event on the admin page. Clicking the button fetches all match data from the event's external API in one call, auto-creates/updates phases based on the group metadata in the response, detects duplicate matches, and presents a combined preview before persisting. The manual "Phase hinzufügen" flow and per-phase import flow are removed. The phase edit UI gains a dropdown for correcting the auto-inferred phase type.

## Glossary

- **SyncSystem**: The server-side component responsible for fetching, parsing, and persisting event data during synchronization
- **AdminUI**: The server-rendered HTML admin interface used by administrators
- **EventAPI**: The external REST API identified by `Event.ApiBaseURL` that returns match data
- **EventPhase**: A tournament phase (group stage or knockout round) stored in the `event_phases` table
- **PhaseType**: One of `phase_group` or `phase_ko`, indicating whether a phase is a group stage or knockout round
- **ApiMatch**: The Go struct representing a single match as returned by the EventAPI
- **GroupObject**: The JSON object within each ApiMatch containing `groupName`, `groupOrderID`, and `groupID`
- **DuplicateMatch**: A match that already exists in the database, identified by matching day, teamA, and teamB via `GetByMetadata`
- **SyncPreview**: The intermediate page showing phases and matches to be created/updated before confirmation

## Requirements

### Requirement 1: Event Sync Button

**User Story:** As an admin, I want a "Daten synchronisieren" button per event on the admin page, so that I can fetch all match data for an event in one action.

#### Acceptance Criteria

1. THE AdminUI SHALL display a "Daten synchronisieren" button for each event in the admin event list.
2. WHEN the admin clicks the "Daten synchronisieren" button, THE SyncSystem SHALL fetch all match data from the EventAPI using the event's `ApiBaseURL` in a single HTTP GET request.
3. IF the EventAPI returns a non-200 status code or a network error occurs, THEN THE SyncSystem SHALL display an error message on the admin page describing the failure.

### Requirement 2: Group Object Parsing

**User Story:** As an admin, I want the system to parse group metadata from the API response, so that phases can be created automatically.

#### Acceptance Criteria

1. THE ApiMatch struct SHALL include a `Group` field that maps to the JSON `group` object containing `groupName`, `groupOrderID`, and `groupID`.
2. WHEN the SyncSystem receives the API response, THE SyncSystem SHALL extract the `groupName`, `groupOrderID`, and `groupID` from each match's GroupObject.
3. THE SyncSystem SHALL identify distinct groups by their `groupOrderID` value.

### Requirement 3: Auto-Upsert Phases

**User Story:** As an admin, I want phases to be automatically created or updated from the API group data, so that I do not have to configure them manually.

#### Acceptance Criteria

1. WHEN the admin confirms the sync preview, THE SyncSystem SHALL create a new EventPhase for each distinct group where no EventPhase with the same `number` exists for the event.
2. WHEN the admin confirms the sync preview and an EventPhase with the same `number` already exists for the event, THE SyncSystem SHALL update that EventPhase's title, phase type, start, and end fields.
3. THE SyncSystem SHALL set the EventPhase `number` field to the group's `groupOrderID`.
4. THE SyncSystem SHALL set the EventPhase `title` field to the group's `groupName`.
5. THE SyncSystem SHALL set the EventPhase `start` field to the earliest `matchDateTime` among all matches in that group.
6. THE SyncSystem SHALL set the EventPhase `end` field to the latest `matchDateTime` among all matches in that group.

### Requirement 4: Phase Type Inference

**User Story:** As an admin, I want the system to automatically determine whether a phase is a group stage or knockout round, so that match scoring works correctly.

#### Acceptance Criteria

1. WHEN the group's `groupName` contains any of the substrings "Finale", "Viertelfinale", "Halbfinale", or "Achtelfinale", THE SyncSystem SHALL set the PhaseType to `phase_ko`.
2. WHEN the group's `groupName` does not contain any of the substrings "Finale", "Viertelfinale", "Halbfinale", or "Achtelfinale", THE SyncSystem SHALL set the PhaseType to `phase_group`.

### Requirement 5: Auto-Upsert Matches

**User Story:** As an admin, I want matches to be automatically inserted while avoiding duplicates, so that re-syncing does not create duplicate entries.

#### Acceptance Criteria

1. WHEN the admin confirms the sync preview, THE SyncSystem SHALL insert a new match for each non-duplicate ApiMatch in the response.
2. THE SyncSystem SHALL determine a match is a DuplicateMatch by calling `GetByMetadata` with the match's date (day portion of `matchDateTime`), `teamA` name, and `teamB` name.
3. WHEN a match is identified as a DuplicateMatch, THE SyncSystem SHALL skip insertion of that match.
4. THE SyncSystem SHALL set the inserted match's `event_phase` to the `groupOrderID` of the match's GroupObject.
5. THE SyncSystem SHALL set the inserted match's `match_type` to the inferred PhaseType of the match's group.

### Requirement 6: Sync Preview Page

**User Story:** As an admin, I want to review all phases and matches before they are persisted, so that I can verify the data is correct.

#### Acceptance Criteria

1. WHEN the SyncSystem successfully fetches and parses the API response, THE AdminUI SHALL display a SyncPreview page showing all phases to be created or updated.
2. THE SyncPreview page SHALL display each phase with its number, title, inferred PhaseType, start time, and end time.
3. THE SyncPreview page SHALL display all matches grouped by phase, showing date, time, teamA, and teamB for each match.
4. THE SyncPreview page SHALL visually indicate which matches are duplicates and will be skipped.
5. THE SyncPreview page SHALL provide a single "Bestätigen" button to confirm and persist all phases and matches.

### Requirement 7: Confirmation Summary

**User Story:** As an admin, I want to see a summary after sync completes, so that I know what was created.

#### Acceptance Criteria

1. WHEN the admin confirms the sync and persistence completes successfully, THE SyncSystem SHALL redirect to the admin page.
2. WHEN the admin confirms the sync and persistence completes successfully, THE SyncSystem SHALL display a flash message showing the count of phases created, phases updated, and matches inserted.

### Requirement 8: Remove Manual Phase Creation Flow

**User Story:** As an admin, I want the manual "Phase hinzufügen" flow removed, so that the admin interface is simplified and phases are only created via sync.

#### Acceptance Criteria

1. THE AdminUI SHALL NOT display the "Phase hinzufügen" button on the admin event list.
2. THE SyncSystem SHALL NOT register the `GET /admin/events/{eventID}/phases/new` route.
3. THE SyncSystem SHALL NOT register the `POST /admin/events/{eventID}/phases/new` route.

### Requirement 9: Remove Per-Phase Import Flow

**User Story:** As an admin, I want the per-phase import flow removed, so that all data import happens through the unified sync button.

#### Acceptance Criteria

1. THE AdminUI SHALL NOT display the "Spieldaten laden" link in the phase table.
2. THE SyncSystem SHALL NOT register the `GET /admin/phases/{phaseID}/import` route.
3. THE SyncSystem SHALL NOT register the `POST /admin/phases/{phaseID}/import` route.

### Requirement 10: Phase Type Dropdown in Edit UI

**User Story:** As an admin, I want a dropdown for phase_type in the phase edit form, so that I can correct the auto-inferred type if needed.

#### Acceptance Criteria

1. THE AdminUI SHALL display a dropdown select element for the `phase_type` field on the phase edit page.
2. THE dropdown SHALL offer exactly two options: `phase_group` and `phase_ko`.
3. THE dropdown SHALL pre-select the current PhaseType value of the phase being edited.
4. WHEN the admin submits the phase edit form with a changed PhaseType, THE SyncSystem SHALL persist the updated PhaseType value.

### Requirement 11: Backward Compatibility of ApiPath Field

**User Story:** As a developer, I want the `EventPhase.ApiPath` field retained in the schema, so that existing events using per-phase import continue to function.

#### Acceptance Criteria

1. THE EventPhase struct SHALL retain the `ApiPath` field.
2. WHEN creating new phases via sync, THE SyncSystem SHALL set the `ApiPath` field to an empty string.
