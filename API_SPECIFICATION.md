# OpenCode Project Manager API Specification

This document provides a comprehensive specification for the OpenCode Project Manager REST API.

**Base URL:** `http://localhost:8090`  
**Authentication:** JWT required for all endpoints unless specified otherwise.  
**Header:** `Authorization: Bearer <token>`

---

## Configuration Management

The Configuration Management API allows users to manage AI agent settings for each project. Configurations are versioned, allowing for history tracking and rollbacks.

### GET /api/projects/:id/config
Retrieves the currently active OpenCode configuration for a specific project.

**Authentication:** Required (JWT)

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | `uuid` | The unique identifier of the project. |

**Success Response (200 OK):**
```json
{
  "id": "a1b2c3d4-e5f6-7g8h-9i0j-k1l2m3n4o5p6",
  "project_id": "b2c3d4e5-f6g7-8h9i-0j1k-l2m3n4o5p6q7",
  "version": 1,
  "is_active": true,
  "model_provider": "openai",
  "model_name": "gpt-4o-mini",
  "model_version": "2024-07-18",
  "api_endpoint": "https://api.openai.com/v1",
  "temperature": 0.7,
  "max_tokens": 4096,
  "enabled_tools": ["file_ops", "terminal", "web_search"],
  "tools_config": {},
  "system_prompt": "You are a helpful AI assistant specializing in Go and React development.",
  "max_iterations": 10,
  "timeout_seconds": 300,
  "created_by": "c3d4e5f6-g7h8-9i0j-k1l2-m3n4o5p6q7r8",
  "created_at": "2026-01-19T14:15:00Z",
  "updated_at": "2026-01-19T14:15:00Z"
}
```

**Error Responses:**
- `400 Bad Request`: `{"error": "invalid project ID"}`
- `401 Unauthorized`: `{"error": "user not authenticated"}`
- `404 Not Found`: `{"error": "config not found"}`
- `500 Internal Server Error`: `{"error": "database connection error"}`

---

### POST /api/projects/:id/config
Creates a new configuration version for a project. This action automatically deactivates the previous configuration and sets the new one as active.

**Authentication:** Required (JWT)

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | `uuid` | The unique identifier of the project. |

**Request Body Schema:**
| Field | Type | Required | Constraints | Description |
|-------|------|----------|-------------|-------------|
| **model_provider** | *string* | Yes | `openai`, `anthropic`, `custom` | AI model provider. |
| **model_name** | *string* | Yes | - | Name of the model. |
| **model_version** | *string* | No | - | Specific version of the model. |
| **api_endpoint** | *string* | No | Must be HTTPS | Custom API endpoint for the provider. |
| **api_key** | *string* | No | - | API key (encrypted on storage, never returned in responses). |
| **temperature** | *float64*| No | 0.0 - 2.0 | Sampling temperature (default 0.7). |
| **max_tokens** | *int* | No | 1 - 128000 | Maximum tokens to generate. |
| **enabled_tools** | *[]string*| Yes | - | List of tools to enable (`file_ops`, `web_search`, `code_exec`, `terminal`). |
| **tools_config** | *object* | No | - | Configuration for specific tools. |
| **system_prompt** | *string* | No | - | Custom system instruction for the AI. |
| **max_iterations**| *int* | No | 1 - 50 | Max agent thought cycles (default 10). |
| **timeout_seconds**| *int* | No | 60 - 3600 | Max execution time (default 300). |

**Example Request:**
```json
{
  "model_provider": "openai",
  "model_name": "gpt-4o-mini",
  "api_key": "sk-proj-...",
  "temperature": 0.8,
  "max_tokens": 8192,
  "enabled_tools": ["file_ops", "web_search", "terminal"],
  "max_iterations": 15,
  "timeout_seconds": 600,
  "system_prompt": "Always write clean, tested Go code."
}
```

**Success Response (201 Created):**
```json
{
  "id": "d5e6f7g8-h9i0-j1k2-l3m4-n5o6p7q8r9s0",
  "project_id": "b2c3d4e5-f6g7-8h9i-0j1k-l2m3n4o5p6q7",
  "version": 2,
  "is_active": true,
  "model_provider": "openai",
  "model_name": "gpt-4o-mini",
  "temperature": 0.8,
  "max_tokens": 8192,
  "enabled_tools": ["file_ops", "web_search", "terminal"],
  "max_iterations": 15,
  "timeout_seconds": 600,
  "system_prompt": "Always write clean, tested Go code.",
  "created_by": "c3d4e5f6-g7h8-9i0j-k1l2-m3n4o5p6q7r8",
  "created_at": "2026-01-19T14:30:00Z",
  "updated_at": "2026-01-19T14:30:00Z"
}
```

**Error Responses:**
- `400 Bad Request`: `{"error": "Key: 'CreateConfigRequest.ModelProvider' Error:Field validation for 'ModelProvider' failed on the 'oneof' tag"}`
- `401 Unauthorized`: `{"error": "user not authenticated"}`

---

### GET /api/projects/:id/config/versions
Retrieves the full configuration history for a specific project, ordered by version descending.

**Authentication:** Required (JWT)

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | `uuid` | The unique identifier of the project. |

**Success Response (200 OK):**
```json
[
  {
    "id": "d5e6f7g8-h9i0-j1k2-l3m4-n5o6p7q8r9s0",
    "project_id": "b2c3d4e5-f6g7-8h9i-0j1k-l2m3n4o5p6q7",
    "version": 2,
    "is_active": true,
    "model_provider": "openai",
    "model_name": "gpt-4o-mini",
    "temperature": 0.8,
    "max_tokens": 8192,
    "enabled_tools": ["file_ops", "web_search", "terminal"],
    "max_iterations": 15,
    "timeout_seconds": 600,
    "created_by": "c3d4e5f6-g7h8-9i0j-k1l2-m3n4o5p6q7r8",
    "created_at": "2026-01-19T14:30:00Z",
    "updated_at": "2026-01-19T14:30:00Z"
  },
  {
    "id": "a1b2c3d4-e5f6-7g8h-9i0j-k1l2m3n4o5p6",
    "project_id": "b2c3d4e5-f6g7-8h9i-0j1k-l2m3n4o5p6q7",
    "version": 1,
    "is_active": false,
    "model_provider": "openai",
    "model_name": "gpt-4o-mini",
    "temperature": 0.7,
    "max_tokens": 4096,
    "enabled_tools": ["file_ops", "terminal"],
    "max_iterations": 10,
    "timeout_seconds": 300,
    "created_by": "c3d4e5f6-g7h8-9i0j-k1l2-m3n4o5p6q7r8",
    "created_at": "2026-01-19T14:15:00Z",
    "updated_at": "2026-01-19T14:15:00Z"
  }
]
```

---

### POST /api/projects/:id/config/rollback/:version
Rolls back the project configuration to a specific previous version. This creates a *new* configuration version with the data from the target version.

**Authentication:** Required (JWT)

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | `uuid` | The unique identifier of the project. |
| `version` | `int` | The version number to roll back to. |

**Success Response (200 OK):**
```json
{
  "message": "config rolled back successfully"
}
```

**Error Responses:**
- `400 Bad Request`: `{"error": "invalid version"}`
- `404 Not Found`: `{"error": "version 1 not found"}`
- `500 Internal Server Error`: `{"error": "failed to rollback configuration"}`

---

## Validation Rules

| Rule | Constraints | Default | Description |
|------|-------------|---------|-------------|
| **Model Providers** | `openai`, `anthropic`, `custom` | - | Supported AI service providers. |
| **Temperature** | 0.0 - 2.0 | 0.7 | Controls randomness. Lower is more deterministic. |
| **Max Tokens** | 1 - 128,000 | - | Limit on generated tokens. Model-specific limits apply. |
| **Max Iterations**| 1 - 50 | 10 | Safety limit on agent thought/action cycles. |
| **Timeout Seconds**| 60 - 3,600 | 300 | Max time allowed for a single agent session. |
| **API Endpoint** | Must use HTTPS | - | Required for `custom` provider. |
| **Enabled Tools** | `file_ops`, `web_search`, `code_exec`, `terminal` | - | Array of strings. Required field. |

---

## Supported Models

The following models are natively supported in the OpenCode Model Registry:

### OpenAI
| Model Name | Max Tokens | Context Size | Input Price ($/1M) | Output Price ($/1M) |
|------------|------------|--------------|--------------------|---------------------|
| `gpt-4o` | 128,000 | 128,000 | 2.50 | 10.00 |
| `gpt-4o-mini` | 128,000 | 128,000 | 0.15 | 0.60 |
| `gpt-4` | 8,192 | 8,192 | 30.00 | 60.00 |
| `gpt-4-turbo` | 4,096 | 128,000 | 10.00 | 30.00 |
| `gpt-3.5-turbo`| 4,096 | 16,385 | 0.50 | 1.50 |

### Anthropic
| Model Name | Max Tokens | Context Size | Input Price ($/1M) | Output Price ($/1M) |
|------------|------------|--------------|--------------------|---------------------|
| `claude-3-opus-20240229` | 4,096 | 200,000 | 15.00 | 75.00 |
| `claude-3-sonnet-20240229` | 4,096 | 200,000 | 3.00 | 15.00 |
| `claude-3-haiku-20240307` | 4,096 | 200,000 | 0.25 | 1.25 |
| `claude-3.5-sonnet-20240620` | 8,192 | 200,000 | 3.00 | 15.00 |

---

## Configuration Versioning

OpenCode implements a robust versioning system for project configurations:

1.  **Immutability:** Existing configuration records are never modified. Every update (via POST) creates a new record with an incremented `version` number.
2.  **Activation:** Only one configuration can be active (`is_active: true`) per project. Creating a new version automatically deactivates the previous one.
3.  **Audit Trail:** Rollback operations create a brand-new version record containing the historical data, ensuring a complete and linear audit trail.
4.  **Security:** API keys are encrypted using AES-256-GCM before database storage. Keys are explicitly excluded from all JSON responses (`json:"-"` tag) and cannot be retrieved once set.

---

## Error Codes

The API uses standard HTTP status codes. Error responses follow the format `{"error": "message"}`.

| Code | Name | Typical Cause | Example JSON |
|------|------|---------------|--------------|
| **400** | Bad Request | Invalid UUID, validation failure, or JSON binding error. | `{"error": "invalid project ID"}` |
| **401** | Unauthorized | Missing, expired, or invalid JWT. | `{"error": "user not authenticated"}` |
| **404** | Not Found | Project, config, or version does not exist. | `{"error": "config not found"}` |
| **500** | Internal Error | Database connection failure or unexpected runtime error. | `{"error": "failed to save configuration"}` |
