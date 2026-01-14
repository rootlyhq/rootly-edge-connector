# Action Registration Payload Examples

Examples of what the Edge Connector sends to `POST /rec/v1/actions` when registering actions.

**All actions are registered**, regardless of trigger type (automatic or callable).

## Registration Behavior

**Backend categorizes actions based on the `trigger` field:**

**Automatic actions** (`on:` section in config):
- Trigger: `alert.created`, `incident.created`, etc.
- Backend categorizes as automatic based on trigger pattern
- **Show in UI as read-only** (visible but not clickable)
- Users can see what automations are configured
- No forms, no user interaction

**Callable actions** (`callable:` section in config):
- Trigger: `action.triggered`, `alert.action_triggered`, or `incident.action_triggered`
- Backend categorizes as callable based on trigger pattern
- **Show in UI as interactive buttons**
- Users can click and fill out parameter forms
- Trigger execution with user-provided inputs

**The Edge Connector sends all actions in a single `actions` array. The backend determines which are automatic and which are callable by examining the `trigger` field.**

## Quick Reference

**Config → API Field Mapping:**
```
config.id       → API.slug        (REQUIRED: machine identifier)
config.name        → API.name        (OPTIONAL: human-readable display)
config.description → API.description (OPTIONAL: multi-line text)
```

**Backend Storage:**
```
API.slug → backend.slug (unique identifier, used for matching)
API.name → backend.name (human display, if empty backend humanizes slug)
API.description → backend.description (rich text for UI)
```

**Example:**
```yaml
# Config
id: restart_server
name: "Restart Production Server"
description: "Restarts the server..."

# API JSON
{
  "slug": "restart_server",
  "name": "Restart Production Server",
  "description": "Restarts the server..."
}

# Backend stores
slug: "restart_server"  (used for lookups)
name: "Restart Production Server"  (displayed in UI)
description: "Restarts the server..."  (shown in UI)
```

**Parameter Types:**
```
type: "string"  → Text input (CANNOT have options)
type: "number"  → Numeric input (CANNOT have options)
type: "boolean" → Checkbox (CANNOT have options)
type: "list"    → Dropdown/select (MUST have options array)
```

**Important:** Only `type: "list"` can have the `options` field. All other types (`string`, `number`, `boolean`) cannot have `options` and will fail validation if you include it.

---

## Automatic Action Registration (Read-Only in UI)

**From actions.yml (on: section):**
```yaml
on:
  alert.created:
    script: /opt/scripts/handle-alert.sh
    parameters:
      alert_id: "{{ id }}"
      severity: "{{ labels.severity }}"
    timeout: 60

  incident.created:
    http:
      url: "https://hooks.slack.com/webhook"
      method: POST
      body: '{"text": "Incident: {{ title }}"}'
```

**Sent to API (POST /rec/v1/actions):**

```json
{
  "actions": [
    {
      "slug": "alert.created",
      "name": "",
      "action_type": "script",
      "trigger": "alert.created",
      "timeout": 60,
      "description": "Handles alert.created events",
      "parameters": []
    },
    {
      "slug": "incident.created",
      "name": "",
      "action_type": "http",
      "trigger": "incident.created",
      "timeout": 30,
      "description": "Handles incident.created events",
      "parameters": []
    }
  ]
}
```

**Backend categorizes these as automatic** because their `trigger` values (`alert.created`, `incident.created`) are not callable trigger patterns.

**Fields for automatic actions:**
- ✅ `slug` - Unique identifier (event type)
- ✅ `action_type` - "script" or "http" (for UI display/icon)
- ✅ `trigger` - Event type that triggers it
- ✅ `timeout` - Execution timeout (for monitoring/debugging)
- ✅ `description` - Optional description

**Fields NOT sent (execution details):**
- ❌ `script` path - Stays on connector
- ❌ `http` config (url, headers, body) - Stays on connector
- ❌ `parameters` definitions - No UI form needed

**UI Display Suggestions:**
- Badge: "🔄 Script: alert.created (60s timeout)"
- Icon based on action_type (script vs http)
- Show timeout for context
- Read-only, not clickable

---

## Callable Action Registration (Shows in UI)

**From actions.yml (callable: section):**
```yaml
callable:
  restart_server:                        # Action slug (key)
    name: Restart Production Server       # Display name in UI
    description: |                         # Description in UI
      Restarts the production server with graceful shutdown.

      Use this when the server becomes unresponsive.
    script: /opt/scripts/restart.sh
    trigger: alert.action_triggered        # Shows on alerts
    parameter_definitions:
      - name: service_name
        type: string
        required: true
        description: Service to restart
      - name: environment
        type: list
        required: false
        default: production
        options: [development, staging, production]
        description: Target environment
    timeout: 300
```

**Sent to API (POST /rec/v1/actions):**

```json
{
  "actions": [
    {
      "slug": "restart_server",
      "name": "Restart Production Server",
      "description": "Restarts the production server with graceful shutdown.\n\nUse this when the server becomes unresponsive.",
      "action_type": "script",
      "trigger": "alert.action_triggered",
      "timeout": 300,
      "parameters": [
        {
          "name": "service_name",
          "type": "string",
          "required": true,
          "description": "Service to restart"
        },
        {
          "name": "environment",
          "type": "list",
          "required": false,
          "description": "Target environment",
          "default": "production",
          "options": ["development", "staging", "production"]
        }
      ]
    }
  ]
}
```

**Backend categorizes this as callable** because the `trigger` value (`alert.action_triggered`) ends with `.action_triggered`.

**Fields for callable actions:**
- ✅ `slug` - Unique identifier
- ✅ `name` - Display name in UI
- ✅ `description` - Description in UI
- ✅ `action_type` - "script" or "http" (for UI display/icon)
- ✅ `trigger` - Event type (as simple string)
- ✅ `timeout` - Execution timeout (for monitoring/debugging)
- ✅ `parameters` - Parameter definitions for UI form

**Fields NOT sent (execution details):**
- ❌ `script` path - Stays on connector
- ❌ `http` config (url, headers, body) - Stays on connector
- ❌ Specific execution parameters/env - Stays on connector

**Why this split:**
- Backend gets UI-relevant metadata
- Connector keeps execution implementation details
- Backend shows what actions do, connector shows how they do it

**Field Mapping:**
- `callable.key` → `API.slug` → `backend.slug` (unique identifier)
- `callable.name` → `API.name` → `backend.name` (required for callable actions)
- `callable.description` → `API.description` → `backend.description`

---

## Callable HTTP Action Registration

**From actions.yml (callable: section):**
```yaml
callable:
  send_webhook:                          # Action slug (key)
    name: Send Webhook                    # Display name
    description: Send webhook notification
    trigger: action.triggered             # Standalone (default)
    http:
      url: "https://example.com/webhook"
      method: POST
      headers:
        Content-Type: application/json
        X-API-Key: "{{ env.API_KEY }}"
      params:
        source: rootly
      body: |
        {
          "message": "{{ parameters.message }}",
          "severity": "{{ parameters.severity }}"
        }
    parameter_definitions:
      - name: message
        type: string
        required: true
        description: Message to send
      - name: severity
        type: list
        required: false
        default: info
        options: [info, warning, critical]
    timeout: 30
```

**Sent to API (POST /rec/v1/actions):**

```json
{
  "actions": [
    {
      "slug": "send_webhook",
      "name": "Send Webhook",
      "description": "Send webhook notification",
      "action_type": "http",
      "trigger": "action.triggered",
      "timeout": 30,
      "parameters": [
        {
          "name": "message",
          "type": "string",
          "required": true,
          "description": "Message to send"
        },
        {
          "name": "severity",
          "type": "list",
          "required": false,
          "default": "info",
          "options": ["info", "warning", "critical"],
          "description": "Message severity"
        }
      ]
    }
  ]
}
```

**Backend categorizes this as callable** because the `trigger` value is `action.triggered`.

**Fields NOT sent (execution details):**
- ❌ `script` path - Stays on connector
- ❌ `http` config (url, headers, body) - Stays on connector
- ❌ Specific execution parameters/env - Stays on connector

**Backend receives:**
- Slug, name, description (identification/display)
- action_type, timeout (monitoring/UI icons)
- Trigger (event type)
- Parameters (UI form generation)

---

## Example: Name Optional (Backend Humanizes)

**From actions.yml:**
```yaml
- id: clear_cache         # REQUIRED
  # name not provided - backend will humanize to "Clear Cache"
  description: "Clears the application cache"
  type: http
  trigger:
    event_type: "action.triggered"
  http:
    url: "https://api.example.com/cache/clear"
    method: POST
  timeout: 30
```

**Sent to API:**
```json
{
  "actions": [
    {
      "slug": "clear_cache",
      "action_type": "http",
      "description": "Clears the application cache",
      "timeout": 30,
      "trigger": {
        "event_types_trigger": ["action.triggered"]
      },
      "parameters": [],
      "http": {
        "url": "https://api.example.com/cache/clear",
        "method": "POST"
      }
    }
  ]
}
```

**Backend will:**
- Store `slug: "clear_cache"`
- Store `name: "clear_cache"` (uses slug as name since name not provided)
- Store `description: "Clears the application cache"`

---

## Multiple Actions Registration

**All actions are registered (both automatic and callable):**

```yaml
on:
  alert.created:
    script: /opt/scripts/handle-alert.sh

callable:
  scale_service:
    name: Scale Service
    description: Scale service capacity
    trigger: alert.action_triggered
    script: /opt/scripts/scale.sh
    parameter_definitions:
      - name: target_capacity
        type: number
        required: true

  clear_cache:
    name: Clear Cache
    http:
      url: "https://api.example.com/cache/clear"
      method: POST
    parameter_definitions:
      - name: cache_type
        type: list
        options: [redis, memcached]
```

**Sent to API:**
```json
{
  "actions": [
    {
      "slug": "alert.created",
      "name": "",
      "action_type": "script",
      "trigger": "alert.created",
      "timeout": 30,
      "parameters": []
    },
    {
      "slug": "scale_service",
      "name": "Scale Service",
      "description": "Scale service capacity",
      "action_type": "script",
      "trigger": "alert.action_triggered",
      "timeout": 30,
      "parameters": [
        {
          "name": "target_capacity",
          "type": "number",
          "required": true
        }
      ]
    },
    {
      "slug": "clear_cache",
      "name": "Clear Cache",
      "action_type": "http",
      "trigger": "action.triggered",
      "timeout": 30,
      "parameters": [
        {
          "name": "cache_type",
          "type": "list",
          "options": ["redis", "memcached"]
        }
      ]
    }
  ]
}
```

**Backend categorizes based on trigger:**
- `alert.created` → **Automatic** (trigger is not `.action_triggered` pattern)
- `scale_service` → **Callable** (trigger is `alert.action_triggered`)
- `clear_cache` → **Callable** (trigger is `action.triggered`)

---

## Response Format

**Success (201 Created):**

```json
{
  "registered": 3,
  "failed": 0,
  "deleted": 0,
  "automatic_count": 1,
  "callable_count": 2,
  "registered_actions": {
    "automatic": ["alert.created"],
    "callable": ["scale_service", "clear_cache"]
  },
  "failures": []
}
```

**Key response fields:**
- `registered` - Total count of successfully registered actions
- `failed` - Count of actions that failed validation
- `deleted` - Count of actions deleted during sync (not in new request)
- `automatic_count` - Count of actions categorized as automatic
- `callable_count` - Count of actions categorized as callable
- `registered_actions.automatic` - Array of automatic action slugs
- `registered_actions.callable` - Array of callable action slugs
- `failures` - Array of failure details with `slug` and `reason`

**Partial failure (207 Multi-Status):**
```json
{
  "registered": 1,
  "failed": 1,
  "deleted": 0,
  "automatic_count": 0,
  "callable_count": 1,
  "registered_actions": {
    "automatic": [],
    "callable": ["scale_service"]
  },
  "failures": [
    {
      "slug": "clear_cache",
      "reason": "Callable actions must have a name for UI display"
    }
  ]
}
```

---

## What Gets Registered

**All actions are registered with the backend for visibility and audit**, including:

### Callable Actions (Show in UI)

**Triggers ending in `.action_triggered`** - Backend generates UI forms from `parameters`:
- `alert.action_triggered` - Actions on alerts (e.g., "Restart Service", "Escalate")
- `incident.action_triggered` - Actions on incidents (e.g., "Page Oncall", "Run Playbook")
- `action.triggered` - Standalone actions (e.g., "Clear Cache", "Deploy Hotfix")

### Automatic Actions (Background Only)

**All other event triggers** - No UI, run automatically:
- `alert.created`, `alert.updated` - Triggered when alerts are created/updated
- `incident.created`, `incident.updated` - Triggered when incidents are created/updated

## Why Register All Actions?

**Benefits of registering everything:**
- ✅ **Visibility** - Backend sees all configured automations
- ✅ **Audit trail** - Track what each connector is doing
- ✅ **Monitoring** - Know which connectors have which actions
- ✅ **Debugging** - See full connector configuration in one place

## How Backend Differentiates

The backend categorizes actions based on the `trigger` field:

| Trigger Pattern | Backend Behavior |
|-----------------|------------------|
| `action.triggered` or `*.action_triggered` | Show in UI as **interactive button**, generate form fields from `parameters`, allow user triggering |
| All other event types | Show in UI as **read-only badge**, display info only, no user interaction |

**Simple rule:**
- Trigger = `action.triggered` or ends with `.action_triggered` → **Callable** (clickable button with form)
- Trigger = any other event type → **Automatic** (visible badge, read-only)

**Both are visible in UI**, but only callable actions can be triggered by users.

---

## Complete Example: Automatic + Callable

**Config (actions.yml):**
```yaml
defaults:
  timeout: 30

on:
  alert.created:
    script: /opt/scripts/handle-alert.sh
    parameters:
      alert_id: "{{ id }}"

  incident.created:
    http:
      url: "https://slack.com/webhook"
      body: '{"text": "{{ title }}"}'

callable:
  restart_service:
    name: Restart Service
    description: Gracefully restart a production service
    trigger: alert.action_triggered
    script: /opt/scripts/restart.sh
    parameter_definitions:
      - name: service_name
        type: string
        required: true
      - name: force
        type: boolean
        default: false

  clear_cache:
    name: Clear Cache
    script: /opt/scripts/clear-cache.sh
    parameter_definitions:
      - name: cache_type
        type: list
        options: [redis, memcached, all]
        default: redis
```

**API Payload (POST /rec/v1/actions):**
```json
{
  "actions": [
    {
      "slug": "alert.created",
      "action_type": "script",
      "trigger": "alert.created",
      "timeout": 30,
      "description": "Handles alert.created events"
    },
    {
      "slug": "incident.created",
      "action_type": "http",
      "trigger": "incident.created",
      "timeout": 30,
      "description": "Handles incident.created events"
    },
    {
      "slug": "restart_service",
      "name": "Restart Service",
      "description": "Gracefully restart a production service",
      "action_type": "script",
      "trigger": "alert.action_triggered",
      "timeout": 30,
      "parameters": [
        {
          "name": "service_name",
          "type": "string",
          "required": true
        },
        {
          "name": "force",
          "type": "boolean",
          "default": false
        }
      ]
    },
    {
      "slug": "clear_cache",
      "name": "Clear Cache",
      "action_type": "script",
      "trigger": "action.triggered",
      "timeout": 30,
      "parameters": [
        {
          "name": "cache_type",
          "type": "list",
          "options": ["redis", "memcached", "all"],
          "default": "redis"
        }
      ]
    }
  ]
}
```

**Backend categorization:**
- `alert.created`, `incident.created` → **Automatic** (trigger is event type)
- `restart_service` → **Callable** (trigger is `alert.action_triggered`)
- `clear_cache` → **Callable** (trigger is `action.triggered`)

**Key points:**
1. Single `actions` array (backend categorizes by `trigger` field)
2. All actions include `action_type` and `timeout` (for UI display/monitoring)
3. Trigger as simple string (not nested object)
4. Execution details stay on connector (script path, http config, etc.)

---

## Testing Locally

See what gets sent:

```bash
# Start connector with TRACE logging
REC_LOG_LEVEL=trace ./bin/rootly-edge-connector

# Look for:
TRACE HTTP request body:
{
  "actions": [...]
}  method=POST url="http://host.docker.internal:3000/rec/v1/actions"
```

Check registration result:
```bash
# Look for:
INFO Registered actions with backend  action_count=10
INFO Action registration response  registered=10 failed=0
```

The count includes ALL actions from your config file (both automatic and user-triggered).
