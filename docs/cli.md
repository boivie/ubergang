# ugctl

`ugctl` is a command-line client for Ubergang's [public API](public-api.md),
for people as well as scripts and agents.

## Installation

Download the archive for your platform from the
[releases](https://github.com/boivie/ubergang/releases). There are builds for
Linux, macOS and Windows, on `amd64` and `arm64`.

```sh
VERSION=0.1.0
PLATFORM=linux_amd64 # or linux_arm64, darwin_amd64, darwin_arm64
BASE=https://github.com/boivie/ubergang/releases/download/v$VERSION

curl -fsSLO "$BASE/ugctl_${VERSION}_$PLATFORM.tar.gz"
tar -xzf "ugctl_${VERSION}_$PLATFORM.tar.gz" ugctl
sudo install ugctl /usr/local/bin/
```

Each archive has a signed build provenance attestation, which proves that it
was built by this repository's release workflow:

```sh
gh attestation verify "ugctl_${VERSION}_$PLATFORM.tar.gz" --repo boivie/ubergang
```

To build it from source instead, which doesn't need the frontend:

```sh
go build -o ugctl ./cmd/ugctl
```

## Authentication

`ugctl` uses an API key, created under **Users → API keys** in the web UI. Its
scope decides what `ugctl` can do; see [scopes](public-api.md#scopes).

Either save the server's URL and the key with `ugctl login`, which prompts for
the key and checks that it works:

```sh
ugctl login https://admin.example.com
```

Or set them in the environment, which is better suited for scripts and agents:

| Variable           | Meaning                                              |
| ------------------ | ---------------------------------------------------- |
| `UBERGANG_URL`     | The admin host, such as `https://admin.example.com`  |
| `UBERGANG_API_KEY` | The API key                                          |
| `UBERGANG_CONFIG`  | Config file, default `~/.config/ubergang/ugctl.yaml` |

The `--url` flag takes precedence over `UBERGANG_URL`, and the environment
takes precedence over the config file. A key in the config file is only sent
to the server it was saved for, and keys are never sent over plain `http`,
except to `localhost`.

`ugctl logout` removes the config file. To revoke a key, delete it in the web
UI.

## Commands

```txt
ugctl login [url]
ugctl logout
ugctl version

ugctl mqtt profiles list
ugctl mqtt profiles get <id>
ugctl mqtt profiles create <id> [--publish TOPIC]... [--subscribe TOPIC]...
ugctl mqtt profiles update <id> [--publish TOPIC]... [--subscribe TOPIC]...
                                [--clear-publish] [--clear-subscribe]
ugctl mqtt profiles delete <id>

ugctl mqtt clients list
ugctl mqtt clients get <id>
ugctl mqtt clients create <id> --profile PROFILE PASSWORD [--value K=V]...
ugctl mqtt clients update <id> [--profile PROFILE] [PASSWORD]
                               [--value K=V]... [--clear-values]
ugctl mqtt clients delete <id>
```

`PASSWORD` is one of `--password-stdin`, `--generate-password` or
`--password <password>`. Prefer the first two, as other local users can see
command-line arguments.

In `update`, flags that aren't given leave things unchanged. The given topics
replace a profile's topics, and the given values replace all of a client's
values. Run `ugctl <command> --help` for details.

### Example

```sh
ugctl mqtt profiles create sensors --publish 'sensors/$ID/#'
ugctl mqtt clients create dev-1 --profile sensors --generate-password \
  --value room=kitchen
```

## Scripting

- `-o json` prints the API's JSON objects, as documented in the
  [public API](public-api.md). `list` prints the API's list response, such as
  `{"clients": [...]}`, and `delete` prints nothing.
- Errors are printed to stderr. With `-o json`, they have the API's error
  shape, `{"error": {"code": "...", "message": "..."}}`, where `code` is one of
  the API's error codes, `usage` or `error`.
- `ugctl` never prompts, except for the key in `ugctl login` when stdin is a
  terminal. Otherwise it reads the key from stdin.

The exit code tells what went wrong:

| Code | Meaning                                                     |
| ---- | ----------------------------------------------------------- |
| 0    | Success                                                     |
| 1    | Other errors, such as a network error or an invalid request |
| 2    | Invalid command-line usage                                  |
| 3    | Not found (`not_found`)                                     |
| 4    | Conflict (`conflict`), such as when it already exists       |
| 5    | Invalid key, or not allowed by its scope                    |

For example, to provision a device unless it already exists:

```sh
ugctl mqtt clients get dev-1 -o json >dev-1.json
status=$?
if [ "$status" -eq 3 ]; then
  ugctl mqtt clients create dev-1 --profile sensors --generate-password \
    -o json >dev-1.json || exit
elif [ "$status" -ne 0 ]; then
  exit "$status"
fi
PASSWORD=$(jq -r .password dev-1.json)
```
