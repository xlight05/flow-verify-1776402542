# Overview

This project adds a new PATCH endpoint to an existing todo management API that allows clients to mark an individual todo item as completed. The endpoint provides a focused, partial-update mechanism so that callers do not need to send the full todo resource to toggle its completion state.

The target users are client applications (web, mobile, or other services) that consume the todo API and need a simple, reliable way to update the completion status of a todo. The high-level approach is to expose a dedicated PATCH route keyed by todo ID that updates only the completion flag and returns the updated resource.

# Capabilities

## Mark Todo as Completed
- Provide a PATCH endpoint that accepts a todo identifier in the URL path.
- Update the targeted todo's completion status to "completed" when the request succeeds.
- Optionally accept a boolean field in the request body to allow setting completion to true or false; default behavior marks as completed.
- Record or update a completion timestamp when a todo transitions to completed.
- Leave all other fields of the todo unchanged.
- Return the full updated todo resource in the response body.

## Request Handling
- Accept requests using the HTTP PATCH method only; reject other methods with an appropriate error.
- Accept and parse JSON request bodies.
- Validate the todo identifier format before processing.
- Ignore unknown fields in the request body or reject them consistently.

## Response Behavior
- Return HTTP 200 with the updated todo when the update succeeds.
- Return HTTP 404 when the specified todo does not exist.
- Return HTTP 400 when the request body or identifier is malformed.
- Return HTTP 405 for unsupported methods on the same path.
- Include a clear, machine-readable error message in all non-success responses.
- Use consistent JSON response formatting matching the rest of the API.

## Idempotency and State
- Marking an already-completed todo as completed must succeed without error and leave the todo in the completed state.
- The operation must be idempotent: repeated identical requests produce the same final state.
- Persist the updated completion status durably so it is reflected in subsequent GET requests.

## Authentication and Authorization
- Require the same authentication mechanism used by other endpoints in the API.
- Reject unauthenticated requests with HTTP 401.
- Ensure only users authorized to modify the targeted todo can complete it; return HTTP 403 otherwise.

## Validation and Error Handling
- Validate that the todo ID exists before attempting the update.
- Handle concurrent updates safely without corrupting todo state.
- Log errors with sufficient context for debugging while not exposing sensitive data in responses.

## Performance and Reliability
- Respond to typical requests within 300 milliseconds under normal load.
- Support the same concurrency and throughput characteristics as existing endpoints.
- Ensure the update is atomic — partial updates must not be persisted on failure.

## Documentation and Testing
- Document the new endpoint, including path, method, request body schema, response schema, and error codes.
- Provide example requests and responses in the API documentation.
- Cover the endpoint with automated tests for success, not-found, unauthorized, malformed input, and idempotency cases.
