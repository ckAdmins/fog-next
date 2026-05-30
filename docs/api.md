# REST API Reference

Base URL: `http://<server>/api/v1`

All requests and responses use `application/json`. Authenticated endpoints require:

```
Authorization: Bearer <access_token>
```

---

## Authentication

### POST /auth/login

Obtain an access + refresh token pair.

**Rate limited:** 10 requests/second per IP, burst 20.

**Request body:**
```json
{ "username": "fog", "password": "secret" }
```

**Response `200`:**
```json
{
  "accessToken": "eyJ...",
  "refreshToken": "eyJ...",
  "expiresIn": 900
}
```

**Response `401`:** Invalid credentials.

---

### POST /auth/refresh

Exchange a refresh token for a new access token.

**Rate limited:** same as login.

**Request body:**
```json
{ "refreshToken": "eyJ..." }
```

**Response `200`:** Same shape as login response.

---

### POST /auth/logout _(auth required)_

Invalidates the current session token server-side.

**Response `204`**

---

## Hosts

### GET /hosts _(auth required)_

List hosts with optional filtering and cursor-based pagination.

| Query param | Type | Description |
|-------------|------|-------------|
| `q` | string | Search by name or description (case-insensitive) |
| `cursor` | UUID | Pagination cursor (last `id` from previous page) |
| `limit` | int | Page size (default 50) |

**Response `200`:**
```json
{
  "data": [
    {
      "id": "uuid",
      "name": "WORKSTATION-01",
      "description": "Lab PC",
      "ip": "192.168.1.101",
      "imageId": "uuid",
      "kernel": "",
      "init": "",
      "kernelArgs": "",
      "isEnabled": true,
      "useAad": false,
      "useWol": true,
      "lastContact": "2026-04-16T12:00:00Z",
      "deployedAt": null,
      "createdAt": "2026-01-01T00:00:00Z",
      "updatedAt": "2026-04-16T12:00:00Z"
    }
  ],
  "total": 42
}
```

---

### POST /hosts _(auth required)_

Create a new host.

**Request body:**
```json
{
  "name": "WORKSTATION-01",
  "description": "Lab PC",
  "ip": "192.168.1.101",
  "imageId": "uuid",
  "isEnabled": true,
  "useWol": true
}
```

**Response `201`:** Created host object.

---

### GET /hosts/{id} _(auth required)_

**Response `200`:** Single host object.  
**Response `404`:** Host not found.

---

### PUT /hosts/{id} _(auth required)_

Update a host. Send only the fields you want to change.

**Response `200`:** Updated host object.

---

### DELETE /hosts/{id} _(auth required)_

**Response `204`**

---

### GET /hosts/{id}/macs _(auth required)_

List MAC addresses for a host.

**Response `200`:**
```json
{
  "data": [
    { "id": "uuid", "hostId": "uuid", "mac": "aa:bb:cc:dd:ee:ff",
      "description": "eth0", "isPrimary": true, "isIgnored": false }
  ]
}
```

---

### POST /hosts/{id}/macs _(auth required)_

Add a MAC address to a host.

**Request body:**
```json
{ "mac": "aa:bb:cc:dd:ee:ff", "description": "eth0" }
```

**Response `201`:** Created MAC object.

---

### DELETE /hosts/{id}/macs/{macId} _(auth required)_

**Response `204`**

---

### GET /hosts/{id}/inventory _(auth required)_

Return the last-collected hardware inventory for a host.

**Response `200`:**
```json
{
  "cpuModel": "Intel Core i7-12700",
  "cpuCores": 12,
  "cpuFreqMhz": 3600,
  "ramMib": 16384,
  "hdModel": "Samsung SSD 870",
  "hdSizeGb": 500,
  "manufacturer": "Dell Inc.",
  "product": "OptiPlex 7090",
  "serial": "ABC1234",
  "uuid": "...",
  "biosVersion": "1.12.0",
  "osName": "Windows 11"
}
```

---

### GET /hosts/{id}/task _(auth required)_

Return the host's currently active task, or `null`.

---

### GET /pending-macs _(auth required)_

List MAC addresses seen via PXE that have not yet been assigned to a host.

---

## Images

### GET /images _(auth required)_

**Response `200`:**
```json
{
  "data": [
    {
      "id": "uuid", "name": "Windows 11 Base",
      "description": "", "path": "/opt/fog/images/win11",
      "imageType": "single", "osType": "windows",
      "sizeBytes": 21474836480,
      "isEnabled": true, "toReplicate": true,
      "storageGroupId": "uuid",
      "createdAt": "...", "updatedAt": "..."
    }
  ],
  "total": 5
}
```

### POST /images _(auth required)_

**Request body:** Image fields (name, path, imageType, osType, storageGroupId, …)

**Response `201`:** Created image object.

### GET /images/{id} / PUT /images/{id} / DELETE /images/{id} _(auth required)_

Standard CRUD. DELETE returns `204`.

---

## Tasks

### GET /tasks _(auth required)_

| Query param | Type | Description |
|-------------|------|-------------|
| `state` | string | Filter by `queued`, `active`, `complete`, `canceled`, `failed` |
| `hostId` | UUID | Filter by host |
| `limit` | int | Page size (default 50) |

**Response `200`:**
```json
{
  "data": [
    {
      "id": "uuid", "hostId": "uuid", "imageId": "uuid",
      "type": "deploy", "state": "active",
      "percent": 42, "bitsPerMinute": 104857600,
      "log": "Deploying partition 2...",
      "createdBy": "fog",
      "scheduledAt": null,
      "startedAt": "...", "completedAt": null,
      "createdAt": "...", "updatedAt": "..."
    }
  ],
  "total": 3
}
```

### POST /tasks _(auth required)_

Create and queue a task.

**Request body:**
```json
{
  "hostId": "uuid",
  "imageId": "uuid",
  "type": "deploy",
  "scheduledAt": null
}
```

Task types: `capture`, `deploy`, `multicast`, `snapin`, `wol`, `inventory`.

**Response `201`:** Created task object.

### GET /tasks/{id} _(auth required)_

**Response `200`:** Single task object.

### DELETE /tasks/{id} _(auth required)_

Cancel the task. Only queued or active tasks can be cancelled.

**Response `204`**

### POST /tasks/{id}/progress _(auth required)_

Update task progress (called by the imaging agent).

**Request body:**
```json
{
  "percent": 55,
  "bitsPerMinute": 104857600,
  "bytesTransferred": 5368709120
}
```

**Response `200`:** Updated task.

---

## Groups

### GET /groups / POST /groups _(auth required)_

List or create host groups.

**Group object:**
```json
{ "id": "uuid", "name": "Lab A", "description": "" }
```

### GET /groups/{id} / PUT /groups/{id} / DELETE /groups/{id} _(auth required)_

Standard CRUD.

### GET /groups/{id}/members _(auth required)_

**Response `200`:** `{ "data": [<Host>, …] }`

### POST /groups/{id}/members _(auth required)_

```json
{ "hostId": "uuid" }
```

### DELETE /groups/{id}/members/{hostId} _(auth required)_

**Response `204`**

---

## Snapins

### GET /snapins / POST /snapins _(auth required)_

List or create snapin records. `POST` creates the database record; use the upload endpoint to attach a file.

**Snapin object:**
```json
{
  "id": "uuid", "name": "Chrome Installer",
  "description": "", "fileName": "chrome.exe",
  "filePath": "/opt/fog/snapins/uuid/chrome.exe",
  "sizeBytes": 104857600,
  "isEnabled": true, "toReplicate": true
}
```

### GET /snapins/{id} / PUT /snapins/{id} / DELETE /snapins/{id} _(auth required)_

Standard CRUD.

### POST /snapins/{id}/upload _(auth required)_

Upload the file for a snapin. Uses multipart form data.

```
Content-Type: multipart/form-data
Field name: file
```

Allowed MIME types: `application/octet-stream`, `application/zip`, `application/x-tar`, `application/gzip`, `application/x-bzip2`, `application/x-xz`, `application/x-rpm`, `application/vnd.debian.binary-package`, `text/x-shellscript`, `text/plain`, and common Windows executable types.

**Response `200`:** Updated snapin object with file metadata.

---

## Storage

### GET /storage/groups / POST /storage/groups _(auth required)_

List or create storage groups.

**Storage group object:**
```json
{ "id": "uuid", "name": "Primary", "description": "" }
```

### GET /storage/groups/{id} / PUT /storage/groups/{id} / DELETE /storage/groups/{id} _(auth required)_

Standard CRUD.

### GET /storage/groups/{id}/nodes _(auth required)_

List nodes in a storage group.

**Storage node object:**
```json
{
  "id": "uuid", "groupId": "uuid",
  "name": "node-01", "host": "192.168.1.200",
  "path": "/opt/fog/images",
  "isMaster": true, "isEnabled": true
}
```

### POST /storage/groups/{id}/nodes _(auth required)_

Add a node to a storage group.

### GET /storage/nodes/{id} / PUT /storage/nodes/{id} / DELETE /storage/nodes/{id} _(auth required)_

Standard CRUD.

---

## Users

### GET /users / POST /users _(auth required — admin only)_

List or create users.

**User object:**
```json
{
  "id": "uuid", "username": "jdoe",
  "role": "readonly", "email": "jdoe@example.com",
  "isActive": true, "lastLoginAt": "..."
}
```

**Create body** additionally requires `"password": "…"`.

### GET /users/{id} / PUT /users/{id} / DELETE /users/{id} _(auth required — admin only)_

Standard CRUD.

### POST /users/{id}/regenerate-token _(auth required — admin only)_

Generate a new long-lived API token for the user.

**Response `200`:**
```json
{ "token": "fog_…" }
```

---

## Settings

### GET /settings _(auth required)_

| Query param | Type | Description |
|-------------|------|-------------|
| `category` | string | Filter by category |

**Response `200`:**
```json
{
  "data": [
    { "key": "FOG_WEB_HOST", "value": "192.168.1.10",
      "category": "network", "description": "FOG server IP or hostname" }
  ]
}
```

### PUT /settings/{key} _(auth required — admin only)_

```json
{ "value": "192.168.1.10" }
```

**Response `200`:** Updated setting.

### DELETE /settings/{key} _(auth required — admin only)_

Reset to default. **Response `204`**

---

## WebSocket

### GET /api/v1/ws _(auth required)_

Upgrade to a WebSocket connection. Events are JSON objects:

```json
{
  "type": "task.progress",
  "payload": { "taskId": "uuid", "percent": 55, "bitsPerMinute": 104857600 },
  "at": "2026-04-16T12:00:00Z"
}
```

| Event type | Fired when |
|------------|-----------|
| `task.progress` | Task percent/speed updated |
| `task.created` | New task queued |
| `task.complete` | Task finished successfully |
| `task.canceled` | Task cancelled |
| `host.online` | Host ping succeeded (was offline) |
| `host.offline` | Host ping failed (was online) |

---

## Error responses

All errors follow this shape:

```json
{ "error": "human-readable message" }
```

| Status | Meaning |
|--------|---------|
| `400` | Bad request (validation error, invalid input) |
| `401` | Missing or invalid/expired token |
| `403` | Authenticated but insufficient role |
| `404` | Resource not found |
| `409` | Conflict (e.g. duplicate name) |
| `429` | Rate limit exceeded |
| `500` | Internal server error |
