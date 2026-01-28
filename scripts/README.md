# Test Scripts for Local Development

This directory contains simple self-contained test scripts for local development and testing of the Rootly Edge Connector.

## Available Test Scripts

### `test-echo.sh`
**Purpose**: Basic parameter echo and validation

Prints all received parameters and environment variables to stdout. Use this to verify that parameters are being correctly passed from the connector to scripts.

**What it does**:
- Displays all `REC_PARAM_*` environment variables
- Shows timestamp and hostname
- Always exits successfully (exit 0)

**Use case**: Verify parameter passing and template substitution

---

### `test-conditional.sh`
**Purpose**: Demonstrates conditional logic based on event data

Shows how to handle different scenarios based on event severity or other parameters.

**What it does**:
- Reads `REC_PARAM_SEVERITY` and `REC_PARAM_SUMMARY`
- Executes different logic for critical/high/medium/low severity
- Uses bash `case` statement for branching

**Use case**: Test conditional automation logic

---

### `test-failure.sh`
**Purpose**: Simulates a script failure

Always exits with code 1 and writes to stderr.

**What it does**:
- Prints error messages to stderr
- Exits with exit code 1
- Simulates a failed automation

**Use case**: Test error handling and result reporting

---

### `test-timeout.sh`
**Purpose**: Tests timeout handling

Runs longer than the configured timeout to verify that the connector properly kills long-running scripts.

**What it does**:
- Sleeps for specified duration (default 60 seconds)
- Prints progress every second
- Should be killed by timeout before completion

**Use case**: Test script timeout enforcement

**Configuration tip**: Set the action's `timeout` to be less than the script's sleep duration.

---

## Usage with Local Development

### 1. Copy configuration files:
```bash
cp config.example.dev.yml config.yml
cp actions.example.dev.yml actions.yml
```

### 2. Run the connector:
```bash
go run cmd/rec/main.go -config config.yml -actions actions.yml
```

### 3. The connector will:
- Poll the mock API (or real API if configured)
- Execute matching scripts based on event types
- Log output to stdout with colored formatting

## Parameter Passing

Scripts receive parameters as environment variables with the `REC_PARAM_` prefix:

| YAML Parameter | Environment Variable |
|----------------|---------------------|
| `alert_id: "123"` | `REC_PARAM_ALERT_ID=123` |
| `summary: "Database timeout"` | `REC_PARAM_SUMMARY=Database timeout` |
| `severity: "critical"` | `REC_PARAM_SEVERITY=critical` |
| `host: "db-01"` | `REC_PARAM_HOST=db-01` |

### Example in actions.yml:
```yaml
on:
  alert.created:
    script: /path/to/your-script.sh
    parameters:
      alert_id: "{{ id }}"
      summary: "{{ summary }}"
      severity: "{{ labels.severity }}"
      host: "{{ data.host }}"
      service: "{{ services[0].name }}"
```

### Accessing in script:
```bash
#!/bin/bash
ALERT_ID="$REC_PARAM_ALERT_ID"
SUMMARY="$REC_PARAM_SUMMARY"
SEVERITY="$REC_PARAM_SEVERITY"
HOST="$REC_PARAM_HOST"
SERVICE="$REC_PARAM_SERVICE"

echo "Processing alert: $SUMMARY"
echo "Alert ID: $ALERT_ID, Severity: $SEVERITY"
echo "Affected: $SERVICE on $HOST"
```

## Testing Without Mock Server

You can test scripts directly by setting environment variables:

```bash
# Test the echo script
REC_PARAM_ALERT_ID="test-123" \
REC_PARAM_SUMMARY="Database timeout detected" \
REC_PARAM_SEVERITY="critical" \
REC_PARAM_HOST="db-01" \
REC_PARAM_SERVICE="postgres" \
./scripts/test-echo.sh

# Test the conditional script
REC_PARAM_SEVERITY="critical" \
REC_PARAM_SUMMARY="High CPU usage" \
./scripts/test-conditional.sh

# Test the failure script
./scripts/test-failure.sh
echo "Exit code: $?"  # Should print 1
```

## Adding Your Own Test Scripts

1. Create a new script in this directory
2. Make it executable: `chmod +x scripts/your-script.sh`
3. Use `REC_PARAM_*` environment variables
4. Add it to `actions.yml` with a trigger
5. Test by running the connector

### Script Template:
```bash
#!/bin/bash
set -e  # Exit on error

# Read parameters
PARAM1="${REC_PARAM_PARAM1:-default}"
PARAM2="${REC_PARAM_PARAM2:-default}"

# Your logic here
echo "Processing with PARAM1=$PARAM1"

# Exit with status
exit 0  # or exit 1 for failure
```
