# Rootly Edge Connector Documentation

## 📁 Documentation Structure

### For Backend Developers (`api/`)

⚠️ **Internal use only** - Not for customer documentation

**Read in order:**

1. **[00-implementation-guide.md](api/00-implementation-guide.md)** 📖 **START HERE**
   - Complete implementation guide for all 3 endpoints
   - Sync behavior, immutability, validation

2. **[01-action-registration.md](api/01-action-registration.md)** - `POST /rec/v1/actions`
   - Payload examples (script, HTTP, multiple actions)
   - Field mapping: id → slug, name, description

3. **[02-event-payloads.md](api/02-event-payloads.md)** - `GET /rec/v1/deliveries`
   - Event types and structure
   - Serializer field reference

4. **[03-event-examples.md](api/03-event-examples.md)** - Event payload examples
   - Alert, incident, action_triggered events

---

### For End Users (`user-guide/`)

**Quick start:** [template-syntax.md](user-guide/template-syntax.md)

User guides and reference:
- **[template-syntax.md](user-guide/template-syntax.md)** - Liquid template quick start
  - Basic syntax, array access, filters
  - Common patterns
  - Real-world examples

- **[liquid-filters.md](user-guide/liquid-filters.md)** - Complete filter reference
  - 50+ filters documented
  - Array, string, number, date filters
  - Usage examples

- **[systemd-installation.md](user-guide/systemd-installation.md)** - Production deployment
  - Linux/systemd installation
  - Service configuration
  - Logging and monitoring

---

### For Developers (`development/`)

Internal development guides:
- **[development.md](development/development.md)** - Local development setup
  - Build instructions
  - Running locally
  - Testing

- **[docker-development.md](development/docker-development.md)** - Docker development
  - Docker setup
  - Local testing

- **[test-coverage-improvements.md](development/test-coverage-improvements.md)** - Testing guide
  - Coverage report (45% → 81%)
  - Gap analysis
  - Test fixtures
  - Future improvements

---

## Quick Navigation

**I'm a backend developer implementing the API:**
→ Read [api/BACKEND_API_NOTES.md](api/BACKEND_API_NOTES.md) first

**I'm configuring actions for my team:**
→ Read [user-guide/template-syntax.md](user-guide/template-syntax.md) first

**I'm deploying to production:**
→ Read [user-guide/systemd-installation.md](user-guide/systemd-installation.md) first

**I'm developing/contributing to the connector:**
→ Read [development/development.md](development/development.md) first

---

## File Organization

```
docs/
├── README.md                           # This file - navigation guide
│
├── api/                                # Backend implementation
│   ├── BACKEND_API_NOTES.md           # 📖 START HERE (backend)
│   ├── action-registration-example.md # Action registration spec
│   ├── api-payload-format.md          # Event payload spec
│   └── api-examples.md                # Real-world examples
│
├── user-guide/                         # End user documentation
│   ├── template-syntax.md             # 📖 START HERE (users)
│   ├── liquid-filters.md              # Complete filter reference
│   └── systemd-installation.md        # Production deployment
│
└── development/                        # Internal development
    ├── development.md                 # Local dev setup
    ├── docker-development.md          # Docker dev
    └── test-coverage-improvements.md  # Testing guide
```

---

## All Documentation Files

| File | Audience | Purpose |
|------|----------|---------|
| **api/BACKEND_API_NOTES.md** | Backend Dev | Implementation guide for all 3 endpoints |
| **api/action-registration-example.md** | Backend Dev | Action registration payload examples |
| **api/api-payload-format.md** | Backend Dev | Event structure and field reference |
| **api/api-examples.md** | Backend Dev | Real-world event payloads |
| **user-guide/template-syntax.md** | End User | Quick start for Liquid templates |
| **user-guide/liquid-filters.md** | End User | Complete filter documentation |
| **user-guide/systemd-installation.md** | Ops/SRE | Production deployment guide |
| **development/development.md** | Contributor | Local development setup |
| **development/docker-development.md** | Contributor | Docker development |
| **development/test-coverage-improvements.md** | Contributor | Testing and coverage report |

---

## See Also

- Main README: [../README.md](../README.md)
- Example configs: [../actions.example.yml](../actions.example.yml), [../actions.example.dev.yml](../actions.example.dev.yml)
