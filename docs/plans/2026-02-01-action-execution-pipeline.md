# Action Execution Pipeline Design

**Date:** 2026-02-01
**Status:** Approved

## Overview

Wire gesture detection to action bindings and plugin execution. When a gesture is recognized, look up the bound action in the database and execute the corresponding plugin.

## Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Action lookup | Direct DB query | Simple, always fresh, SQLite sub-millisecond |
| No action bound | Silent skip | Expected during setup, no log spam |
| Plugin failure | Log and continue | Don't block real-time pipeline |
| Scope | Full CRUD API | Pipeline useless without way to create bindings |

## Section 1: Actions Repository (Store Layer)

### Data Structure

```go
// Action represents a gesture-to-plugin binding
type Action struct {
    ID         string
    GestureID  string
    PluginName string
    ActionName string
    Config     json.RawMessage
    Enabled    bool
    CreatedAt  time.Time
}
```

### Repository Methods

| Method | Purpose |
|--------|---------|
| `Create(action *Action) error` | Insert new action binding |
| `GetByID(id string) (*Action, error)` | Fetch single action |
| `GetByGestureID(gestureID string) (*Action, error)` | Lookup action for a gesture (used by pipeline) |
| `List() ([]*Action, error)` | List all actions |
| `Update(action *Action) error` | Modify existing action |
| `Delete(id string) error` | Remove action binding |

### Key Points

- One action per gesture (1:1 relationship)
- `GetByGestureID` returns `nil, nil` when no action exists (silent skip)
- Uses existing `actions` table from migrations
- ID generated via UUID

## Section 2: Actions REST API

### Endpoints

| Method | Path | Purpose |
|--------|------|---------|
| `POST` | `/api/actions` | Create action binding |
| `GET` | `/api/actions` | List all actions |
| `GET` | `/api/actions/{id}` | Get single action |
| `PUT` | `/api/actions/{id}` | Update action |
| `DELETE` | `/api/actions/{id}` | Delete action |

### Request/Response DTOs

```go
// CreateActionRequest
{
    "gesture_id": "uuid",
    "plugin_name": "system-control",
    "action_name": "volume_up",
    "config": {"amount": 10}
}

// ActionResponse
{
    "id": "uuid",
    "gesture_id": "uuid",
    "plugin_name": "system-control",
    "action_name": "volume_up",
    "config": {"amount": 10},
    "enabled": true
}
```

### Validation

- Verify `gesture_id` exists before creating
- Verify `plugin_name` exists via PluginManager
- Verify `action_name` is in plugin's action list
- Reject duplicate bindings for same gesture

### File

`internal/server/api/actions.go`

## Section 3: Pipeline Wiring

### Implementation

```go
func (a *App) executeAction(gestureID, gestureName string) {
    // 1. Look up action binding
    action, err := a.config.Store.Actions().GetByGestureID(gestureID)
    if err != nil {
        log.Printf("Error looking up action: %v", err)
        return
    }
    if action == nil || !action.Enabled {
        return // No action bound or disabled - silent skip
    }

    // 2. Get plugin
    plug, err := a.pluginMgr.Get(action.PluginName)
    if err != nil {
        log.Printf("Plugin not found: %s", action.PluginName)
        return
    }

    // 3. Build request
    req := &plugin.Request{
        Action:  action.ActionName,
        Gesture: gestureName,
        Config:  action.Config,
    }

    // 4. Execute (async to not block pipeline)
    go func() {
        resp, err := a.pluginExec.Execute(plug, req)
        if err != nil {
            log.Printf("Plugin execution failed: %v", err)
            return
        }
        if !resp.Success {
            log.Printf("Plugin returned error: %s", resp.Error)
        }
    }()
}
```

### Key Points

- Async execution via goroutine - pipeline never blocks on plugin
- Silent skip when no action bound
- Logs errors but continues

## Section 4: Testing Strategy

### Unit Tests

| File | Coverage |
|------|----------|
| `internal/store/action_test.go` | Repository CRUD operations, GetByGestureID returns nil for unbound |
| `internal/server/api/actions_test.go` | API endpoints, validation errors, duplicate rejection |
| `internal/app/pipeline_test.go` | executeAction with mock store and mock plugin executor |

### New Mocks Needed

```go
// MockActionRepository for pipeline tests
type MockActionRepository struct {
    actions map[string]*Action
}

// MockPluginExecutor for pipeline tests
type MockPluginExecutor struct {
    responses map[string]*Response
}
```

### Integration Test (Stretch)

Full flow: create gesture → bind action → trigger detection → verify plugin called

## Section 5: File Summary

### New Files

| File | Purpose |
|------|---------|
| `internal/store/action.go` | ActionRepository implementation |
| `internal/store/action_test.go` | Repository tests |
| `internal/server/api/actions.go` | REST API handlers |
| `internal/server/api/actions_test.go` | API tests |

### Modified Files

| File | Changes |
|------|---------|
| `internal/app/pipeline.go` | Wire executeAction to store + plugin executor |
| `internal/app/app.go` | Add Actions() accessor method |
| `internal/store/store.go` | Add Actions() repository accessor |
| `internal/server/server.go` | Register /api/actions routes |

### Dependencies

None new - uses existing store, plugin packages.
