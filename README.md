# Go Thai ID Card Reader

A Go-based service that monitors Thai National ID smart card readers and broadcasts card data to connected WebSocket clients.

## Caution

Claude code completely writes this application.

## Features

- Real-time monitoring of PC/SC smart card readers
- Automatic detection of card insertion and removal
- Extraction of public data from Thai National ID cards
- WebSocket broadcasting of card events to all connected clients
- RESTful health check endpoint
- Cross-platform support (Windows, macOS, Linux)

## Prerequisites

- Go 1.24 or higher
- PC/SC smart card reader
- PC/SC drivers installed on your system:
  - **Windows**: Usually pre-installed
  - **macOS**: Pre-installed
  - **Linux**: Install `pcscd` and `libpcsclite-dev`

## Installation

```bash
# Install dependencies
go mod download

# Build the application
go build -o card-service ./cmd/card-service
```

## Configuration

The service can be configured via `configs/config.yaml` or environment variables:

```yaml
server:
  port: 8080

log:
  level: "info"
```

Environment variables (override config file):
- `SERVER_PORT`: WebSocket server port (default: 8080)
- `LOG_LEVEL`: Logging level (default: info)

## Usage

1. Start the service:
```bash
./card-service
```

2. Connect to the WebSocket endpoint:
```
ws://localhost:8080/ws
```

3. Insert a Thai National ID card into the reader

## WebSocket Messages

### Card Inserted
```json
{
  "type": "CARD_INSERTED",
  "payload": {
    "citizenId": "1234567890123",
    "firstNameTh": "ชื่อ",
    "lastNameTh": "นามสกุล",
    "firstNameEn": "FIRSTNAME",
    "lastNameEn": "LASTNAME",
    "dateOfBirth": "1990-01-01",
    "gender": "male",
    "address": {
      "houseNo": "28/70",
      "moo": "",
      "soi": "สุขขุมวิท 70 แยก 5-1",
      "street": "",
      "subdistrict": "จอมทอง",
      "district": "จอมทอง",
      "province": "กรุงเทพมหานคร",
      "fullAddress": "28/70 ซอยสุขขุมวิท 70 แยก 5-1 แขวงจอมทอง เขตจอมทอง จังหวัดกรุงเทพมหานคร"
    },
    "issueDate": "2020-01-01",
    "expireDate": "2030-01-01",
    "photoBase64": "..."
  }
}
```

### Card Removed
```json
{
  "type": "CARD_REMOVED",
  "payload": null
}
```

### Error
```json
{
  "type": "ERROR",
  "payload": {
    "code": 1001,
    "message": "No smart card reader found."
  }
}
```

## Error Codes

| Code | Message |
|------|---------|
| 1001 | No smart card reader found |
| 1002 | No smart card detected in the reader |
| 1003 | Failed to read data from the smart card |
| 1004 | The inserted card is not a supported Thai ID card |

## API Endpoints

- `GET /health` - Health check endpoint
- `GET /ws` - WebSocket endpoint

## Development

### Project Structure
```
thai-card-websocket/
├── cmd/card-service/       # Application entry point
├── internal/
│   ├── api/               # HTTP/WebSocket handlers
│   ├── config/            # Configuration management
│   ├── domain/            # Domain models and interfaces
│   └── infra/             # Infrastructure implementations
│       ├── smartcard/     # PC/SC card reader
│       └── websocket/     # WebSocket hub
├── configs/               # Configuration files
└── go.mod
```

### Running Tests
```bash
go test ./...
```

### Lint

```bash
golangci-lint run
```

### Inspired from

* https://github.com/somprasongd/go-thai-smartcard
* https://github.com/bencomtech/ThaiNationalIDCard.NET

## License

This project is licensed under the MIT License.