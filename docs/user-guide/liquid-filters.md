# Liquid Template Filters Reference

Complete reference for all filters available in the Edge Connector templates.

**Source:** https://github.com/osteele/liquid (Shopify Liquid implementation in Go)

## Quick Reference

| Category | Filters |
|----------|---------|
| **Array** | compact, concat, first, join, last, map, reverse, sort, sort_natural, uniq |
| **String** | append, capitalize, downcase, escape, escape_once, lstrip, newline_to_br, prepend, remove, remove_first, replace, replace_first, rstrip, slice, split, strip, strip_html, strip_newlines, truncate, truncatewords, upcase, url_decode, url_encode |
| **Number** | abs, ceil, divided_by, floor, minus, modulo, plus, round, times |
| **Utility** | default, json, inspect, type |
| **Date** | date |

---

## Array Filters

### compact
Removes nil values from array.
```yaml
{{ items | compact }}
# Input: [1, nil, 2, nil, 3]
# Output: [1, 2, 3]
```

### concat
Combines two arrays.
```yaml
{{ array1 | concat: array2 }}
# Input: [1, 2] + [3, 4]
# Output: [1, 2, 3, 4]
```

### first
Returns first element of array.
```yaml
{{ services | first }}
{{ services.first.name }}  # Shorthand
# Input: [{name: "DB"}, {name: "API"}]
# Output: {name: "DB"}
```

### join
Joins array elements with separator.
```yaml
{{ items | join: ", " }}
# Input: ["Database", "API", "Cache"]
# Output: "Database, API, Cache"
```

### last
Returns last element of array.
```yaml
{{ services | last }}
{{ services.last.name }}  # Shorthand
# Input: [{name: "DB"}, {name: "API"}]
# Output: {name: "API"}
```

### map
Extracts property from each object in array.
```yaml
{{ services | map: "name" }}
# Input: [{name: "DB", id: 1}, {name: "API", id: 2}]
# Output: ["DB", "API"]

# Often combined with join:
{{ services | map: "name" | join: ", " }}
# Output: "DB, API"
```

### reverse
Reverses array order.
```yaml
{{ items | reverse }}
# Input: [1, 2, 3]
# Output: [3, 2, 1]
```

### sort
Sorts array alphabetically.
```yaml
{{ items | sort }}
# Input: ["charlie", "alice", "bob"]
# Output: ["alice", "bob", "charlie"]

# Sort by property:
{{ services | sort: "name" }}
# Sorts objects by their 'name' property
```

### sort_natural
Sorts with natural ordering (numbers).
```yaml
{{ items | sort_natural }}
# Input: ["item-10", "item-2", "item-1"]
# Output: ["item-1", "item-2", "item-10"]
```

### uniq
Removes duplicate values.
```yaml
{{ items | uniq }}
# Input: ["a", "b", "a", "c", "b"]
# Output: ["a", "b", "c"]
```

---

## String Filters

### append
Concatenates string to end.
```yaml
{{ filename | append: ".log" }}
# Input: "app"
# Output: "app.log"
```

### capitalize
Uppercases first character.
```yaml
{{ status | capitalize }}
# Input: "open"
# Output: "Open"
```

### downcase
Converts to lowercase.
```yaml
{{ status | downcase }}
# Input: "CRITICAL"
# Output: "critical"
```

### escape
HTML-encodes string.
```yaml
{{ text | escape }}
# Input: "<script>alert('xss')</script>"
# Output: "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;"
```

### escape_once
Escapes HTML without double-encoding.
```yaml
{{ text | escape_once }}
```

### lstrip
Removes leading whitespace.
```yaml
{{ text | lstrip }}
# Input: "   hello"
# Output: "hello"
```

### newline_to_br
Converts newlines to `<br>` tags.
```yaml
{{ text | newline_to_br }}
# Input: "line1\nline2"
# Output: "line1<br>line2"
```

### prepend
Concatenates string to beginning.
```yaml
{{ message | prepend: "[ALERT] " }}
# Input: "High latency"
# Output: "[ALERT] High latency"
```

### remove
Removes all occurrences of substring.
```yaml
{{ text | remove: "error: " }}
# Input: "error: Connection failed"
# Output: "Connection failed"
```

### remove_first
Removes first occurrence only.
```yaml
{{ text | remove_first: "test" }}
```

### replace
Replaces all occurrences.
```yaml
{{ text | replace: " ", "_" }}
# Input: "hello world"
# Output: "hello_world"
```

### replace_first
Replaces first occurrence only.
```yaml
{{ text | replace_first: "error", "warning" }}
```

### rstrip
Removes trailing whitespace.
```yaml
{{ text | rstrip }}
# Input: "hello   "
# Output: "hello"
```

### slice
Extracts substring.
```yaml
{{ text | slice: 0, 5 }}
# Input: "hello world"
# Output: "hello"
```

### split
Splits string into array.
```yaml
{{ text | split: "," }}
# Input: "a,b,c"
# Output: ["a", "b", "c"]
```

### strip
Removes leading and trailing whitespace.
```yaml
{{ text | strip }}
# Input: "  hello  "
# Output: "hello"
```

### strip_html
Removes HTML tags.
```yaml
{{ text | strip_html }}
# Input: "<b>Bold</b> text"
# Output: "Bold text"
```

### strip_newlines
Removes newline characters.
```yaml
{{ text | strip_newlines }}
```

### truncate
Shortens string with ellipsis.
```yaml
{{ summary | truncate: 50 }}
# Input: "This is a very long summary that needs to be truncated"
# Output: "This is a very long summary that needs to be tr..."
```

### truncatewords
Truncates by word count.
```yaml
{{ summary | truncatewords: 5 }}
# Input: "This is a very long summary"
# Output: "This is a very long..."
```

### upcase
Converts to uppercase.
```yaml
{{ status | upcase }}
# Input: "open"
# Output: "OPEN"
```

### url_decode
Decodes URL-encoded string.
```yaml
{{ url | url_decode }}
# Input: "hello%20world"
# Output: "hello world"
```

### url_encode
URL-encodes string.
```yaml
{{ text | url_encode }}
# Input: "hello world"
# Output: "hello%20world"
```

---

## Number Filters

### abs
Absolute value.
```yaml
{{ num | abs }}
# Input: -5
# Output: 5
```

### ceil
Rounds up to integer.
```yaml
{{ num | ceil }}
# Input: 4.2
# Output: 5
```

### divided_by
Division.
```yaml
{{ num | divided_by: 10 }}
# Input: 50
# Output: 5
```

### floor
Rounds down to integer.
```yaml
{{ num | floor }}
# Input: 4.8
# Output: 4
```

### minus
Subtraction.
```yaml
{{ num | minus: 5 }}
# Input: 10
# Output: 5
```

### modulo
Remainder (modulo operation).
```yaml
{{ num | modulo: 3 }}
# Input: 10
# Output: 1
```

### plus
Addition.
```yaml
{{ num | plus: 10 }}
# Input: 5
# Output: 15
```

### round
Rounds to decimal places.
```yaml
{{ num | round: 2 }}
# Input: 3.14159
# Output: 3.14
```

### times
Multiplication.
```yaml
{{ num | times: 2 }}
# Input: 5
# Output: 10
```

---

## Utility Filters

### default
Returns default if value is nil/false/empty.
```yaml
{{ missing | default: "N/A" }}
{{ optional_field | default: "default_value" }}
```

### json
Converts to JSON string.
```yaml
{{ data | json }}
# Input: {host: "prod-01", port: 3000}
# Output: '{"host":"prod-01","port":3000}'
```

### inspect
Debug representation (JSON-like).
```yaml
{{ variable | inspect }}
# Useful for debugging template data
```

### type
Returns variable type.
```yaml
{{ variable | type }}
# Returns: "string", "array", "map", etc.
```

---

## Date Filters

### date
Formats timestamp using strftime format.

```yaml
{{ started_at | date: "%Y-%m-%d" }}          # 2025-10-26
{{ created_at | date: "%H:%M:%S" }}          # 21:30:00
{{ timestamp | date: "%B %d, %Y" }}          # October 26, 2025
{{ time | date: "%Y-%m-%d %H:%M:%S %Z" }}    # 2025-10-26 21:30:00 UTC
```

**Common format codes:**
- `%Y` - Year (2025)
- `%m` - Month (01-12)
- `%d` - Day (01-31)
- `%H` - Hour 24h (00-23)
- `%M` - Minute (00-59)
- `%S` - Second (00-59)
- `%B` - Month name (October)
- `%A` - Day name (Monday)

---

## Combining Filters

Multiple filters can be chained:

```yaml
# Extract, sort, join
{{ services | map: "name" | sort | join: " | " }}
# Result: "API | Cache | Database"

# Clean and format
{{ summary | strip | truncate: 100 | capitalize }}

# Convert and encode
{{ data | json | url_encode }}

# Default and transform
{{ status | default: "unknown" | upcase }}
```

---

## Edge Connector Specific Examples

### Alert Templates
```yaml
# Alert message for Slack
text: "[{{ labels.severity | upcase }}] {{ source | capitalize }}: {{ summary }}"
# Result: "[CRITICAL] Datadog: High database latency"

# Service list
affected: "{{ services | map: 'name' | join: ', ' | default: 'None' }}"
# Result: "Database, API Gateway" or "None"
```

### Incident Templates
```yaml
# Incident title
title: "[{{ severity.name }}] {{ title | truncate: 100 }}"
# Result: "[SEV1] Production API Gateway Outage..."

# Functionality list
impact: "Affected: {{ functionalities | map: 'name' | join: ', ' }}"
# Result: "Affected: API Requests, Database Access"
```

### Action Templates
```yaml
# User-friendly message
message: "{{ triggered_by.name | default: 'Unknown' }} restarted {{ parameters.service_name | upcase }}"
# Result: "Quentin Rousseau restarted API-GATEWAY"

# Environment safety
env_check: "{{ parameters.environment | default: 'development' | downcase }}"
# Ensures environment is lowercase
```

---

## Best Practices

1. **Always use `default`** for optional fields
2. **Use `map` + `join`** instead of loops
3. **Sanitize user input** with `escape` for HTML contexts
4. **Use `truncate`** for long text in notifications
5. **Use `upcase`/`downcase`** for consistent formatting
6. **Chain filters** for complex transformations

---

## Testing Templates

Test templates locally:
```bash
# Set environment variable
export TEST_VAR="test_value"

# In actions.yml
parameters:
  test: "{{ env.TEST_VAR | default: 'fallback' }}"
```

Check logs for rendered values:
```bash
REC_LOG_LEVEL=debug ./bin/rootly-edge-connector
# Look for: "Executing script" log with parameter values
```

---

## More Information

- **Liquid documentation**: https://shopify.github.io/liquid/
- **Go Liquid library**: https://github.com/osteele/liquid
- **Standard filters source**: https://github.com/osteele/liquid/blob/main/filters/standard_filters.go
- **Sort filters source**: https://github.com/osteele/liquid/blob/main/filters/sort_filters.go
