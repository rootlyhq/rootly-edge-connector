# Template Syntax - Quick Start Guide

The Edge Connector uses **Liquid templates** for dynamic value substitution.

**📖 Complete filter reference:** [liquid-filters.md](liquid-filters.md)
**Library:** https://github.com/osteele/liquid

## Basic Syntax

### Simple Fields
```yaml
{{ id }}              # Alert/Incident ID
{{ summary }}         # Summary text
{{ status }}          # Status
```

### Nested Fields
```yaml
{{ labels.severity }}        # labels.severity
{{ data.host }}              # data.host (custom monitoring data)
{{ severity.name }}          # severity.name (incident severity object)
```

### Array Access
```yaml
{{ services[0].name }}       # First service name
{{ services.first.name }}    # First service (helper)
{{ services.last.slug }}     # Last service slug
{{ environments[0].slug }}   # First environment
```

### Environment Variables
```yaml
{{ env.API_KEY }}            # OS environment variable
{{ env.AWS_REGION }}         # Region from env
```

## Filters

Filters transform values using the pipe `|` operator.

**📖 See [liquid-filters.md](liquid-filters.md) for complete filter reference (50+ filters)**

### Most Common Filters

```yaml
# Array filters
{{ services | map: "name" | join: ", " }}     # Extract and join
{{ services.first.name }}                      # First element (shorthand)
{{ environments | sort: "slug" }}              # Sort by property

# String filters
{{ status | upcase }}                          # "OPEN"
{{ summary | truncate: 50 }}                   # Shorten text
{{ name | default: "N/A" }}                    # Fallback value

# Number filters
{{ count | plus: 1 }}                          # Math operations

# Date filters
{{ started_at | date: "%Y-%m-%d %H:%M" }}     # Format timestamp
```

## Real-World Examples

### Alert with Service Info
```yaml
# Template
message: "[{{ labels.severity | upcase }}] {{ summary }} on {{ services.first.name }} ({{ environments[0].slug }})"

# Result
message: "[CRITICAL] High database latency on Production Database (production)"
```

### List All Services
```yaml
# Template
services_list: "{{ services | map: 'name' | join: ', ' }}"

# Result
services_list: "Database, API Gateway, Cache Service"
```

### HTTP POST Body
```yaml
http:
  body: |
    {
      "alert_id": "{{ id }}",
      "summary": "{{ summary }}",
      "severity": "{{ labels.severity | upcase }}",
      "affected_services": "{{ services | map: 'name' | join: ', ' }}",
      "environment": "{{ environments.first.slug }}",
      "started_at": "{{ started_at | date: '%Y-%m-%d %H:%M:%S' }}"
    }
```

### Conditional Text (Action)
```yaml
# Template
reason: "{{ parameters.reason | default: 'Manual action triggered' }}"

# If parameters.reason is empty, uses default
reason: "Manual action triggered"
```

## Advanced Features

### Chaining Filters
```yaml
# Multiple filters in sequence
{{ services | map: "name" | sort | join: " | " | upcase }}
# Result: "API | CACHE | DATABASE"
```

### Nested Array Access
```yaml
{{ services[0].tags[0] }}          # Access nested arrays
{{ data.metrics.values[5] }}       # Deep nesting with arrays
```

### Safe Navigation
Liquid handles nil gracefully - missing values return empty string:
```yaml
{{ missing.field }}                # Returns ""
{{ array[999].name }}              # Returns "" (out of bounds)
{{ undefined | default: "N/A" }}   # Returns "N/A"
```

## Common Patterns

### Alert Notification
```yaml
parameters:
  title: "[{{ labels.severity | upcase }}] {{ summary }}"
  host: "{{ data.host | default: 'unknown' }}"
  service: "{{ services.first.name }}"
  environment: "{{ environments.first.slug }}"
```

### Incident Message
```yaml
parameters:
  message: "SEV{{ severity.name | remove: 'SEV' }}: {{ title }}"
  services: "{{ services | map: 'name' | join: ', ' }}"
  functionalities: "{{ functionalities | map: 'name' | join: ', ' }}"
```

### Action Context
```yaml
parameters:
  service: "{{ parameters.service_name }}"
  environment: "{{ parameters.environment | default: 'production' }}"
  entity: "{{ entity_id }}"
  user: "{{ triggered_by.email }}"
```

## Backward Compatibility

Old syntax still works:
```yaml
{{ field }}          # Still works
{{ nested.field }}   # Still works
{{ env.VAR }}        # Still works
{{ event.field }}    # Still works (event.* prefix)
```

New syntax (Liquid):
```yaml
{{ array[0].field }}           # NEW: Array index
{{ array.first.field }}        # NEW: Array helper
{{ field | filter }}           # NEW: Filters
{{ field | filter: "arg" }}    # NEW: Filter with args
```

## Limitations

- **No custom Liquid tags**: Only {{ }} output tags, no {% %} logic tags
- **No loops**: {% for %} not supported (use filters instead)
- **No conditionals**: {% if %} not supported (use default filter instead)

This keeps templates simple and safe for security.

## Tips

1. **Use `default` filter** for optional fields:
   ```yaml
   {{ data.host | default: "localhost" }}
   ```

2. **Use `map` + `join`** for arrays:
   ```yaml
   {{ services | map: "name" | join: ", " }}
   ```

3. **Test templates** with mock events before deploying

4. **Keep it simple** - Complex logic belongs in scripts, not templates
