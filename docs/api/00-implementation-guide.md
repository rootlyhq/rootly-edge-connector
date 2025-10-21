# Backend API Implementation Notes

This document explains what the Rails backend needs to implement for the Edge Connector.

**Related documentation:**
- **[api-payload-format.md](api-payload-format.md)** - Complete event payload specification and format details
- **[api-examples.md](api-examples.md)** - Real-world payload examples for all event types

## POST /rec/v1/actions - Register/Sync Actions

**Purpose**: Replace all actions for this connector with the provided list (sync/upsert behavior).

**Request:**
```json
{
  "actions": [
    {
      "slug": "restart_server",
      "name": "Restart Production Server",
      "action_type": "script",
      "description": "Restarts the production server with graceful shutdown",
      "timeout": 300,
      "parameters": [
        {
          "name": "service_name",
          "type": "string",
          "required": true,
          "description": "Service to restart"
        }
      ]
    }
  ]
}
```

**Field mapping:**
- `slug` (REQUIRED): Machine identifier for matching (e.g., "restart_server")
- `name` (OPTIONAL): Human-readable display name (e.g., "Restart Production Server")
- `description` (OPTIONAL): Multi-line description for UI

**Backend implementation requirements:**

### 1. Identify the Connector
- Extract connector from auth token
- All actions belong to this specific edge connector instance

### 2. Sync Actions (Upsert + Delete)

```ruby
# Sync behavior implementation:
def sync_actions(connector, new_actions)
  # Get current action slugs for this connector
  current_slugs = connector.actions.source_connector.pluck(:slug)
  new_slugs = new_actions.map { |a| a[:slug] }

  # Upsert (create or update) - match by SLUG
  new_actions.each do |action_data|
    connector.actions.find_or_initialize_by(slug: action_data[:slug]) do |action|
      action.source = "connector"
    end.tap do |action|
      # IMPORTANT: If name is provided, use it and parameterize for slug validation
      # If name is empty, connector already sent valid slug
      if action_data[:name].present?
        # Verify name parameterizes to the provided slug
        expected_slug = action_data[:name].parameterize(separator: "_")
        if expected_slug != action_data[:slug]
          # Name doesn't match slug - this is OK, they're independent
          # Store name as-is for display
        end
        action.name = action_data[:name]
      else
        # No name provided - store slug as name (backend won't humanize)
        action.name = action_data[:slug]
      end

      action.action_type = action_data[:action_type]
      action.description = action_data[:description]
      action.timeout = action_data[:timeout]
      action.metadata = {
        parameters: action_data[:parameters],
        http: action_data[:http]  # For HTTP actions
      }.compact
      action.save!
    end
  end

  # Delete connector-sourced actions not in new list (cleanup stale actions)
  to_delete = current_slugs - new_slugs
  connector.actions.source_connector.where(slug: to_delete).destroy_all

  # Return counts
  {
    registered: new_actions.size,
    failed: 0,  # Or count validation failures
    actions: new_actions,
    failures: []
  }
end
```

### 3. Response Format

**Success (201 Created):**
```json
{
  "registered": 2,
  "failed": 0,
  "actions": [...],  // Echoes back the registered actions
  "failures": []
}
```

**Partial success (207 Multi-Status):**
```json
{
  "registered": 1,
  "failed": 1,
  "actions": [...],  // Successfully registered actions
  "failures": [
    {
      "name": "invalid_action",
      "reason": "Name cannot be blank"
    }
  ]
}
```

### 4. What Happens on Connector Restart

**Scenario 1: Action slug changed**
```yaml
# Old config:
- id: restart_server
  name: "Restart Server"

# New config:
- id: restart_database  # ID changed
  name: "Restart Database"
```

**Backend receives:**
```json
{"actions": [{"slug": "restart_database", "name": "Restart Database", ...}]}
```

**Backend should:**
1. Create new action with `slug: "restart_database"`
2. Delete action with `slug: "restart_server"` (not in new list)

---

**Scenario 2: Action removed**
```yaml
# Old config:
- id: restart_server
- id: check_status

# New config:
- id: restart_server  # check_status removed
```

**Backend receives:**
```json
{"actions": [{"slug": "restart_server", ...}]}
```

**Backend should:**
1. Keep/update action with `slug: "restart_server"`
2. Delete action with `slug: "check_status"` (not in new list)

---

**Scenario 3: Only display name changed**
```yaml
# Old config:
- id: restart_server
  name: "Restart Server"

# New config:
- id: restart_server      # Slug unchanged
  name: "Restart Prod Server" # Name changed
```

**Backend receives:**
```json
{"actions": [{"slug": "restart_server", "name": "Restart Prod Server", ...}]}
```

**Backend should:**
1. Find existing action by `slug: "restart_server"`
2. Update `name` to "Restart Prod Server"

---

## Key Points

1. **Idempotent**: Calling multiple times with same data has same result
2. **Declarative**: Connector sends "here's my current state", not "do these changes"
3. **Per-connector**: Actions are scoped to the connector (from auth token)
4. **Sync behavior**: Upsert actions by slug, delete actions not in request
5. **Match on slug**: Always use `slug` for finding/matching actions
6. **Name handling**: If `name` is empty, use `slug` as `name` (don't humanize)

## Slug Format

**Connector sends slugs as-is** (admin must provide valid slug in config):
- Must be lowercase
- Alphanumeric with underscores/hyphens
- Examples: `restart_server`, `send-webhook`, `clear_cache_v2`

**Backend validation:**
```ruby
validates :slug,
  presence: true,
  uniqueness: {scope: :edge_connector_id},
  format: {with: /\A[a-z0-9][a-z0-9_-]*\z/}
```

**No normalization needed** - connector ensures slugs are valid.

## Immutability Rules

**IMMUTABLE fields** (cannot be changed after creation):
- `slug` - Changing slug creates a new action (old one is deleted)
- `action_type` - Cannot change script ↔ http

**MUTABLE fields** (can be updated):
- `name` - Human display name can change
- `description` - Description can be updated
- `timeout` - Timeout can be adjusted
- `parameters` - Parameter definitions can be modified
- `http` - HTTP configuration can be updated (for HTTP actions)

**Why slug is immutable:**
- Prevents accidental rename (creates new action instead)
- Clear that changing slug = replacing the action
- Slug is the stable identifier

---

## GET /rec/v1/deliveries - Event Delivery Format

> **📖 See [api-payload-format.md](api-payload-format.md)** for complete payload specification and field reference.
>
> **📖 See [api-examples.md](api-examples.md)** for real-world payload examples.

### Event Types

**Automatic triggers:**
- `alert.created`, `alert.updated`
- `incident.created`, `incident.updated`

**User-triggered actions:**
- `alert.action_triggered` - Action on an alert
- `incident.action_triggered` - Action on an incident
- `action.triggered` - Standalone action (no entity)

### Critical Changes Required

#### 1. Use `event_type` field (not `type`)

```ruby
{
  event_type: "alert.created"  # ← Correct (matches database)
}
```

#### 2. Use flat structure (no data.data nesting)

**❌ Wrong:**
```ruby
{data: {data: {action_name: "..."}, event: {...}}}
```

**✅ Correct:**
```ruby
{data: {action_name: "...", entity_id: "...", parameters: {...}}}
```

#### 3. Don't include action_run_id

The `delivery_id` is sufficient - backend has the FK to action_run.

### Backend Implementation

**Serialization logic:**

```ruby
# In EdgeConnectors::DeliverySerializer
def serialize_delivery(delivery, action_run)
  {
    id: delivery.id,
    event_id: delivery.event_id,
    event_type: event_type_for_action_run(action_run),
    timestamp: delivery.created_at,
    data: serialize_action_data(action_run)
  }
end

def event_type_for_action_run(action_run)
  if action_run.alert_id.present?
    "alert.action_triggered"
  elsif action_run.incident_id.present?
    "incident.action_triggered"
  else
    "action.triggered"  # Standalone
  end
end

def serialize_action_data(action_run)
  EdgeConnectors::ActionRunSerializer.new(
    action_run,
    adapter: :attributes
  ).as_json.merge(
    entity_id: action_run.alert_id || action_run.incident_id
  ).compact  # Remove nil entity_id for standalone actions
end
```

**ActionRunSerializer updates:**
```ruby
# Remove action_run_id from attributes
class EdgeConnectors::ActionRunSerializer < ActiveModel::Serializer
  attributes :action_name, :parameters

  belongs_to :triggered_by, serializer: EdgeConnectors::UserSerializer

  def action_name
    object.action.name
  end

  # Don't include action_run_id - delivery_id is sufficient
end
```
