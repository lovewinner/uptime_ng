# uptime_ng API Reference

Base path: `/api`

Authentication: protected endpoints require `Authorization: Bearer <jwt>`. WebSocket uses `ws://host/api/ws?token=<jwt>`.

Error shape:

```json
{"error":"message","code":"stable_error_code"}
```

`error` is a human-readable message kept for existing clients. `code` is stable enough for UI branching and tests.

## Auth

### `POST /auth/register`

Request:

```json
{"username":"admin","password":"secret123"}
```

Response `201`:

```json
{"token":"jwt","user_id":1,"username":"admin","role":"admin"}
```

The first registered user becomes `admin`; later users become `user`.

### `POST /auth/login`

Request:

```json
{"username":"admin","password":"secret123"}
```

Response `200`: same as register.

### `GET /auth/profile`

Response:

```json
{"id":1,"username":"admin","role":"admin"}
```

### `GET /auth/users` admin

Response:

```json
[{"id":1,"username":"admin","role":"admin","active":true}]
```

### `PATCH /auth/users/:id` admin

Request fields are optional:

```json
{"role":"user","active":true,"password":"newpass123"}
```

The API rejects self-deactivation and removal/deactivation of the last active admin.

## Monitors

### `GET /monitors`

Response:

```json
[{"monitor":{},"tags":[],"notification_ids":[1]}]
```

### `POST /monitors`

Creates an active monitor and starts it in the scheduler.

Core fields:

```json
{
  "name": "Website",
  "type": "http",
  "group_id": null,
  "url": "https://example.com",
  "hostname": "example.com",
  "port": 443,
  "method": "GET",
  "interval": 60,
  "timeout": 30,
  "max_retries": 3,
  "retry_interval": 60,
  "resend_interval": 600,
  "notification_ids": [1],
  "tag_names": ["prod"],
  "tag_colors": ["#409EFF"]
}
```

`type` may be `http`, `tcp`, `ping`, `dns`, `push`, or `group`. Group monitors can be nested through `group_id`; they run on their own `interval` and are `UP` only when every child monitor/group is `UP`.

HTTP fields:

```json
{
  "accepted_status_codes": ["200-299"],
  "headers": "X-Test: yes",
  "body": "{\"ping\":true}",
  "keyword": "ok",
  "invert_keyword": false,
  "ignore_tls": false,
  "max_redirects": 10,
  "auth_method": "basic|bearer|oauth2-cc|ntlm|mtls",
  "basic_auth_user": "user",
  "basic_auth_pass": "pass",
  "auth_domain": "DOMAIN",
  "auth_workstation": "WORKSTATION",
  "bearer_token": "token",
  "oauth_token_url": "https://issuer/token",
  "oauth_client_id": "id",
  "oauth_client_secret": "secret",
  "oauth_scopes": "scope1 scope2",
  "oauth_auth_method": "body|basic",
  "oauth_audience": "audience",
  "tls_cert": "PEM",
  "tls_key": "PEM",
  "tls_ca": "PEM",
  "retry_only_on_status_code": false,
  "cache_bust": false,
  "save_response": false,
  "save_error_response": true,
  "response_max_length": 4096
}
```

DNS fields:

```json
{"dns_resolve_type":"A|AAAA|CNAME|MX|TXT|NS","dns_resolve_server":"8.8.8.8:53"}
```

Ping fields:

```json
{"ping_count":4,"ping_per_request_timeout":1000}
```

### `GET /monitors/:id`

Returns:

```json
{"monitor":{},"tags":[],"notification_ids":[]}
```

### `PUT /monitors/:id`

Same request shape as create. Active monitors are restarted after successful update.

### `DELETE /monitors/:id`

Deletes monitor-related heartbeats, stats, incidents, tags, notifications, and stops the scheduler runner.

### `POST /monitors/:id/pause`

Sets `active=false` and stops the scheduler runner.

### `POST /monitors/:id/resume`

Sets `active=true` and starts the scheduler runner.

## Heartbeats and Incidents

### `GET /monitors/status`

Response:

```json
[{"id":1,"name":"Website","type":"http","group_id":null,"status":1,"ping_ms":42,"uptime_24h":0.999,"active":true}]
```

Group status is aggregated recursively: any child `DOWN` makes the group `DOWN`; otherwise any child `PENDING` makes it `PENDING`; otherwise a group with `UP` children is `UP`; an empty group is `PENDING`.

### `GET /monitors/:id/status`

Returns the status for a single monitor or group. Group status is calculated on the server for the requested group.

### `GET /monitors/:id/beats?period=3600`

Returns heartbeats newer than `period` seconds.

### `GET /monitors/:id/beats/important?limit=50`

Returns important heartbeats in reverse chronological order.

### `GET /monitors/:id/incidents`

Returns recent incidents.

## SLA

### `GET /monitors/:id/uptime?period=day|week|month|quarter|year`

Returns time-weighted uptime for the natural period.

### `GET /monitors/:id/uptime/data?granularity=minutely|hourly|daily&num=30`

Returns aggregated trend points.

### `GET /monitors/:id/uptime/summary`

Returns:

```json
{"uptime_24h":1,"uptime_30d":0.999,"uptime_1y":0.9999}
```

### `GET /monitors/uptime/overall?period=month`

Returns all monitors' SLA and stores a `SLAReport` snapshot.

## Notifications

### `GET /notifications`

Lists current user's notification configs.

### `POST /notifications`

Request:

```json
{"name":"Ops","type":"feishu","config":"{\"webhook_url\":\"https://...\"}"}
```

Email config example:

```json
{
  "name": "Mail",
  "type": "email",
  "config": "{\"to\":\"ops@example.com\",\"cc\":\"dev@example.com\",\"subject_template\":\"[uptime_ng] {{NAME}} {{STATUS}}\",\"body_template\":\"<p>{{MSG}}</p>\"}"
}
```

### `GET /notifications/:id`

Returns one notification config.

### `PUT /notifications/:id`

Same request shape as create.

### `DELETE /notifications/:id`

Deletes config and monitor links.

### `POST /notifications/:id/test`

Sends a real test message. Returns `400` for invalid config and `502` for provider/send failures.

## Maintenance

Maintenance windows suppress checks and notifications while active. A window with `monitor_id=null` applies to all monitors owned by the user.

### `GET /maintenance`

Lists maintenance windows.

### `POST /maintenance`

Request:

```json
{"name":"Deploy","monitor_id":1,"start_at":"2026-07-18T10:00:00Z","end_at":"2026-07-18T11:00:00Z","active":true}
```

### `PUT /maintenance/:id`

Same request shape as create.

### `DELETE /maintenance/:id`

Deletes a maintenance window.

## Import / Export

### `GET /monitors/export?ids=[1,2]`

Exports monitors, `group_path`, tags, linked notification names, and referenced notifications. Sensitive config keys are masked. Import restores groups from `group_path` before assigning child monitors.

### `POST /monitors/import/preview`

Request:

```json
{"data": {"version":"1.0","monitors":[],"notifications":[]},"strategy":"skip"}
```

Response includes new monitor count, conflicts, new tags, notification count, and masked notification count.

### `POST /monitors/import`

Request uses strategy `skip`, `overwrite`, or `copy`.

Response:

```json
{"imported":1,"created":1,"updated":0,"skipped":0,"errors":[]}
```
