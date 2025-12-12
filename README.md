# Volcengine SSH IP Updater (Go Version)

A Go-based utility that automatically monitors your public IP address and updates the ingress rules of a specified **Volcengine (火山引擎) Security Group** to allow SSH access only from your current IP.

This version is a rewrite of the original Python tool, offering better performance and a single binary deployment.

## Features

- 🔄 **Automatic Monitoring**: Checks public IP changes every 15 minutes (configurable).
- 🔐 **Security Group Update**: Automatically updates SSH (port 22) ingress rules.
- 🌐 **Web Interface**: Built-in Dashboard for configuration and log monitoring.
- ⚡ **High Performance**: Written in Go, lightweight and fast.
- 📦 **Zero Dependency Deployment**: Compiles to a single binary.

## Project Structure

```text
/
├── cmd/
│   └── server/       # Main entry point
├── internal/
│   ├── config/       # Database & Configuration logic
│   ├── models/       # Data models
│   ├── service/      # Core logic (IP fetcher, Volcengine API)
│   └── web/          # Gin Web Handlers
├── templates/        # HTML Templates
└── instance/         # Database storage (created at runtime)
```

## Requirements

- Go 1.20+ (for building)

## Installation & Usage

### 1. Build

```bash
go mod tidy
go build -o volcengine-updater cmd/server/main.go
```

### 2. Run

```bash
./volcengine-updater
```

The server will start at `http://localhost:5000`.

### 3. Configure

1. Open `http://localhost:5000` in your browser.
2. Go to **Settings**.
3. Enter your:
   - Volcengine Access Key & Secret Key
   - Region (e.g., `cn-beijing`)
   - Security Group ID
   - SSH Port

## Development

```bash
# Run directly
go run cmd/server/main.go
```

## License

MIT