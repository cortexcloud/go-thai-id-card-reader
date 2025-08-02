# Software Requirements Specification (SRS)

## Thai ID Card WebSocket Service

### 1. Overview

This document outlines the software requirements for a Go-based application that monitors for the insertion of a Thai
National ID smart card into a connected PC/SC reader. Upon successful detection and reading of a card, the application
will extract the public data and broadcast it to all connected clients via a WebSocket.

The application will be built using Go version 1.24 and the Echo web framework. It will follow the principles of
Domain-Driven Design (DDD) and adhere to the standard Go project layout.

### 2. Functional Requirements

* **FR1: Smart Card Reader Detection:** The application must continuously scan for and detect connected PC/SC compliant
  smart card readers.
* **FR2: Card Event Monitoring:** The application must monitor the status of the detected reader(s) for two primary
  events:
    * `CARD_INSERTED`: When a smart card is inserted into the reader.
    * `CARD_REMOVED`: When a smart card is removed from the reader.
* **FR3: Data Extraction:** Upon a `CARD_INSERTED` event, the application must initiate a connection to the card and
  read all available public information as specified by the Thai National ID card standard. This includes, but is not
  limited to, personal details, address, and the photo.
* **FR4: Data Serialization:** The extracted card data, including the photo, must be serialized into a structured JSON
  format. The photo should be encoded as a Base64 string.
* **FR5: WebSocket Server:** The application must host a WebSocket server using the Echo framework on a configurable
  port. The server will have a specific endpoint (e.g., `/ws`) for clients to connect.
* **FR6: Message Broadcasting:**
    * On a successful card read, the application will broadcast a JSON message of type `CARD_INSERTED` containing the
      card data to all connected WebSocket clients.
    * When a card is removed, the application will broadcast a JSON message of type `CARD_REMOVED`.
* **FR7: Error Handling & Broadcasting:** If an error occurs during card reading (e.g., unsupported card, read failure),
  the application will broadcast a JSON message of type `ERROR` containing a descriptive error message to all connected
  WebSocket clients.

### 3. Non-Functional Requirements

* **NFR1: Performance:** The time from card insertion to WebSocket message broadcast should be less than 3 seconds under
  normal conditions.
* **NFR2: Reliability:** The application should be able to run continuously and gracefully handle reader
  connection/disconnection without crashing. It should automatically resume monitoring when a reader is reconnected.
* **NFR3: Concurrency:** The WebSocket server must be able to handle multiple concurrent client connections efficiently.
* **NFR4: Platform Compatibility:** The application should be compilable and runnable on major operating systems (
  Windows, macOS, and Linux).
* **NFR5: Configuration:** Key parameters such as the server port must be configurable. The application will support
  configuration via a file (e.g., `config.yaml`) and environment variables. **Environment variables will override values
  set in the configuration file.**

### 4. System Architecture & Repository Structure

The project will follow the standard Go project layout and principles of Domain-Driven Design (DDD) to ensure a clean
separation of concerns.

```
thai-card-websocket/
├── cmd/
│   └── card-service/
│       └── main.go              # Main application entry point, wiring everything together.
├── internal/
│   ├── api/                     # Presentation Layer (Echo Framework)
│   │   ├── handler.go           # WebSocket connection handler.
│   │   └── server.go            # Echo server setup and routing.
│   ├── config/                  # Application configuration loading.
│   │   └── config.go            # Logic to load config from file and env vars.
│   ├── domain/                  # Core Domain Layer (Business Logic & Models)
│   │   ├── card.go              # Contains the ThaiIdCard struct and service interfaces.
│   │   └── message.go           # Contains WebSocket message structs.
│   └── infra/                   # Infrastructure Layer (External Concerns)
│       ├── smartcard/           # Implementation of the card reading logic.
│       │   └── pcsc_reader.go   # Interacts with the PC/SC library.
│       └── websocket/           # WebSocket broadcasting implementation.
│           └── hub.go           # Manages WebSocket clients and message broadcasting.
├── configs/
│   └── config.yaml              # Application configuration.
├── go.mod
├── go.sum
└── README.md
```

**Component Descriptions:**

* **`cmd/card-service`**: The executable part of the application. Its sole responsibility is to initialize
  dependencies (config, logger, infrastructure, domain services, API layer) and start the application.
* **`internal/config`**: Responsible for loading, parsing, and providing access to application configuration from files
  and environment variables.
* **`internal/domain`**: The heart of the application. It contains the core data structures (`ThaiIdCard`) and business
  logic, completely independent of any framework or external library. It defines interfaces for services it depends on (
  e.g., `CardReaderService`).
* **`internal/infra`**: Contains the concrete implementations of the interfaces defined in the domain. The `smartcard`
  package will contain the low-level code to communicate with the card reader hardware (via a Go PC/SC library), and the
  `websocket` package will manage the pool of connected clients.
* **`internal/api`**: The presentation layer. It handles incoming HTTP/WebSocket requests using the Echo framework and
  translates them into calls to the domain services. It is responsible for the transport layer concerns.

### 5. Data Structures (Structs)

The following Go structs will be used for data modeling and serialization.

#### 5.1. Thai ID Card Message Struct

This struct represents the public data extracted from the smart card. JSON tags are included for proper serialization.

```go
// ThaiIdCard holds all the public information from a Thai National ID card.
type ThaiIdCard struct {
CitizenID   string `json:"citizenId"`
FirstNameTH string `json:"firstNameTh"`
LastNameTH  string `json:"lastNameTh"`
FirstNameEN string `json:"firstNameEn"`
LastNameEN  string `json:"lastNameEn"`
DateOfBirth string `json:"dateOfBirth"` // Format: YYYY-MM-DD
Gender      string `json:"gender"`      // "Male" or "Female"
Address     string `json:"address"`
IssueDate   string `json:"issueDate"`  // Format: YYYY-MM-DD
ExpireDate  string `json:"expireDate"` // Format: YYYY-MM-DD
PhotoBase64 string `json:"photoBase64"` // Base64 encoded string of the photo.
}
```

#### 5.2. WebSocket Message Struct

This is a generic wrapper for all messages sent to clients, providing a consistent message format.

```go
// WebSocketMessage is the standard structure for all messages sent to clients.
type WebSocketMessage struct {
Type    string      `json:"type"` // e.g., "CARD_INSERTED", "CARD_REMOVED", "ERROR"
Payload interface{} `json:"payload"`
}
```

#### 5.3. Error Message Struct

This struct will be used as the `Payload` for `WebSocketMessage` when the `Type` is `ERROR`.

```go
// ErrorResponse defines the structure for broadcasting an error message.
type ErrorResponse struct {
Code    int    `json:"code"`    // Internal error code for specific issues.
Message string `json:"message"` // Human-readable error message.
}

// Predefined Error Messages
const (
ErrCodeReaderNotFound = 1001
ErrMsgReaderNotFound = "No smart card reader found."

ErrCodeCardNotDetected = 1002
ErrMsgCardNotDetected = "No smart card detected in the reader."

ErrCodeReadFailed = 1003
ErrMsgReadFailed = "Failed to read data from the smart card."

ErrCodeUnsupportedCard = 1004
ErrMsgUnsupportedCard = "The inserted card is not a supported Thai ID card."
)
```

### 6. Configuration Management

The application's behavior can be customized through a `config.yaml` file located in the `/configs` directory or via
environment variables. Environment variables take precedence over the configuration file.

#### 6.1. Configuration Parameters

| **Parameter**  | **YAML Key**  | **Environment Variable** | **Description**                                             | **Default Value** |
|----------------|---------------|--------------------------|-------------------------------------------------------------|-------------------|
| WebSocket Port | `server.port` | `WEBSOCKET_PORT`         | The port for the WebSocket server.                          | `8080`            |
| Log Level      | `log.level`   | `LOG_LEVEL`              | The logging level (e.g., `debug`, `info`, `warn`, `error`). | `info`            |

#### 6.2. Example `config.yaml`

```yaml
server:
  port: 8080

log:
  level: "info"
```

### Appendix

# APDU Command

Get response 0x00, 0xC0, 0x00, 0x00 + Len in hex

INS 0xB0 | Read Bin
INS 0xC0 | Get response

SELECT FILE
CLA = 0X00
INS = 0XA4
P1 = 0x04 | Direct selection by DF name (data field=DF name)
P2 = 0x00
Lc = 0x08 ( len of DF Name )

Thailand Personal DF Name
0xA0, 0X00, 0x00, 0x00, 0x54, 0x48, 0x00, 0x01

| Description  	  | CLA  	 | INS  	 | P1   	 | P2   	 | Lc   	 | Data 	                                           | Le   	 |
|-----------------|--------|--------|--------|--------|--------|--------------------------------------------------|--------|
| Select        	 | 0x00 	 | 0xA4 	 | 0X04 	 | 0x00 	 | 0x08	  | 0xA0, 0X00, 0x00, 0x00, 0x54, 0x48, 0x00, 0x01 	 | 	      |
| GET RESPONSE  	 | 0X00 	 | 0XC0 	 | 0x00 	 | 0x00 	 | 	      | 	                                                | <Len>	 |
| CID           	 | 0x80 	 | 0xB0 	 | 0x00 	 | 0x04 	 | 0x02 	 | 0x00 	                                           | 0x0D 	 |
| TH Fullname   	 | 0x80 	 | 0xB0 	 | 0x00 	 | 0x11 	 | 0x02 	 | 0x00 	                                           | 0x64 	 |
| EN Fullname   	 | 0x80 	 | 0xB0 	 | 0x00 	 | 0x75 	 | 0x02 	 | 0x00 	                                           | 0x64 	 |
| Date of birth 	 | 0x80 	 | 0xB0 	 | 0x00 	 | 0xD9 	 | 0x02 	 | 0x00 	                                           | 0x08 	 |
| Gender        	 | 0x80 	 | 0xB0 	 | 0x00 	 | 0xE1 	 | 0x02 	 | 0x00 	                                           | 0x01 	 |
| Card Issuer   	 | 0x80 	 | 0xB0 	 | 0x00 	 | 0xF6 	 | 0x02 	 | 0x00 	                                           | 0x64 	 |
| Issue Date    	 | 0x80 	 | 0xB0 	 | 0x01 	 | 0x67 	 | 0x02 	 | 0x00 	                                           | 0x08 	 |
| Expire Date   	 | 0x80 	 | 0xB0 	 | 0x01 	 | 0x6F 	 | 0x02 	 | 0x00 	                                           | 0x08 	 |
| Address       	 | 0x80 	 | 0xB0 	 | 0x15 	 | 0x79 	 | 0x02 	 | 0x00 	                                           | 0x64 	 |
| Photo_Part1/20  | 0x80 	 | 0xB0 	 | 0x01 	 | 0x7B 	 | 0x02 	 | 0x00 	                                           | 0xFF 	 |
| Photo_Part2/20  | 0x80 	 | 0xB0 	 | 0x02 	 | 0x7A 	 | 0x02 	 | 0x00 	                                           | 0xFF 	 |
| Photo_Part3/20  | 0x80 	 | 0xB0 	 | 0x03 	 | 0x79 	 | 0x02 	 | 0x00 	                                           | 0xFF 	 |
| Photo_Part4/20  | 0x80 	 | 0xB0 	 | 0x04 	 | 0x78 	 | 0x02 	 | 0x00 	                                           | 0xFF 	 |
| Photo_Part5/20  | 0x80 	 | 0xB0 	 | 0x05 	 | 0x77 	 | 0x02 	 | 0x00 	                                           | 0xFF 	 |
| Photo_Part6/20  | 0x80 	 | 0xB0 	 | 0x06 	 | 0x76 	 | 0x02 	 | 0x00 	                                           | 0xFF 	 |
| Photo_Part7/20  | 0x80 	 | 0xB0 	 | 0x07 	 | 0x75 	 | 0x02 	 | 0x00 	                                           | 0xFF 	 |
| Photo_Part8/20  | 0x80 	 | 0xB0 	 | 0x08 	 | 0x74 	 | 0x02 	 | 0x00 	                                           | 0xFF 	 |
| Photo_Part9/20  | 0x80 	 | 0xB0 	 | 0x09 	 | 0x73 	 | 0x02 	 | 0x00 	                                           | 0xFF 	 |
| Photo_Part10/20 | 0x80 	 | 0xB0 	 | 0x0A 	 | 0x72 	 | 0x02 	 | 0x00 	                                           | 0xFF 	 |
| Photo_Part11/20 | 0x80 	 | 0xB0 	 | 0x0B 	 | 0x71 	 | 0x02 	 | 0x00 	                                           | 0xFF 	 |
| Photo_Part12/20 | 0x80 	 | 0xB0 	 | 0x0C 	 | 0x70 	 | 0x02 	 | 0x00 	                                           | 0xFF 	 |
| Photo_Part13/20 | 0x80 	 | 0xB0 	 | 0x0D 	 | 0x6F 	 | 0x02 	 | 0x00 	                                           | 0xFF 	 |
| Photo_Part14/20 | 0x80 	 | 0xB0 	 | 0x0E 	 | 0x6E 	 | 0x02 	 | 0x00 	                                           | 0xFF 	 |
| Photo_Part15/20 | 0x80 	 | 0xB0 	 | 0x0F 	 | 0x6D 	 | 0x02 	 | 0x00 	                                           | 0xFF 	 |
| Photo_Part16/20 | 0x80 	 | 0xB0 	 | 0x10 	 | 0x6C 	 | 0x02 	 | 0x00 	                                           | 0xFF 	 |
| Photo_Part17/20 | 0x80 	 | 0xB0 	 | 0x11 	 | 0x6B 	 | 0x02 	 | 0x00 	                                           | 0xFF 	 |
| Photo_Part18/20 | 0x80 	 | 0xB0 	 | 0x12 	 | 0x6A 	 | 0x02 	 | 0x00 	                                           | 0xFF 	 |
| Photo_Part19/20 | 0x80 	 | 0xB0 	 | 0x13 	 | 0x69 	 | 0x02 	 | 0x00 	                                           | 0xFF 	 |
| Photo_Part20/20 | 0x80 	 | 0xB0 	 | 0x14 	 | 0x68 	 | 0x02 	 | 0x00 	                                           | 0xFF 	 |