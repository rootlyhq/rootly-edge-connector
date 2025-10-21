# Config Format Migration Progress

**Status:** ✅ Core implementation complete, ready for full migration

## What's Been Done

### 1. New Simplified Format Implemented ✅

**New config structure:**
```yaml
defaults:
  timeout: 30
  source_type: local
  env:
    ENVIRONMENT: production

on:                      # Automatic actions (event type = key, 1:1 mapping)
  alert.created:
    script: /opt/scripts/handle-alert.sh
    parameters:
      alert_id: "{{ id }}"
      severity: "{{ labels.severity }}"

callable:                # Callable actions (action slug = key)
  restart_service:
    name: Restart Service
    description: Restart a production service
    trigger: alert.action_triggered
    script: /opt/scripts/restart.sh
    parameter_definitions:
      - name: service_name
        type: string
        required: true
    # parameters: auto-generated from parameter_definitions!
```

**Benefits:**
- 50-60% less YAML than old format
- Clear separation: `on:` (automatic) vs `callable:` (user-triggered)
- No redundant `id` field (key IS the identifier)
- Auto-detect `type` from `http:` field presence
- Auto-generate `parameters` from `parameter_definitions`
- Global `defaults` reduce repetition

### 2. Implementation Complete ✅

**Files modified:**
- `internal/config/config.go` - New structs: OnAction, CallableAction, ActionDefaults
- `internal/config/loader.go` - Calls ConvertToActions() after parsing
- `internal/config/validator.go` - Allow dots in IDs (for alert.created)
- All test fixtures updated to new format
- 8 new tests for conversion logic

**Code additions:**
- `ConvertToActions()` - Transforms on/callable → internal Action array
- `onActionToAction()` - Converts automatic actions
- `callableActionToAction()` - Converts callable actions
- `autoGenerateParameters()` - Maps parameter_definitions → parameters
- `mergeEnv()` - Merges global + local env vars
- Helper functions for defaults

### 3. Example File Created ✅

**`actions.simple.yml`** - Full working example demonstrating:
- `on:` section with alert.created, incident.created
- `callable:` section with 3 different actions
- Global defaults usage
- Auto-detection and auto-generation
- Different trigger types (alert.action_triggered, incident.action_triggered, action.triggered)

### 4. Tests Passing ✅

- All 117 config tests pass
- All conversion tests pass
- Validation works with new format
- `--validate` flag shows nice output

### 5. Documentation Updated ✅

**Updated files:**
- `docs/api/01-action-registration.md` - New format examples, clarified UI behavior
- `docs/api/02-event-payloads.md` - Updated terminology (automatic vs callable)
- `docs/api/03-event-examples.md` - Updated section titles

**Key clarifications:**
- Automatic actions show in UI as **read-only badges** (visible but not clickable)
- Callable actions show in UI as **interactive buttons** (clickable with forms)
- Both are registered for visibility

## What's Left To Do

### 1. Migrate Example Files 🔴

Current example files still use old format:
- [ ] `actions.example.yml` - Convert to on:/callable: format
- [ ] `actions.example.dev.yml` - Convert to on:/callable: format
- [ ] Update comments and documentation in example files

### 2. Update Main README 🔴

- [ ] Update Quick Start section with new format
- [ ] Update action examples throughout README
- [ ] Add migration guide section
- [ ] Update template variable examples

### 3. Create Migration Guide 🔴

Create `docs/user-guide/migration-v2.md`:
- [ ] Old format → New format conversion guide
- [ ] Side-by-side examples
- [ ] Common patterns (automatic actions, callable actions, HTTP actions)
- [ ] Automated migration script (optional)

### 4. Update All Documentation 🔴

Files that need updates:
- [ ] `docs/development/development.md` - Update examples
- [ ] `docs/user-guide/systemd-installation.md` - Update action examples
- [ ] `CLAUDE.md` - Update guidance for Claude Code

### 5. Integration Tests 🔴

- [ ] Update integration test configs to new format (tests/integration/e2e_test.go)
- [ ] Ensure all integration tests pass with new format

### 6. Docker Files 🔴

- [ ] Update Dockerfile to use new example format
- [ ] Update Dockerfile.dev
- [ ] Update docker-compose examples if any

## Key Features of New Format

### Auto-Detection

**Type detection:**
```yaml
# No need to specify type: script or type: http
callable:
  webhook:
    http:        # ← Presence of http: auto-detects type: http
      url: "..."

  script_action:
    script: "..." # ← Script path auto-detects type: script
```

### Auto-Generation

**Parameter mappings:**
```yaml
# OLD: Manual repetition
parameter_definitions:
  - name: service_name
    type: string
parameters:
  service_name: "{{ parameters.service_name }}"  # Had to repeat!

# NEW: Auto-generated
parameter_definitions:
  - name: service_name
    type: string
# parameters: automatically becomes { service_name: "{{ parameters.service_name }}" }
```

**Only specify parameters when you need CUSTOM mappings:**
```yaml
callable:
  restart:
    parameter_definitions:
      - name: service_name
        type: string
    parameters:
      service: "{{ parameters.service_name }}"  # Custom: service (not service_name)
      alert_id: "{{ entity_id }}"               # Additional non-UI parameter
      triggered_by: "{{ triggered_by.email }}"  # Additional non-UI parameter
```

### Global Defaults

```yaml
defaults:
  timeout: 30
  source_type: local
  env:
    LOG_LEVEL: info
    REGION: us-west-2

# All actions inherit these unless overridden
callable:
  action1:
    script: /path/to/script.sh
    timeout: 60  # Override: uses 60 instead of 30

  action2:
    script: /path/to/other.sh
    # Uses default timeout: 30
```

## Backward Compatibility

**BREAKING CHANGE:** Old `actions:` array format is NO LONGER supported.

Users must migrate to `on:`/`callable:` format.

## Testing the New Format

```bash
# Validate your new config
./bin/rootly-edge-connector --validate -config config.yml -actions actions.yml

# Example output shows:
# ✅ Actions configuration is valid
#    Total actions: 5
#
# 📊 Action Summary:
# ┌────────────────┬──────────┬───────────┬─────────────────────┐
# │ ID             │ Type     │ Source    │ Trigger             │
# ├────────────────┼──────────┼───────────┼─────────────────────┤
# │ alert.created  │ script   │ local     │ alert.created       │
# │ restart        │ script   │ local     │ action.triggered    │
# └────────────────┴──────────┴───────────┴─────────────────────┘
#
# 📞 Callable Actions (with parameter_definitions): 1
#    • restart (2 parameters)
```

## Next Session TODO

1. **Update example files** - Convert actions.example.yml and actions.example.dev.yml
2. **Create migration guide** - docs/user-guide/migration-v2.md
3. **Update README** - Main README.md with new format
4. **Update all docs** - CLAUDE.md, development docs, user guides
5. **Integration tests** - Update e2e tests to use new format
6. **Docker files** - Update Dockerfiles with new example configs

## Technical Notes

### Conversion Logic

The `ConvertToActions()` method transforms the new format to internal Action array:

1. **on: actions** → Action with ID = event type, no parameter_definitions
2. **callable: actions** → Action with ID = slug, has parameter_definitions
3. **Auto-detection** → type: http if http: present, else script
4. **Auto-generation** → parameters from parameter_definitions if not specified
5. **Defaults merging** → global defaults + action-specific overrides

### Matching Logic (Already Implemented)

For `action.triggered` events:
- Checks `event.Action.Slug` (if Action metadata exists)
- Falls back to `event.Data["action_name"]` (if Action is null)
- Matches against action's slug (the key in `callable:` section)

### Validation

- IDs now allow dots (for event types like `alert.created`)
- All existing validations still work
- parameter_definitions schema validation unchanged

## Commits This Session

1. `a357b13` - JSON Schema validation for parameter_definitions
2. `f8719fe` - List defaults/duplicates validation
3. `7472ddd` - Complete alias → id terminology cleanup
4. `dbdddd8` - Fixed flaky worker pool test
5. `88e4a37` - RPM/DEB package publishing
6. `2bf9b8e` - Fixed flaky script test
7. `bbe037b` - Added 5 new platforms (9 total)
8. `bebb890` - Removed redundant CI build job
9. `7c7a70f` - Improved .gitignore
10. `c2dfb47` - Fixed Windows path tests
11. `4200214` - Extracted integration test fixtures
12. `e7cb5dc` - Regression tests for action_name
13. `ccde58d` - Added --validate flag
14. `d16c45b` - data.action_name fallback matching
15. `9947cc0` - Updated API docs terminology
16. `ace78e7` - **NEW SIMPLIFIED CONFIG FORMAT** 🎉
17. `459b8d4` - Clarified UI visibility (automatic actions show as read-only)
18. `3e79aad` - Added this migration progress tracker
19. `f51191f` - Proposed simplified API registration payload
20. `0fed3ce` - Updated event examples to new format

## Success Metrics

- ✅ 50-60% reduction in config file size
- ✅ Clear on:/callable: separation
- ✅ Auto-detection and auto-generation working
- ✅ All 117 tests passing
- ✅ Validation working with nice output
- ✅ Documentation updated

**Ready to continue migration!**
