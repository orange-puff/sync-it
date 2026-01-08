# sync-it

A lightweight web-based file transfer utility. Upload, download, and manage files through a simple web interface.

## Features

- Upload files via web UI
- Download files by ID
- Delete files
- View list of uploaded files with metadata
- Automatic cleanup on startup/shutdown
- Network-accessible from any device on the same network

## Project Structure

- `main.go` - Server setup and HTTP routes
- `handlers.go` - API request handlers
- `storage.go` - File storage and metadata management
- `static/` - Web UI (HTML, CSS, JavaScript)
- `uploads/` - Storage directory for uploaded files

## Building

```bash
go build -o sync-it
```

## Running

```bash
# Run on default port 80
./sync-it

# Run on custom port
./sync-it -port 8080
```

The server will display:
- Local access URL (localhost)
- Network access URL (local IP address)

Access the web interface at the provided URL.

## API Endpoints

- `GET /api/info` - Server info (IP and port)
- `POST /api/upload` - Upload a file
- `GET /api/files` - List all uploaded files
- `GET /api/download/{id}` - Download a file by ID
- `DELETE /api/delete/{id}` - Delete a file by ID
