# goimports-all

A drop-in replacement for `goimports` that supports Go's `./...` package pattern.

## Why?

The standard `goimports` tool doesn't support the `./...` pattern that works everywhere else in Go tooling:
```bash
goimports -w ./...
# stat ./...: no such file or directory
```

This tool fixes that:
```bash
goimports-all -w ./...
# ✓ works
```

## Installation
```bash
go install github.com/uradical/goimports-all@latest
```

## Usage
```bash
goimports-all [flags] [path ...]
```

Supports all the same flags as `goimports`:

| Flag | Description |
|------|-------------|
| `-w` | Write result to (source) file instead of stdout |
| `-l` | List files whose formatting differs |
| `-d` | Display diffs instead of rewriting files |
| `-e` | Report all errors (not just first 10) |
| `-local` | Put imports beginning with this string after 3rd-party packages |
| `-format-only` | Do not fix imports, just format |
| `-srcdir` | Choose imports as if source code is in srcdir |
| `-v` | Verbose logging |

## Examples

Format all files in a repository:
```bash
goimports-all -w ./...
```

List files that need formatting:
```bash
goimports-all -l ./...
```

Group local imports separately:
```bash
goimports-all -w -local github.com/myorg/myproject ./...
```

Show diffs without writing:
```bash
goimports-all -d ./...
```

## License

MIT