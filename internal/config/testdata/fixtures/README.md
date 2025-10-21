# Test Fixtures

This directory contains YAML configuration fixtures for config loading and validation tests.

## Fixtures

### Valid Configuration

- `valid_actions.yml` - Minimal valid actions configuration demonstrating both script and HTTP action types. Contains proper field defaults, triggers, and timeout values. Used for testing successful config loading.

- `valid_parameter_definitions.yml` - Comprehensive examples of all valid parameter types (string, number, boolean, list) with proper configurations. Demonstrates correct usage of defaults, options, and required fields.

### Edge Cases and Error Scenarios

- `action_with_id_and_name.yml` - Tests the handling of actions with both `id` (required machine identifier) and `name` (optional human-readable display) fields. Used to verify that `id` is the primary identifier for matching and `name` is optional for UI display.

- `duplicate_ids.yml` - Invalid configuration with duplicate action IDs. Used to test validation that rejects configurations with non-unique action identifiers.

### Parameter Definition Validation

- `list_with_invalid_default.yml` - Invalid configuration where a list parameter's default value is not in the options array. Used to test validation that ensures default values must be one of the available options for list types.

- `list_with_duplicate_options.yml` - Invalid configuration with duplicate values in a list parameter's options array. Used to test validation that ensures all options are unique.

- `string_with_options.yml` - Invalid configuration where a string parameter has an options field (only list type can have options). Used to test type-specific validation rules enforced by the JSON Schema.

## Usage

These fixtures are loaded in tests using the config loader:

```go
cfg, err := config.LoadActions("testdata/fixtures/valid_actions.yml")
require.NoError(t, err)
assert.Len(t, cfg.Actions, 2)
```

For error testing:

```go
_, err := config.LoadActions("testdata/fixtures/duplicate_ids.yml")
require.Error(t, err)
assert.Contains(t, err.Error(), "duplicate")
```

## Adding New Fixtures

When adding new test scenarios:
1. Create a descriptive filename using underscores (e.g., `missing_required_field.yml`)
2. Add realistic action configurations that match actual use cases
3. Document the fixture in this README
4. Ensure YAML is properly formatted and validated

## Maintenance

When config schema changes:
1. Update existing fixtures to match new structure
2. Run tests to identify breaking changes
3. Add new fixtures for new validation rules
4. Keep fixtures minimal but realistic
