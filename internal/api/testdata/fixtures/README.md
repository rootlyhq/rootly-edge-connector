# Test Fixtures

This directory contains JSON fixtures for API client tests.

## Fixtures

### Event Payloads

- `events_alert_and_incident.json` - Contains two events: an alert.created and an incident.created event. Matches the structure returned by the Rootly API `/rec/v1/deliveries` endpoint. Includes full serializer data with services, environments, functionalities, and severity information.

- `events_action_triggered.json` - Contains three action triggered events demonstrating all action types:
  - `alert.action_triggered` - Action triggered on an alert entity
  - `incident.action_triggered` - Action triggered on an incident entity
  - `action.triggered` - Standalone action not tied to an entity

  Each includes action parameters, entity references, and triggered_by user information.

## Usage

These fixtures are loaded in tests using helper functions:

```go
func loadFixture(t *testing.T, filename string) api.EventsResponse {
    data, err := os.ReadFile("testdata/fixtures/" + filename)
    require.NoError(t, err)

    var response api.EventsResponse
    err = json.Unmarshal(data, &response)
    require.NoError(t, err)

    return response
}
```

## Maintenance

When the Rootly API changes:
1. Update fixture files to match new API structure
2. Run tests to identify what needs updating
3. Update assertions and code as needed

This ensures tests accurately reflect real API behavior.
