# API Payload Examples

Real-world examples of event payloads from `GET /rec/v1/deliveries`.

## Event Type Categories

**Automatic triggers** - Configured in `on:` section (future format):
- `alert.created`, `alert.updated`
- `incident.created`, `incident.updated`

**Callable actions** - Configured in `callable:` section (future format):
- `alert.action_triggered` - Actions on alerts
- `incident.action_triggered` - Actions on incidents
- `action.triggered` - Standalone actions

---

## alert.created - Production Database Alert

```json
{
  "id": "delivery-abc-123",
  "event_id": "event-def-456",
  "event_type": "alert.created",
  "timestamp": "2025-10-26T21:30:00Z",
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
      "query_count": 342
    },
    "started_at": "2025-10-26T21:29:45Z",
    "ended_at": null,
    "created_at": "2025-10-26T21:29:50Z",
    "updated_at": "2025-10-26T21:29:50Z",
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

### Template Usage

```yaml
parameters:
  alert_id: "{{ id }}"
  alert_summary: "{{ summary }}"
  severity: "{{ labels.severity }}"
  host: "{{ data.host }}"
  latency: "{{ data.latency_ms }}"
  service_name: "{{ services.0.name }}"      # First service
  environment: "{{ environments.0.slug }}"   # First environment
```

## alert.created - PagerDuty Integration

```json
{
  "id": "delivery-xyz-789",
  "event_id": "event-ghi-012",
  "event_type": "alert.created",
  "timestamp": "2025-10-26T21:35:00Z",
  "data": {
    "id": "alert-pagerduty-123",
    "source": "pagerduty",
    "summary": "API service is down",
    "status": "open",
    "labels": {
      "severity": "high",
      "urgency": "high",
      "impact": "critical"
    },
    "data": {
      "incident_key": "PD-12345",
      "incident_url": "https://example.pagerduty.com/incidents/12345",
      "triggered_by": "monitoring_service",
      "escalation_policy": "Engineering On-Call"
    },
    "started_at": "2025-10-26T21:34:30Z",
    "ended_at": null,
    "created_at": "2025-10-26T21:34:35Z",
    "updated_at": "2025-10-26T21:34:35Z",
    "services": [
      {
        "id": "service-api-uuid",
        "name": "API Gateway",
        "slug": "api-gateway"
      },
      {
        "id": "service-auth-uuid",
        "name": "Authentication Service",
        "slug": "auth-service"
      }
    ],
    "environments": [
      {
        "id": "env-prod-uuid",
        "name": "Production",
        "slug": "production",
        "color": "#E74C3C"
      }
    ]
  }
}
```

### Template Usage

```yaml
parameters:
  pagerduty_key: "{{ data.incident_key }}"
  pagerduty_url: "{{ data.incident_url }}"
  urgency: "{{ labels.urgency }}"
  all_services: "{{ services | join:', ' }}"  # "API Gateway, Authentication Service"
```

## alert.updated - Status Change

```json
{
  "id": "delivery-update-456",
  "event_id": "event-update-789",
  "event_type": "alert.updated",
  "timestamp": "2025-10-26T22:00:00Z",
  "data": {
    "id": "6aeb35ae-ca31-4bcf-91bd-c4ecce44dedc",
    "source": "datadog",
    "summary": "High database latency detected",
    "status": "resolved",
    "labels": {
      "severity": "critical",
      "component": "database"
    },
    "data": {
      "host": "prod-db-01.example.com",
      "latency_ms": 150,
      "resolution": "auto-scaled database pool"
    },
    "started_at": "2025-10-26T21:29:45Z",
    "ended_at": "2025-10-26T21:59:30Z",
    "created_at": "2025-10-26T21:29:50Z",
    "updated_at": "2025-10-26T22:00:00Z",
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

## incident.created - Full Example

Based on `EdgeConnectors::IncidentSerializer`:

```json
{
  "id": "delivery-inc-001",
  "event_id": "event-inc-002",
  "event_type": "incident.created",
  "timestamp": "2025-10-26T23:00:00Z",
  "data": {
    "id": "incident-uuid-123",
    "sequential_id": 42,
    "title": "Production API Gateway Outage",
    "slug": "production-api-gateway-outage",
    "summary": "Complete outage affecting all customers",
    "status": "started",
    "kind": "normal",
    "private": false,
    "detected_at": "2025-10-26T22:58:00Z",
    "acknowledged_at": null,
    "started_at": "2025-10-26T22:58:00Z",
    "mitigated_at": null,
    "resolved_at": null,
    "cancelled_at": null,
    "created_at": "2025-10-26T22:59:00Z",
    "updated_at": "2025-10-26T23:00:00Z",
    "services": [
      {
        "id": "service-api-uuid",
        "name": "API Gateway",
        "slug": "api-gateway"
      }
    ],
    "environments": [
      {
        "id": "env-prod-uuid",
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

## alert.action_triggered - Callable Action on Alert

User clicks action button in Rootly UI on an alert.

```json
{
  "id": "delivery-alert-action-123",
  "event_id": "event-alert-action-456",
  "event_type": "alert.action_triggered",
  "timestamp": "2025-10-26T23:15:00Z",
  "action": {
    "id": "01939a0e-7c4a-7e4e-b5e1-5c3f8e2c9a1b",
    "name": "Restart Test Service",
    "slug": "restart_test_service"
  },
  "data": {
    "entity_id": "alert-uuid-999",
    "parameters": {
      "service_name": "api-gateway",
      "environment": "production",
      "force_restart": true,
      "drain_timeout": 30
    },
    "triggered_by": {
      "id": 50,
      "name": "Quentin Rousseau",
      "email": "quentin@rootly.com"
    }
  }
}
```

### Template Usage in Actions

```yaml
# Callable action for alerts
callable:
  restart_test_service:
    name: Restart Test Service
    trigger: alert.action_triggered
    script: /opt/scripts/restart-service.sh
    parameter_definitions:
      - name: service_name
        type: string
      - name: environment
        type: list
        options: [dev, staging, production]
      - name: force_restart
        type: boolean
    parameters:
      # User inputs (auto-mapped from parameter_definitions)
      service_name: "{{ parameters.service_name }}"
      environment: "{{ parameters.environment }}"
      force: "{{ parameters.force_restart }}"
      # Context data (additional non-UI params)
      entity_id: "{{ entity_id }}"
      triggered_by: "{{ triggered_by.email }}"
      region: us-west-2
```

## incident.action_triggered - Callable Action on Incident

User clicks action button in Rootly UI on an incident.

```json
{
  "id": "delivery-inc-action-1",
  "event_id": "event-inc-action-2",
  "event_type": "incident.action_triggered",
  "timestamp": "2025-10-26T23:20:00Z",
  "action": {
    "id": "01939b2f-8d5c-7f2e-a3c1-4d5e6f7a8b9c",
    "name": "Scale Infrastructure",
    "slug": "scale_infrastructure"
  },
  "data": {
    "entity_id": "incident-uuid-456",
    "parameters": {
      "target_capacity": 200,
      "scaling_policy": "aggressive"
    },
    "triggered_by": {
      "id": 42,
      "name": "Sarah Johnson",
      "email": "sarah@example.com"
    }
  }
}
```

## action.triggered - Standalone Callable Action

User triggers a standalone action (not tied to any alert or incident).

```json
{
  "id": "delivery-standalone-1",
  "event_id": "event-standalone-2",
  "event_type": "action.triggered",
  "timestamp": "2025-10-26T23:25:00Z",
  "action": {
    "id": "01939c4d-9e6f-7a3b-c2d1-8e9f0a1b2c3d",
    "name": "Clear Global Cache",
    "slug": "clear_global_cache"
  },
  "data": {
    "parameters": {
      "cache_type": "redis",
      "scope": "global"
    },
    "triggered_by": {
      "id": 50,
      "name": "Quentin Rousseau",
      "email": "quentin@rootly.com"
    }
    // No entity_id - this is a standalone action
  }
}
```

## Edge Cases

### Alert with No Services

```json
{
  "event_type": "alert.created",
  "data": {
    "id": "alert-no-svc",
    "summary": "Orphaned alert",
    "status": "open",
    "services": [],        // ← Empty array
    "environments": []     // ← Empty array
  }
}
```

### Alert with Custom Data (Datadog)

```json
{
  "event_type": "alert.created",
  "data": {
    "id": "alert-datadog-1",
    "source": "datadog",
    "summary": "CPU usage above 90%",
    "status": "open",
    "data": {
      "tags": ["env:production", "service:api", "host:prod-01"],
      "metric": "system.cpu.usage",
      "value": 94.2,
      "threshold": 90.0,
      "monitor_id": "12345678",
      "monitor_name": "High CPU Usage"
    }
  }
}
```

### Standalone Action with No Parameters

```json
{
  "event_type": "action.triggered",
  "action": {
    "id": "01939d5e-af7g-8h4c-d3e2-9f0a1b2c3d4e",
    "name": "Clear Cache",
    "slug": "clear_cache"
  },
  "data": {
    "parameters": {},      // ← No user inputs required
    "triggered_by": {
      "id": 50,
      "name": "Quentin Rousseau",
      "email": "quentin@rootly.com"
    }
    // No entity_id - standalone action
  }
}
```

## HTTP Action Examples

### alert.created → Slack Notification

```yaml
# Automatic action for alerts
on:
  alert.created:
    http:
      url: "{{ env.SLACK_WEBHOOK_URL }}"
      method: POST
      headers:
        Content-Type: application/json
      body: |
        {
          "text": ":warning: New Alert",
          "attachments": [{
            "color": "danger",
            "fields": [
            {"title": "Summary", "value": "{{ summary }}", "short": false},
            {"title": "Source", "value": "{{ source }}", "short": true},
            {"title": "Severity", "value": "{{ labels.severity }}", "short": true},
            {"title": "Host", "value": "{{ data.host }}", "short": true},
            {"title": "Environment", "value": "{{ environments.0.name }}", "short": true}
          ]
        }]
      }
  timeout: 10
```

### incident.created → PagerDuty Integration

```yaml
# Automatic action for incidents
on:
  incident.created:
    http:
      url: "https://api.pagerduty.com/incidents"
      method: POST
      headers:
        Authorization: "Token token={{ env.PAGERDUTY_TOKEN }}"
        Content-Type: application/json
        From: "{{ env.PAGERDUTY_FROM_EMAIL }}"
      body: |
        {
          "incident": {
            "type": "incident",
          "title": "[{{ severity.name }}] {{ title }}",
          "service": {
            "id": "{{ env.PAGERDUTY_SERVICE_ID }}",
            "type": "service_reference"
          },
          "urgency": "high",
          "body": {
            "type": "incident_body",
            "details": "{{ summary }}\n\nAffected services: {{ services.0.name }}"
          }
        }
      }
  timeout: 15
```

### alert.action_triggered → Restart Service API

```yaml
callable:
  restart_service_api:
    name: Restart Service
    trigger: alert.action_triggered
    parameter_definitions:
      - name: service_name
        type: string
        required: true
      - name: force_restart
        type: boolean
        default: false
    http:
      url: "https://api.example.com/v1/services/{{ parameters.service_name }}/restart"
      method: POST
    headers:
      Authorization: "Bearer {{ env.API_TOKEN }}"
      Content-Type: "application/json"
      X-Triggered-By: "{{ triggered_by.email }}"
    body: |
      {
        "force": {{ parameters.force_restart }},
        "reason": "Manual restart via Rootly",
        "alert_id": "{{ entity_id }}"
      }
  timeout: 60
```

### action.triggered → Clear Global Cache

```yaml
callable:
  clear_cache:
    name: Clear Cache
    # trigger defaults to: action.triggered (standalone)
    parameter_definitions:
      - name: cache_type
        type: list
        options: [redis, memcached, all]
        required: true
    http:
      url: "https://cache-api.example.com/v1/clear"
      method: POST
      headers:
        X-API-Key: "{{ env.CACHE_API_KEY }}"
    params:
      type: "{{ parameters.cache_type }}"
    body: |
      {
        "triggered_by": "{{ triggered_by.email }}",
        "scope": "global"
      }
  timeout: 30
```

**HTTP Action Behavior:**
- Exit code = HTTP status code (200, 404, 500, etc.)
- Stdout = Response body + status message
- Stderr = Error message (if request fails)
- Success = 2xx status codes
- Failure = 4xx, 5xx status codes

## Template Access Patterns

### Simple Fields
```yaml
alert_id: "{{ id }}"
status: "{{ status }}"
summary: "{{ summary }}"
```

### Nested Objects
```yaml
severity: "{{ labels.severity }}"
host: "{{ data.host }}"
metric_value: "{{ data.value }}"
```

### Arrays (First Element)
```yaml
service_name: "{{ services.0.name }}"
service_slug: "{{ services.0.slug }}"
environment: "{{ environments.0.slug }}"
```

### Environment Variables
```yaml
api_key: "{{ env.DATADOG_API_KEY }}"
region: "{{ env.AWS_REGION }}"
```

### Mixed
```yaml
message: "[{{ labels.severity }}] {{ summary }} on {{ data.host }} in {{ environments.0.name }}"
# Result: "[critical] High database latency detected on prod-db-01.example.com in Production"
```

## Testing Locally

Create a test event payload file:

```bash
# test-alert.json
{
  "events": [{
    "id": "test-delivery-1",
    "event_id": "test-event-1",
    "event_type": "alert.created",
    "timestamp": "2025-10-26T23:00:00Z",
    "data": {
      "id": "test-alert-123",
      "source": "test",
      "summary": "Test alert for local development",
      "status": "open",
      "labels": {"severity": "critical"},
      "data": {"host": "localhost"},
      "services": [{"id": "svc-1", "name": "Test Service", "slug": "test"}],
      "environments": [{"id": "env-1", "name": "Development", "slug": "dev"}]
    }
  }]
}
```

Then post to your local mock server to trigger actions.
