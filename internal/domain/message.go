package domain

type WebSocketMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

const (
	ErrCodeReaderNotFound  = 1001
	ErrMsgReaderNotFound   = "No smart card reader found."
	
	ErrCodeCardNotDetected = 1002
	ErrMsgCardNotDetected  = "No smart card detected in the reader."
	
	ErrCodeReadFailed      = 1003
	ErrMsgReadFailed       = "Failed to read data from the smart card."
	
	ErrCodeUnsupportedCard = 1004
	ErrMsgUnsupportedCard  = "The inserted card is not a supported Thai ID card."
)