# Docker Hub Cleaner

A CLI tool to manage Docker Hub images with retention policies, tag filtering, and dry-run mode.

## Features

- **Retention by days**: Keep images created within X days
- **Retention by count**: Keep last X images (lexicographically or semver ordered)
- **Tag filtering**: Filter by tag patterns with regex support
- **Dry-run mode**: Report what would be deleted without actually deleting
- **Flexible sorting**: Lexicographical or semantic version sorting
- **Prefix stripping**: Support for custom tag prefixes (e.g., `develop-1.2.3`)

## Installation

### From Source

```bash
go install github.com/ataraskov/docker-hub-cleaner/cmd/docker-hub-cleaner@latest
```

### Build Locally

```bash
git clone https://github.com/ataraskov/docker-hub-cleaner.git
cd docker-hub-cleaner
make build
```

The binary will be available at `./bin/docker-hub-cleaner`.

## Usage

### Basic Example

```bash
# Delete tags older than 30 days (dry-run)
docker-hub-cleaner \
  -u myuser \
  -r myuser/myapp \
  --keep-days 30 \
  --dry-run
```

### Keep Last N Images

```bash
# Keep last 10 tags, sorted by semver
docker-hub-cleaner \
  -u myuser \
  -r myuser/myapp \
  --keep-count 10 \
  --sort-method semver
```

### Filter by Tag Pattern

```bash
# Clean dev tags older than 7 days
docker-hub-cleaner \
  -u myuser \
  -r myuser/myapp \
  --tag-pattern "^dev-.*" \
  --keep-days 7
```

### Custom Prefix Stripping

```bash
# Keep last 5 semver tags with custom prefix (develop-1.2.3, bug-2.0.1, etc.)
docker-hub-cleaner \
  -u myuser \
  -r myuser/myapp \
  --tag-pattern "^(develop|bug)-.*" \
  --strip-prefix "^(develop|bug)-" \
  --sort-method semver \
  --keep-count 5
```

### Combine Retention Policies

```bash
# Keep tags from last 30 days OR keep last 10 tags (whichever is more)
docker-hub-cleaner \
  -u myuser \
  -r myuser/myapp \
  --keep-days 30 \
  --keep-count 10 \
  --sort-method semver
```

### Using Environment Variables

```bash
export DOCKER_HUB_USERNAME=myuser
export DOCKER_HUB_PASSWORD=mypassword

docker-hub-cleaner \
  -r myuser/myapp \
  --keep-days 30 \
  --dry-run
```

### Using Personal Access Token

```bash
docker-hub-cleaner \
  --token YOUR_PERSONAL_ACCESS_TOKEN \
  -r myuser/myapp \
  --keep-days 30
```

## Command-Line Flags

### Authentication

| Flag | Short | Environment Variable | Description |
|------|-------|---------------------|-------------|
| `--username` | `-u` | `DOCKER_HUB_USERNAME` | Docker Hub username |
| `--password` | `-p` | `DOCKER_HUB_PASSWORD` | Docker Hub password |
| `--token` | `-t` | `DOCKER_HUB_TOKEN` | Personal Access Token |

### Repository

| Flag | Short | Required | Description |
|------|-------|----------|-------------|
| `--repository` | `-r` | Yes | Repository name (format: username/repo) |

### Retention Policies

| Flag | Default | Description |
|------|---------|-------------|
| `--keep-days` | 0 | Keep images created within X days |
| `--keep-count` | 0 | Keep last X images |
| `--sort-method` | lexicographical | Sorting method: `lexicographical` or `semver` |

**Note:** At least one retention policy (`--keep-days` or `--keep-count`) must be specified.

### Filtering

| Flag | Description |
|------|-------------|
| `--tag-pattern` | Regex pattern for tags to include (e.g., `^dev-.*`) |
| `--exclude-pattern` | Regex pattern for tags to exclude |
| `--strip-prefix` | Regex pattern to strip from tag before semver parsing (e.g., `^(develop|bug)-`) |

### Execution

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--dry-run` | | false | Report changes without deleting |
| `--verbose` | `-v` | false | Verbose output |
| `--concurrency` | | 5 | Number of concurrent API requests |

## How It Works

The tool follows this processing pipeline:

1. **Authenticate** with Docker Hub using username/password or token
2. **Fetch all tags** from the repository (with pagination)
3. **Apply regex filters** to include/exclude tags
4. **Sort tags** using lexicographical or semantic version sorting
5. **Apply retention policies** to determine which tags to keep
6. **Delete tags** or report in dry-run mode
7. **Display summary** with statistics

## Retention Policy Logic

When multiple retention policies are specified (e.g., both `--keep-days` and `--keep-count`), the tool uses **OR logic**: a tag is kept if **ANY** policy says to keep it.

### Example

```bash
docker-hub-cleaner \
  -r myuser/myapp \
  --keep-days 30 \
  --keep-count 5
```

This will keep:
- All tags created in the last 30 days, **OR**
- The 5 most recent tags (even if older than 30 days)

## Semantic Version Sorting

When using `--sort-method semver`:

- Tags are validated as semantic versions
- The `v` prefix is optional (both `v1.2.3` and `1.2.3` are valid)
- Valid semver tags are sorted correctly (e.g., `v2.0.0` > `v1.10.0` > `v1.9.0`)
- Invalid semver tags are grouped separately and sorted lexicographically
- Use `--strip-prefix` to remove custom prefixes before semver validation

### Example with Prefix Stripping

If your tags follow a pattern like `develop-1.2.3`, `develop-2.0.0`, etc.:

```bash
docker-hub-cleaner \
  -r myuser/myapp \
  --tag-pattern "^develop-.*" \
  --strip-prefix "^develop-" \
  --sort-method semver \
  --keep-count 5
```

This will:
1. Filter tags matching `^develop-.*`
2. Strip the `develop-` prefix
3. Sort remaining versions semantically (`2.0.0` > `1.2.3`)
4. Keep the 5 most recent versions

## Examples

### Clean up old development tags

```bash
docker-hub-cleaner \
  -u myuser \
  -r myuser/myapp \
  --tag-pattern "^dev-" \
  --keep-days 7 \
  --dry-run
```

### Keep only stable releases

```bash
docker-hub-cleaner \
  -u myuser \
  -r myuser/myapp \
  --tag-pattern "^v[0-9]+\.[0-9]+\.[0-9]+$" \
  --sort-method semver \
  --keep-count 10
```

### Clean all tags except latest

```bash
docker-hub-cleaner \
  -u myuser \
  -r myuser/myapp \
  --exclude-pattern "^latest$" \
  --keep-days 90
```

## Safety Features

- **Dry-run mode**: Always test with `--dry-run` first
- **Detailed logging**: Use `--verbose` to see what's happening
- **Rate limiting**: Built-in rate limiting to avoid API throttling
- **Error handling**: Continues processing even if individual deletions fail

## Building

```bash
# Build for current platform
make build

# Run tests
make test

# Build for all platforms
make build-all
```

## License

MIT

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.
