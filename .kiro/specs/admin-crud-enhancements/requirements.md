# Requirements Document

## Introduction

This feature extends the admin interface of the go-tipp application with full CRUD (Create, Read, Update, Delete) capabilities for managing events and their phases. Currently the admin can create events and add phases, but cannot list phases per event, edit event or phase details, or delete events or phases. This spec covers the missing operations: listing phases inline on the admin page, editing events, editing phases, deleting events (with cascade), and deleting phases.

## Glossary

- **Admin_Interface**: The server-rendered HTML admin page and its associated routes, accessible only to authenticated admin users
- **Event**: A tournament entity stored in the `events` table, identified by ID, with name, slug, API base URL, and active status
- **Phase**: A time-bounded subdivision of an event stored in the `event_phases` table, identified by ID, with number, title, API path, phase type, start, and end timestamps
- **Cascade_Delete**: Database-level deletion behavior where removing a parent record automatically removes all dependent child records (phases, matches, tipps, goals)
- **Confirm_Dialog**: A browser-native JavaScript `confirm()` dialog presented before destructive actions

## Requirements

### Requirement 1: List Phases Inline on Admin Page

**User Story:** As an admin, I want to see all phases for each event listed inline on the admin index page, so that I can quickly review the phase configuration without navigating to a separate page.

#### Acceptance Criteria

1. WHEN the admin index page is loaded, THE Admin_Interface SHALL display all phases for each event nested directly beneath that event's row in the events table.
2. WHEN phases exist for an event, THE Admin_Interface SHALL display each phase's number, title, phase type, start date, and end date.
3. WHEN no phases exist for an event, THE Admin_Interface SHALL display a message indicating that no phases are configured for that event.
4. THE Admin_Interface SHALL order displayed phases by their number in ascending sequence.

### Requirement 2: Edit Event Details

**User Story:** As an admin, I want to edit an existing event's name, slug, and API base URL, so that I can correct mistakes or update configuration without recreating the event.

#### Acceptance Criteria

1. WHEN the admin navigates to the edit event page, THE Admin_Interface SHALL display a pre-filled form with the event's current name, slug, and API base URL values.
2. WHEN the admin submits a valid edit event form, THE Admin_Interface SHALL update the event record in the database and redirect to the admin index page with a success flash message.
3. WHEN the admin submits an edit event form with a blank name, THE Admin_Interface SHALL redisplay the form with a validation error on the name field.
4. WHEN the admin submits an edit event form with a slug that does not match the pattern of lowercase alphanumeric characters and hyphens, THE Admin_Interface SHALL redisplay the form with a validation error on the slug field.
5. WHEN the admin submits an edit event form with a slug that is already used by a different event, THE Admin_Interface SHALL redisplay the form with a duplicate slug validation error.
6. WHEN the admin submits an edit event form with a blank API base URL, THE Admin_Interface SHALL redisplay the form with a validation error on the API base URL field.
7. THE Admin_Interface SHALL provide an "Bearbeiten" link for each event on the admin index page that navigates to the edit event form.

### Requirement 3: Edit Phase Details

**User Story:** As an admin, I want to edit an existing phase's number, title, API path, phase type, start, and end timestamps, so that I can adjust phase configuration after initial creation.

#### Acceptance Criteria

1. WHEN the admin navigates to the edit phase page, THE Admin_Interface SHALL display a pre-filled form with the phase's current number, title, API path, phase type, start, and end values.
2. WHEN the admin submits a valid edit phase form, THE Admin_Interface SHALL update the phase record in the database and redirect to the admin index page with a success flash message.
3. WHEN the admin submits an edit phase form with a number less than 1, THE Admin_Interface SHALL redisplay the form with a validation error on the number field.
4. WHEN the admin submits an edit phase form with a blank title, THE Admin_Interface SHALL redisplay the form with a validation error on the title field.
5. WHEN the admin submits an edit phase form with a blank API path, THE Admin_Interface SHALL redisplay the form with a validation error on the API path field.
6. WHEN the admin submits an edit phase form with a phase type that is not "phase_group" or "phase_ko", THE Admin_Interface SHALL redisplay the form with a validation error on the phase type field.
7. WHEN the admin submits an edit phase form where the end timestamp is not after the start timestamp, THE Admin_Interface SHALL redisplay the form with a validation error on the end field.
8. THE Admin_Interface SHALL provide a "Bearbeiten" link for each phase on the admin index page that navigates to the edit phase form.

### Requirement 4: Delete Event

**User Story:** As an admin, I want to delete an event and have all associated data (phases, matches, tipps, goals) removed automatically, so that I can clean up test data or remove obsolete tournaments.

#### Acceptance Criteria

1. WHEN the admin clicks the "Löschen" button for an event, THE Admin_Interface SHALL display a JavaScript confirm() dialog asking the admin to confirm the deletion.
2. WHEN the admin confirms the deletion in the confirm dialog, THE Admin_Interface SHALL submit a POST request to delete the event.
3. WHEN the admin cancels the deletion in the confirm dialog, THE Admin_Interface SHALL remain on the admin index page without submitting any request.
4. WHEN a delete event request is processed, THE Admin_Interface SHALL remove the event record from the database, triggering cascade deletion of all associated phases, matches, tipps, and goals.
5. WHEN a delete event request completes successfully, THE Admin_Interface SHALL redirect to the admin index page with a success flash message.
6. IF the specified event does not exist, THEN THE Admin_Interface SHALL return a 404 Not Found response.
7. THE Admin_Interface SHALL display the "Löschen" button inline next to each event on the admin index page.

### Requirement 5: Delete Phase

**User Story:** As an admin, I want to delete a single phase from an event, so that I can remove incorrectly configured phases without deleting the entire event.

#### Acceptance Criteria

1. WHEN the admin clicks the "Löschen" button for a phase, THE Admin_Interface SHALL display a JavaScript confirm() dialog asking the admin to confirm the deletion.
2. WHEN the admin confirms the deletion in the confirm dialog, THE Admin_Interface SHALL submit a POST request to delete the phase.
3. WHEN the admin cancels the deletion in the confirm dialog, THE Admin_Interface SHALL remain on the admin index page without submitting any request.
4. WHEN a delete phase request is processed, THE Admin_Interface SHALL remove the phase record from the database.
5. WHEN a delete phase request completes successfully, THE Admin_Interface SHALL redirect to the admin index page with a success flash message.
6. IF the specified phase does not exist, THEN THE Admin_Interface SHALL return a 404 Not Found response.
7. THE Admin_Interface SHALL display the "Löschen" button inline next to each phase on the admin index page.

### Requirement 6: CSRF Protection and Admin Authorization

**User Story:** As a system operator, I want all new admin CRUD operations to be protected by CSRF tokens and admin authentication, so that the application remains secure.

#### Acceptance Criteria

1. THE Admin_Interface SHALL include a valid CSRF token in every form that performs a state-changing operation (edit, delete).
2. WHEN a POST request is received without a valid CSRF token, THE Admin_Interface SHALL reject the request.
3. THE Admin_Interface SHALL require admin authentication for all new event and phase CRUD routes.
4. WHEN an unauthenticated or non-admin user attempts to access a CRUD route, THE Admin_Interface SHALL deny access.
