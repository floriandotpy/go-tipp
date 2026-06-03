# Requirements Document

## Introduction

This feature adds a REST API to the go-tipp application, enabling programmatic access for reading matches and submitting tipps. The primary use case is allowing AI players to participate in tipping groups alongside human users. Users can enable API access in their settings and receive a personal Bearer token scoped exclusively to their own account.

## Glossary

- **API_Token**: A cryptographically random, per-user secret string used for Bearer token authentication against the REST API. Each token grants read and write access only to the owning user's data.
- **Token_Store**: The database table that persists API tokens, linking each token hash to a user ID along with metadata (creation timestamp, optional label).
- **API_Middleware**: The HTTP middleware that authenticates incoming REST API requests by extracting and validating the Bearer token from the Authorization header.
- **Tipp_API**: The set of REST API endpoints under the `/api/v1/` path prefix that allow reading matches and submitting tipps.
- **Match**: A scheduled football match within an event, identified by its ID, containing team names, start time, and optional result data.
- **Tipp**: A user's score prediction for a specific match, consisting of two integer values (tipp_a, tipp_b).
- **Active_Event**: The currently active event in the system, determining which matches are available for tipping.

## Requirements

### Requirement 1: API Token Generation

**User Story:** As a user, I want to generate an API token from my account settings, so that I can authenticate programmatic access to read matches and submit tipps on my behalf.

#### Acceptance Criteria

1. WHEN an authenticated user requests a new API token, THE Token_Store SHALL generate a cryptographically random token of at least 32 bytes, prefix it with a fixed identifier (e.g., "gtipp_"), and return the plaintext token to the user exactly once in the HTTP response.
2. WHEN a new API token is generated, THE Token_Store SHALL persist only a bcrypt hash (cost 12) of the token alongside the user ID and creation timestamp.
3. WHEN a user already has an existing API token and requests a new one, THE Token_Store SHALL revoke the previous token by deleting it before storing the new token hash.
4. THE Token_Store SHALL enforce a limit of one active API token per user.
5. WHEN an authenticated user requests to revoke their API token, THE Token_Store SHALL delete the token hash from the database before returning a success response to the user.
6. IF token generation or persistence fails, THEN THE Token_Store SHALL return an error response indicating the failure and SHALL NOT revoke any previously active token.
7. IF an authenticated user requests to revoke their API token but no active token exists, THEN THE Token_Store SHALL return a response indicating that no token is active.
8. IF an unauthenticated user attempts to generate or revoke an API token, THEN THE Token_Store SHALL reject the request with an authentication error.

### Requirement 2: API Token Management UI

**User Story:** As a user, I want to manage my API token through the web interface, so that I can enable and disable programmatic access without admin intervention.

#### Acceptance Criteria

1. WHEN an authenticated user navigates to their settings page, THE Tipp_API SHALL display the current API access status: "enabled" if a token exists for the user, or "disabled" if no token exists, along with a "Generate Token" action when no token exists and a "Revoke Token" action when a token exists.
2. WHEN a user clicks "Generate Token" and no token currently exists for that user, THE Tipp_API SHALL create a new token, display the plaintext token value in a read-only text field with a copy-to-clipboard button, and display a warning message indicating that the token will not be shown again after leaving the page.
3. WHEN a user clicks "Generate Token" and a token already exists for that user, THE Tipp_API SHALL replace the existing token by revoking it and generating a new one, displaying the new plaintext token value in the same manner as a first-time generation.
4. WHEN a user clicks "Revoke Token" and a token exists, THE Tipp_API SHALL delete the stored token and confirm successful revocation with a flash message, returning the settings page to the "disabled" state.
5. IF an unauthenticated user attempts to access the token management page or submit a generate/revoke action, THEN THE Tipp_API SHALL redirect the user to the login page without performing any token operation.
6. THE Tipp_API SHALL store at most one API token per user at any time.

### Requirement 3: Bearer Token Authentication

**User Story:** As an API consumer, I want to authenticate requests using a Bearer token, so that I can access the API without maintaining a browser session.

#### Acceptance Criteria

1. WHEN a request to a `/api/v1/` endpoint includes an Authorization header with the format `Bearer <token>`, and the token matches a stored bcrypt hash linked to an active user, THE API_Middleware SHALL authenticate the request and associate it with the token's owning user.
2. WHEN a request to a `/api/v1/` endpoint includes a Bearer token that does not match any stored bcrypt hash, THE API_Middleware SHALL respond with HTTP 401 Unauthorized and a JSON body containing an error message indicating invalid credentials.
3. WHEN a request to a `/api/v1/` endpoint omits the Authorization header, or provides an Authorization header that does not start with "Bearer ", THE API_Middleware SHALL respond with HTTP 401 Unauthorized and a JSON body containing an error message indicating missing authentication.
4. THE API_Middleware SHALL validate tokens by comparing the provided plaintext token against the stored bcrypt hash for the identified user.
5. THE API_Middleware SHALL bypass CSRF protection and session handling for all `/api/v1/` endpoints.
6. IF the user account associated with a valid token no longer exists in the database, THEN THE API_Middleware SHALL respond with HTTP 401 Unauthorized and a JSON body containing an error message indicating invalid credentials.

### Requirement 4: Read Matches Endpoint

**User Story:** As an API consumer, I want to retrieve the list of matches for the active event, so that I can determine which matches are available for tipping.

#### Acceptance Criteria

1. WHEN an authenticated API request is made to `GET /api/v1/matches`, THE Tipp_API SHALL return a JSON array of all matches belonging to the active event, ordered by start time ascending then by match ID ascending.
2. THE Tipp_API SHALL include for each match: the match ID, team names (team_a, team_b), start time in ISO 8601 format (UTC), match type, event phase number, whether the match is finished, and whether the match currently accepts tipps (true if the match start time is in the future and both team names are non-empty, false otherwise).
3. WHEN no active event exists, THE Tipp_API SHALL respond with HTTP 404 Not Found and a JSON error body indicating no active event is configured.
4. IF a match is finished, THEN THE Tipp_API SHALL include result fields (result_a, result_b) and, when available, extra-time results (result_aet_a, result_aet_b) and penalty results (result_apen_a, result_apen_b).
5. IF the API request is missing valid authentication credentials, THEN THE Tipp_API SHALL respond with HTTP 401 Unauthorized and a JSON error body indicating that authentication is required.
6. WHEN the active event has no matches, THE Tipp_API SHALL return an empty JSON array.

### Requirement 5: Read User Tipps Endpoint

**User Story:** As an API consumer, I want to retrieve my existing tipps, so that I can check which matches I have already tipped and avoid duplicate submissions.

#### Acceptance Criteria

1. WHEN an authenticated API request is made to `GET /api/v1/tipps`, THE Tipp_API SHALL return a JSON array of all tipps submitted by the authenticated user for the active event, where each entry contains the fields: match_id (integer), tipp_a (integer), tipp_b (integer), created (ISO 8601 UTC timestamp), and changed (ISO 8601 UTC timestamp).
2. WHEN the authenticated user has no tipps for the active event, THE Tipp_API SHALL return an empty JSON array with HTTP status 200.
3. IF no active event is configured in the system, THEN THE Tipp_API SHALL return an error response indicating that no active event exists.
4. IF the API request to `GET /api/v1/tipps` is not authenticated, THEN THE Tipp_API SHALL return HTTP status 401 with an error response indicating missing or invalid authentication.

### Requirement 6: Submit Tipp Endpoint

**User Story:** As an API consumer, I want to submit a tipp for a specific match, so that an AI player or script can participate in the tipping game.

#### Acceptance Criteria

1. WHEN an authenticated API request is made to `POST /api/v1/tipps` with a valid JSON body containing match_id (integer), tipp_a (integer), and tipp_b (integer), THE Tipp_API SHALL create or update the tipp for the authenticated user and that match, using an upsert pattern.
2. WHEN the specified match does not accept tipps (match start time is not in the future OR either team name is empty), THE Tipp_API SHALL respond with HTTP 409 Conflict and a JSON error body explaining the match is locked.
3. WHEN the specified match_id does not exist or does not belong to the active event, THE Tipp_API SHALL respond with HTTP 404 Not Found and a JSON error body.
4. WHEN tipp_a or tipp_b is not an integer in the range 0 to 99, THE Tipp_API SHALL respond with HTTP 422 Unprocessable Entity and a JSON error body listing the validation errors per field.
5. WHEN the tipp is successfully created or updated, THE Tipp_API SHALL respond with HTTP 200 OK and a JSON body containing the saved tipp data (match_id, tipp_a, tipp_b, created, changed).
6. IF the API request does not include valid authentication credentials, THEN THE Tipp_API SHALL respond with HTTP 401 Unauthorized and a JSON error body.
7. IF the request body is missing or is not valid JSON, THEN THE Tipp_API SHALL respond with HTTP 400 Bad Request and a JSON error body indicating a malformed request.
8. IF any of the required fields (match_id, tipp_a, tipp_b) are missing from the JSON body, THEN THE Tipp_API SHALL respond with HTTP 422 Unprocessable Entity and a JSON error body listing the missing fields.

### Requirement 7: API Scope Restriction

**User Story:** As a system operator, I want API tokens to be scoped exclusively to the owning user's data, so that one user cannot read or modify another user's tipps via the API.

#### Acceptance Criteria

1. THE Tipp_API SHALL only return tipps where the tipp's user_id matches the authenticated token owner's user_id when serving read requests.
2. THE Tipp_API SHALL only create or update tipps where the tipp's user_id matches the authenticated token owner's user_id when serving write requests.
3. THE API_Middleware SHALL derive the user identity exclusively from the validated token and SHALL reject any request that includes a user_id parameter in the URL path, query string, or request body.
4. IF a read request matches a tipp belonging to a different user, THEN THE Tipp_API SHALL exclude that tipp from the response without returning an error indication that reveals the tipp's existence.
5. IF a write request targets a tipp belonging to a different user, THEN THE Tipp_API SHALL reject the request and return an error response indicating insufficient permissions, without modifying any data.
6. IF the API token is missing, invalid, or expired, THEN THE API_Middleware SHALL reject the request and return an authentication error response before any data access occurs.

### Requirement 8: API Error Response Format

**User Story:** As an API consumer, I want consistent and machine-readable error responses, so that I can handle failures programmatically.

#### Acceptance Criteria

1. THE Tipp_API SHALL return all error responses as JSON objects with a Content-Type of `application/json` and an HTTP status code from the set {400, 401, 404, 409, 422, 500}.
2. THE Tipp_API SHALL include an "error" field in every error response body containing a non-empty string that describes the failure, with a maximum length of 256 characters.
3. WHEN validation errors occur, THE Tipp_API SHALL return HTTP status 422 and include a "fields" object in the response body mapping each invalid field name to a non-empty string describing the validation failure for that field.
4. IF an error is not a validation error, THEN THE Tipp_API SHALL omit the "fields" object from the error response body.
5. IF an internal server error occurs, THEN THE Tipp_API SHALL return HTTP status 500 with an "error" field and SHALL NOT expose internal implementation details in the error message.
