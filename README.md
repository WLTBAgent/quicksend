# quicksend

A simple self-hosted file transfer service. Upload files associated with a pre-shared key, and recipients can list and download them. Files are automatically deleted after download, or purged after 24 hours by a background janitor.

## API

All endpoints require an `X-Key` header with a 32-character pre-shared key.

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/upload` | Upload a file (multipart form, field name `file`) |
| `GET` | `/list` | List available files for the key (JSON) |
| `GET` | `/download/<name>` | Download a file (removed after download) |

### Examples

```bash
KEY="abcdefghijklmnopqrstuvwxyz123456"

# Upload
curl -X POST -H "X-Key: $KEY" -F "file=@document.pdf" https://quicksend.example.com/upload

# List
curl -H "X-Key: $KEY" https://quicksend.example.com/list

# Download
curl -H "X-Key: $KEY" -O https://quicksend.example.com/download/document.pdf
```

## CLI Client

Quicksend includes a CLI client (`qs`) for managing peers and transferring files. Build it with:

```bash
go build -o qs ./cmd/qs
```

### Usage

```bash
# Add a peer (nickname, URL, 32-char key)
qs add-key alice https://quicksend.example.com abcdefghijklmnopqrstuvwxyz123456

# List configured peers
qs peers

# List files from a peer (numbered output)
qs list alice

# Download a file by index
qs download alice 1

# Upload a file
qs upload alice document.pdf

# Remove a peer
qs rm-key alice
```

Config is stored at `$XDG_CONFIG_HOME/quicksend/config.json` (defaults to `~/.config/quicksend/config.json`).

## Deployment

Quicksend is designed to run behind a TLS proxy such as Caddyserver or nginx.

## How It Works

- Files are stored on disk, keyed by the 32-character pre-shared key
- Metadata is persisted in a JSON file alongside the data
- Downloaded files are automatically removed from the server
- An hourly janitor job removes any files older than 24 hours

## License

[MIT](LICENSE)
