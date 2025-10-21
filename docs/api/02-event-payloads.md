# API Payload Format Specification

This document defines the event payload format for `GET /rec/v1/deliveries`.

## Event Types Overview

### Event Types

**Automatic triggers** (configured in `on:` section):
- `alert.created`
- `alert.updated`
- `incident.created`
- `incident.updated`

**Callable actions** (configured in `callable:` section):

```json
// For actions on alerts
{"event_type": "alert.action_triggered"}

// For actions on incidents
{"event_type": "incident.action_triggered"}

// For standalone actions (not tied to any entity)
{"event_type": "action.triggered"}
```

**Benefits:**
- ✅ Immediately clear what entity type you're working with
- ✅ Consistent naming: `alert.*`, `incident.*`, `action.*`
- ✅ No need for separate entity_type field - it's in the event name!
- ✅ Supports standalone actions (cache clear, health checks, etc.)

### Action Metadata Structure

**For alert.action_triggered and incident.action_triggered:**

Backend sends Action metadata at top level (recommended):

```json
{
  "event_type": "alert.action_triggered",
  "action": {
    "id": "action-uuid",
    "name": "Restart Service",
    "slug": "restart_service"
  },
  "data": {
    "entity_id": "alert-uuid",
    "parameters": {...},
    "triggered_by": {...}
  }
}
```

**For action.triggered (standalone):**

**Required fields:**
- **`action`** (object): Action metadata at top level (same as alert/incident actions)
  - **`action.slug`** (string): Action identifier (matches key in `callable:` section)
  - **`action.name`** (string): Display name
  - **`action.id`** (string, UUID): Action UUID in Rootly
- **`data.parameters`** (object): User-provided parameters from UI
- **`data.triggered_by`** (object): User who triggered (id, name, email)
- No `entity_id` - standalone action (not tied to alert/incident)

---

## Event Payload Structure

This document defines the expected JSON payload format for events from `GET /rec/v1/deliveries`.

### Standard Event Format

**For action_triggered events:**

```json
{
  "id": "7ff2efb5-cc16-4fd5-a350-ba790fa91b13",
  "event_id": "82a9699f-3a68-49a8-9ae6-d0c04e91f956",
  "event_type": "alert.action_triggered",
  "timestamp": "2025-10-26T14:26:55Z",
  "action": {
    "id": "01939a0e-7c4a-7e4e-b5e1-5c3f8e2c9a1b",
    "name": "Restart Test Service",
    "slug": "restart_test_service"
  },
  "data": {
    "entity_id": "alert-uuid-789",
    "parameters": {
      "environment": "development",
      "service_name": "oops"
    },
    "triggered_by": {
      "id": 50,
      "name": "Quentin Rousseau",
      "email": "quentin@rootly.com"
    }
  }
}
```

### Key Fields

#### Top Level (all events)
- **`id`** (string, UUID): Delivery ID - unique per connector instance
- **`event_id`** (string, UUID): Event ID - shared across all connectors
- **`event_type`** (string): Event type
  - Automatic: `alert.created`, `alert.updated`, `incident.created`, `incident.updated`
  - User-triggered: `alert.action_triggered`, `incident.action_triggered`, `action.triggered`
- **`timestamp`** (string, ISO8601): When the event occurred
- **`action`** (object, optional): Action metadata (only for action_triggered events)
  - **`action.id`** (string, UUID): Action UUID in Rootly
  - **`action.name`** (string): Human-readable display name
  - **`action.slug`** (string): Machine identifier (matches trigger.action_name)
- **`data`** (object): Event payload data

#### Data Fields (varies by event type)

**For alert.action_triggered:**
- **`action`** (object): Action metadata at top level
  - **`action.slug`** (string): Must match `trigger.action_name` in actions.yml
- **`data.entity_id`** (string, UUID): Alert UUID
- **`data.parameters`** (object): User-provided parameters from UI
- **`data.triggered_by`** (object): User who triggered (id, name, email)

**For incident.action_triggered:**
- **`action`** (object): Action metadata at top level
  - **`action.slug`** (string): Must match `trigger.action_name` in actions.yml
- **`data.entity_id`** (string, UUID): Incident UUID
- **`data.parameters`** (object): User-provided parameters from UI
- **`data.triggered_by`** (object): User who triggered (id, name, email)

**For action.triggered (standalone callable actions):**

```json
{
  "event_type": "action.triggered",
  "action": {
    "id": "action-uuid",
    "name": "Clear Cache",
    "slug": "clear_cache"
  },
  "data": {
    "parameters": {...},
    "triggered_by": {...}
  }
}
```

**Required fields:**
- **`action`** (object): Action metadata at top level
  - **`action.slug`** (string): Action identifier (matches key in `callable:` section)
  - **`action.name`** (string): Display name
  - **`action.id`** (string, UUID): Action UUID in Rootly
- **`data.parameters`** (object): User-provided parameters from UI
- **`data.triggered_by`** (object): User who triggered (id, name, email)
- No `entity_id` - standalone action (not tied to alert/incident)

**Matching logic:**
- Connector matches `action.slug` against action keys in `callable:` section
- Action metadata must be present for proper matching

**For alert.created / alert.updated:**
- **`data.id`** (string, UUID): Alert ID
- **`data.summary`** (string): Alert summary
- **`data.status`** (string): Alert status
- **`data.source`** (string): Source system
- **`data.created_at`** (string, ISO8601): When alert was created
- **`data.services`** (array): Associated services
- **`data.environments`** (array): Associated environments

**For incident.created / incident.updated:**
- **`data.id`** (string, UUID): Incident ID
- **`data.title`** (string): Incident title
- **`data.severity`** (string): Incident severity
- **`data.status`** (string): Incident status
- ... (similar to alerts)

## Template Variable Access

**All actions use Liquid template syntax** for consistency:
- Script actions: `{{ field }}` in the `parameters` section
- HTTP actions: `{{ field }}` in URL, headers, params, and body

With the flat structure, templates work intuitively:

```yaml
parameters:
  # Action metadata (from top-level action object)
  action_name: "{{ action.name }}"          # Display name
  action_slug: "{{ action.slug }}"          # Machine identifier
  action_id: "{{ action.id }}"              # Action UUID

  # Event data fields
  user_email: "{{ triggered_by.email }}"    # Nested access
  environment: "{{ parameters.environment }}" # User parameter
  entity_id: "{{ entity_id }}"              # Alert/incident UUID
```

**Benefits:**
- ✅ Consistent syntax across all action types
- ✅ Powerful Liquid filters: `{{ services | map: "name" | join: ", " }}`
- ✅ No need to remember dot prefix (unlike Go templates)
- ✅ Clear separation: `action` metadata vs `data` payload

## Important: No Double Nesting

**❌ DO NOT use this structure:**
```json
{
  "data": {
    "data": {              // ← Double nesting - BAD!
      "action_name": "..."
    },
    "event": {...}
  }
}
```

**✅ USE this structure:**
```json
{
  "data": {
    "action_name": "..."   // ← Direct access - GOOD!
  }
}
```

The event metadata (`type`, `event_id`, `timestamp`) is already at the top level, so there's no need to duplicate it inside `data.event`.

## Example Payloads

### alert.action_triggered (User Action on Alert)

```json
{
  "id": "delivery-uuid",
  "event_id": "event-uuid",
  "event_type": "alert.action_triggered",
  "timestamp": "2025-10-26T14:26:55Z",
  "action": {
    "id": "01939a0e-7c4a-7e4e-b5e1-5c3f8e2c9a1b",
    "name": "Restart Test Service",
    "slug": "restart_test_service"
  },
  "data": {
    "entity_id": "alert-uuid",           // ← Alert UUID
    "parameters": {
      "service_name": "api",
      "environment": "production"
    },
    "triggered_by": {
      "id": 50,
      "name": "Quentin Rousseau",
      "email": "quentin@rootly.com"
    }
  }
}
```

**Why separate `action` object?**
- Clear separation between action metadata and event data
- Access action name: `{{ action.name }}` or slug: `{{ action.slug }}`
- No confusion between `data.parameters` and action identifier

**Why `entity_id`?**
- Scripts can fetch full alert details: `GET /v1/alerts/$ENTITY_ID`
- Links action execution to the specific alert
- Useful for logging: "Action {{ action.name }} was run on Alert {{ entity_id }}"

### alert.created (Automatic Trigger)

Based on `EdgeConnectors::AlertSerializer`:

```json
{
  "id": "delivery-uuid",
  "event_id": "event-uuid",
  "event_type": "alert.created",
  "timestamp": "2025-10-26T15:02:40Z",
  "data": {
    "id": "6aeb35ae-ca31-4bcf-91bd-c4ecce44dedc",
    "source": "datadog",
    "summary": "High database latency detected",
    "status": "open",
    "labels": {
      "severity": "critical",
      "component": "database",
      "region": "us-west-2"
    },
    "data": {
      "host": "prod-db-01.example.com",
      "latency_ms": 1500,
      "threshold_ms": 500,
      "monitor_id": "12345678",
      "tags": ["env:production", "service:database"]
    },
    "started_at": "2025-10-26T15:01:30Z",
    "ended_at": null,
    "created_at": "2025-10-26T15:02:40Z",
    "updated_at": "2025-10-26T15:02:40Z",
    "services": [
      {
        "id": "service-uuid-1",
        "name": "DB - Production Database",
        "slug": "db-production"
      }
    ],
    "environments": [
      {
        "id": "env-uuid-1",
        "name": "Production",
        "slug": "production",
        "color": "#E74C3C"
      }
    ]
  }
}
```

**Alert Fields:**
- `id` - Alert UUID
- `source` - Source system (datadog, pagerduty, newrelic, etc.)
- `summary` - Alert description
- `status` - Alert status (open, acknowledged, resolved)
- `labels` - Key-value labels (severity, component, etc.)
- `data` - Source-specific custom data (monitor info, tags, metrics)
- `started_at` - When alert started (ISO8601)
- `ended_at` - When alert ended (null if still active)
- `created_at` - When alert was created in Rootly
- `updated_at` - Last update time
- `services` - Associated services (can be empty)
- `environments` - Associated environments (can be empty)

### incident.created

Based on `EdgeConnectors::IncidentSerializer`:

```json
{
  "id": "delivery-uuid",
  "event_id": "event-uuid",
  "event_type": "incident.created",
  "timestamp": "2025-10-26T16:00:00Z",
  "data": {
    "id": "incident-uuid-123",
    "sequential_id": 42,
    "title": "Production API Gateway Outage",
    "slug": "production-api-gateway-outage",
    "summary": "Complete outage affecting all customers",
    "status": "started",
    "kind": "normal",
    "private": false,
    "detected_at": "2025-10-26T15:58:00Z",
    "acknowledged_at": null,
    "started_at": "2025-10-26T16:00:00Z",
    "mitigated_at": null,
    "resolved_at": null,
    "cancelled_at": null,
    "created_at": "2025-10-26T16:00:00Z",
    "updated_at": "2025-10-26T16:00:00Z",
    "services": [
      {
        "id": "service-uuid-1",
        "name": "API Gateway",
        "slug": "api-gateway"
      }
    ],
    "environments": [
      {
        "id": "env-uuid-1",
        "name": "Production",
        "slug": "production",
        "color": "#E74C3C"
      }
    ],
    "functionalities": [
      {
        "id": "func-uuid-1",
        "name": "API Requests",
        "slug": "api-requests"
      }
    ],
    "severity": {
      "id": "sev-uuid-1",
      "name": "SEV1",
      "slug": "sev1",
      "color": "#FF0000"
    }
  }
}
```

**Incident Fields:**
- `id` - Incident UUID
- `sequential_id` - Human-readable number (e.g., #42)
- `title` - Incident title
- `slug` - URL-friendly slug
- `summary` - Detailed description
- `status` - Lifecycle status (started, mitigated, resolved, cancelled)
- `kind` - Incident type (normal, retrospective, test)
- `private` - Visibility flag
- **Timestamps** (all nullable, ISO8601):
  - `detected_at` - When issue was first detected
  - `acknowledged_at` - When incident commander acknowledged
  - `started_at` - When incident officially started
  - `mitigated_at` - When impact was mitigated
  - `resolved_at` - When fully resolved
  - `cancelled_at` - If cancelled
  - `created_at` - Created in Rootly
  - `updated_at` - Last updated
- `services` - Array of affected services
- `environments` - Array of affected environments
- `functionalities` - Array of affected functionalities
- `severity` - Severity object (SEV1, SEV2, etc.)

## Template Access Examples

### Alert Templates

```yaml
# Automatic action for alerts
on:
  alert.created:
    script: /opt/scripts/handle-alert.sh
    parameters:
      alert_id: "{{ id }}"
      summary: "{{ summary }}"
      status: "{{ status }}"
      source: "{{ source }}"
      severity: "{{ labels.severity }}"
      component: "{{ labels.component }}"
      host: "{{ data.host }}"
      latency_ms: "{{ data.latency_ms }}"
      monitor_id: "{{ data.monitor_id }}"
      service_name: "{{ services.0.name }}"
      service_slug: "{{ services.0.slug }}"
      environment: "{{ environments.0.slug }}"
      started_at: "{{ started_at }}"
      created_at: "{{ created_at }}"
      region: "{{ env.AWS_REGION }}"
      api_key: "{{ env.DATADOG_API_KEY }}"
```

### Alert Action Templates

```yaml
# Callable action for alerts
callable:
  restart_service:
    name: Restart Service
    trigger: alert.action_triggered
    script: /opt/scripts/restart.sh
    parameter_definitions:
      - name: service_name
        type: string
      - name: environment
        type: list
        options: [dev, staging, production]
      - name: force_restart
        type: boolean
    parameters:
      # User inputs (auto-mapped from parameter_definitions above)
      service_name: "{{ parameters.service_name }}"
      environment: "{{ parameters.environment }}"
      force_restart: "{{ parameters.force_restart }}"
      # Context (additional non-UI params)
      entity_id: "{{ entity_id }}"
      triggered_by: "{{ triggered_by.email }}"
      region: us-west-2
```

### Standalone Action Templates

```yaml
# Standalone callable action
callable:
  clear_cache:
    name: Clear Cache
    # trigger defaults to: action.triggered
    script: /opt/scripts/clear-cache.sh
    parameter_definitions:
      - name: cache_type
        type: list
        options: [redis, memcached]
      - name: scope
        type: list
        options: [global, regional, local]
    parameters:
      # User inputs (auto-mapped)
      cache_type: "{{ parameters.cache_type }}"
      scope: "{{ parameters.scope }}"
      # Context
      triggered_by: "{{ triggered_by.email }}"
```

### Incident Templates

```yaml
# Automatic action for incidents
on:
  incident.created:
    script: /opt/scripts/incident-response.sh
    parameters:
      incident_id: "{{ id }}"
      incident_number: "{{ sequential_id }}"
      title: "{{ title }}"
      summary: "{{ summary }}"
      status: "{{ status }}"
      severity_name: "{{ severity.name }}"
      detected_at: "{{ detected_at }}"
      started_at: "{{ started_at }}"
      services: "{{ services.0.name }}"
      environments: "{{ environments.0.slug }}"
```

### HTTP Action Templates

HTTP actions send webhooks/API calls using **Liquid template syntax** (same as script actions).

**Important:** HTTP actions use Liquid templates, so use `{{ field }}` syntax (no dot prefix).

#### Simple HTTP POST

```yaml
# Automatic PagerDuty integration
on:
  alert.created:
    http:
      url: "https://api.pagerduty.com/incidents"
      method: POST
      headers:
        Authorization: "Token token={{ env.PAGERDUTY_TOKEN }}"
        Content-Type: application/json
      body: |
        {
          "incident": {
            "type": "incident",
            "title": "{{ summary }}",
            "service": {
              "id": "{{ env.PAGERDUTY_SERVICE_ID }}",
              "type": "service_reference"
            },
            "urgency": "high",
            "body": {
              "type": "incident_body",
              "details": "Alert from {{ source }}: {{ summary }}"
            }
          }
        }
```

#### HTTP with Query Parameters

```yaml
on:
  incident.created:
    http:
      url: "https://api.example.com/v1/incidents"
      method: POST
      headers:
        X-API-Key: "{{ env.API_KEY }}"
      params:
        severity: "{{ severity.slug }}"
        environment: "{{ environments.0.slug }}"
      body: |
        {
          "title": "{{ title }}",
          "status": "{{ status }}"
        }
```

#### Slack Webhook (Complex JSON)

```yaml
on:
  alert.created:
    http:
      url: "{{ env.SLACK_WEBHOOK_URL }}"
      method: POST
      headers:
        Content-Type: application/json
      body: |
        {
          "text": ":fire: Alert: {{ summary }}",
          "blocks": [
            {
              "type": "section",
              "fields": [
                {"type": "mrkdwn", "text": "*Severity:* {{ labels.severity }}"},
                {"type": "mrkdwn", "text": "*Host:* {{ data.host }}"},
                {"type": "mrkdwn", "text": "*Service:* {{ services.0.name }}"},
                {"type": "mrkdwn", "text": "*Environment:* {{ environments.0.name }}"}
              ]
            },
            {
              "type": "section",
              "text": {
                "type": "mrkdwn",
                "text": "Alert started at {{ started_at }}"
              }
            }
          ]
        }
```

#### HTTP Action for alert.action_triggered

```yaml
callable:
  restart_service:
    name: Restart Service
    trigger: alert.action_triggered
    parameter_definitions:
      - name: service_name
        type: string
      - name: environment
        type: list
        options: [dev, staging, production]
    http:
      url: "https://api.example.com/services/restart"
      method: POST
      headers:
        Authorization: "Bearer {{ env.API_TOKEN }}"
        Content-Type: application/json
        X-Triggered-By: "{{ triggered_by.email }}"
      body: |
        {
          "service_name": "{{ parameters.service_name }}",
          "environment": "{{ parameters.environment }}",
          "entity": {
            "type": "alert",
            "id": "{{ entity_id }}"
          },
          "triggered_by": "{{ triggered_by.email }}"
        }
```

**Note:** HTTP actions return the HTTP status code as the exit code:
- 200-299 = success (exit code = status code)
- 400-599 = failure (exit code = status code)


## Quick Reference: All Serializer Fields

Based on actual Rails serializers in `app/serializers/edge_connectors/`:

### AlertSerializer
```
id, source, summary, status, labels, data,
started_at, ended_at, created_at, updated_at,
services[], environments[]
```

### IncidentSerializer
```
id, sequential_id, title, slug, summary, status, kind, private,
detected_at, acknowledged_at, started_at, mitigated_at, resolved_at, cancelled_at,
created_at, updated_at,
services[], environments[], functionalities[], severity{}
```

### ActionSerializer (action object, top-level)
```
id, name, slug
```

### ActionRunSerializer (data object for action_triggered events)
```
parameters{}, triggered_by{},
entity_id (for alert/incident actions only)
```

### Nested Object Serializers

**ServiceSerializer:**
```
id, name, slug
```

**EnvironmentSerializer:**
```
id, name, slug, color
```

**FunctionalitySerializer:**
```
id, name, slug
```

**SeveritySerializer:**
```
id, name, slug, color
```

**UserSerializer (triggered_by):**
```
id, email, name (from full_name)
```

## Benefits of Flat Structure

1. **Simple access**: `{{ action_name }}` instead of `{{ data.action_name }}`
2. **No confusion**: Clear what's event metadata vs payload data
3. **Consistent**: Works the same for all event types
4. **Easier debugging**: Less nesting to trace through
5. **Smaller payloads**: No redundant event metadata duplication
6. **Better template readability**: Shorter, clearer template expressions

## Backend Changes Needed

**For action_triggered events**, add action metadata at top level:

```ruby
# GOOD - action metadata separate from data
{
  action: EdgeConnectors::ActionSerializer.new(action_run.action),
  data: {
    entity_id: action_run.alert_id || action_run.incident_id,
    parameters: action_run.parameters,
    triggered_by: EdgeConnectors::UserSerializer.new(action_run.user)
  }
}
```

**ActionSerializer should include:**
```ruby
class EdgeConnectors::ActionSerializer < ActiveModel::Serializer
  attributes :id, :name, :slug
end
```

The event metadata (`id`, `event_id`, `event_type`, `timestamp`) already exists at the top level, so don't duplicate it.

**Benefits:**
- ✅ Clear separation: action metadata vs event data
- ✅ Access to both display name and machine identifier
- ✅ Consistent with top-level structure (id, event_id, action, data)
- ✅ No confusion between parameters and action identifier

---

Once you update the API, the connector will have access to both `{{ action.name }}` (display name) and `{{ action.slug }}` (identifier)! 🎯