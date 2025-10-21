# Test Fixtures

This directory contains fixtures for end-to-end integration tests.

## Fixtures

### Event Payloads

- `event_alert_for_script.json` - Alert.created event with full data structure for testing script execution. Contains alert data with host information, services, and environments. Used to test script parameter substitution and execution workflow.

- `event_alert_action_triggered.json` - Alert.action_triggered event for testing user-triggered actions. Includes action metadata (id, name, slug) and demonstrates action matching logic.

- `event_alert_simple.json` - Minimal alert.created event with only required fields. Used for testing multiple actions matching the same event type.

- `event_alert_no_match.json` - Alert.created event with minimal data. Used for testing the "no matching action" scenario.

- `event_alert_for_failure.json` - Alert.created event for testing script execution failures and error reporting.

- `event_incident_for_http.json` - Full incident.created event payload used for testing HTTP action execution in integration tests. Contains complete incident data including:
  - Sequential ID and metadata (title, slug, summary)
  - Status tracking (detected_at, started_at, etc.)
  - Associated resources (services, environments, functionalities)
  - Severity information

  This fixture is used to test the complete workflow: API polling → event processing → HTTP executor → result reporting.

## Usage

These fixtures are loaded in integration tests to simulate real API responses:

```go
// Load fixture for mock API server
fixture := loadFixture(t, "event_incident_for_http.json")
mockServer.SetResponse(fixture)

// Test full workflow
client := api.NewClient(mockServer.URL(), "test-token")
poller := poller.New(client, config)
// ... test polling, execution, reporting
```

## Purpose

Integration test fixtures differ from unit test fixtures:
- **Unit test fixtures**: Test individual components in isolation
- **Integration test fixtures**: Test complete workflows end-to-end

These fixtures should match realistic production scenarios to ensure the system behaves correctly in real-world conditions.

## Maintenance

When adding new integration tests:
1. Create fixtures that represent real production scenarios
2. Use realistic data structures matching backend serializers
3. Include all necessary fields for the workflow being tested
4. Document the test scenario in this README

When API changes:
1. Update fixtures to match new API responses
2. Run integration tests to verify end-to-end behavior
3. Update test expectations as needed
