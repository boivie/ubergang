# Public API

Ubergang has a small public API for automation, such as provisioning MQTT
clients from a script. It's served on the admin host under `/api/v1/`.

## Authentication

Create an API key under **Users → API keys** in the web UI. The key is shown
once; only a hash of it is stored. Send it as a bearer token:

```sh
curl -H "Authorization: Bearer ug_..." https://admin.example.com/api/v1/mqtt/clients
```

API keys aren't bound to a user, and session cookies aren't accepted by the
public API. Deleting a key revokes it immediately.

### Scopes

Each key has one scope. Each scope includes the ones before it.

- `read`: list and get MQTT profiles and clients. Client passwords are hidden.
- `write`: also create, update and delete them, including client passwords.

## Conventions

- Request bodies are JSON and need `Content-Type: application/json`. Unknown
  fields are rejected, so typos aren't silently ignored.
- `POST` creates a resource and fails with `409` if it already exists. It
  returns `201` with the resource and a `Location` header.
- `PATCH` changes only the fields that are given, and fails with `404` if the
  resource doesn't exist. IDs can't be changed.
- `DELETE` returns `204`, or `404` if the resource doesn't exist.
- IDs must be 1–256 bytes and can't contain `/`, control characters or leading
  or trailing whitespace.
- Errors have this shape:

  ```json
  { "error": { "code": "not_found", "message": "MQTT client not found" } }
  ```

| Status | `code`                   | When                                    |
| ------ | ------------------------ | --------------------------------------- |
| 400    | `invalid_request`        | Malformed JSON, unknown/invalid fields  |
| 401    | `unauthorized`           | Missing or invalid API key              |
| 403    | `insufficient_scope`     | The key's scope doesn't allow it        |
| 404    | `not_found`              | No such resource or endpoint            |
| 409    | `conflict`               | Already exists, or profile still in use |
| 415    | `unsupported_media_type` | The body isn't `application/json`       |
| 422    | `profile_not_found`      | The client's profile doesn't exist      |

## MQTT profiles

```json
{
  "id": "sensors",
  "allow_publish": ["sensors/$ID/#", "rooms/$room/#"],
  "allow_subscribe": []
}
```

Topics can use `$ID` (the client's ID) and `$<key>` for any of the client's
`values`.

- `GET /api/v1/mqtt/profiles` (`read`): returns `{"profiles": [...]}`.
- `GET /api/v1/mqtt/profiles/{id}` (`read`).
- `POST /api/v1/mqtt/profiles` (`write`): `id`, `allow_publish`,
  `allow_subscribe`.
- `PATCH /api/v1/mqtt/profiles/{id}` (`write`): `allow_publish` and/or
  `allow_subscribe`.
- `DELETE /api/v1/mqtt/profiles/{id}` (`write`): fails with `409` if any
  client still uses it.

Changing a profile disconnects the clients that use it, so they reconnect with
the new permissions.

## MQTT clients

```json
{
  "id": "dev-1",
  "profile_id": "sensors",
  "password": "s3cret",
  "values": { "room": "kitchen" }
}
```

`password` is only included for keys with the `write` scope.

- `GET /api/v1/mqtt/clients` (`read`): returns `{"clients": [...]}`.
- `GET /api/v1/mqtt/clients/{id}` (`read`).
- `POST /api/v1/mqtt/clients` (`write`): `id`, `profile_id` and `password`
  (all required), and `values`.
- `PATCH /api/v1/mqtt/clients/{id}` (`write`): any of `profile_id`,
  `password` and `values` (which replaces the whole map).
- `DELETE /api/v1/mqtt/clients/{id}` (`write`).

Changing a client's password or profile, or deleting it, disconnects it.

## Example: provisioning a device

```sh
KEY=ug_...
BASE=https://admin.example.com/api/v1

PASSWORD=$(openssl rand -hex 16)

curl -X POST "$BASE/mqtt/clients" \
  -H "Authorization: Bearer $KEY" \
  -H "Content-Type: application/json" \
  -d '{"id": "dev-1", "profile_id": "sensors", "password": "'"$PASSWORD"'"}'
```
