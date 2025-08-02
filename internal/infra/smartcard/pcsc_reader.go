package smartcard

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"log"
	"time"

	"github.com/cortex-x/go-thai-id-card-reader/internal/domain"
	"github.com/ebfe/scard"
	"golang.org/x/text/encoding/charmap"
)

type PCSCReader struct {
	context           *scard.Context
	cardInsertHandler func(card *domain.ThaiIdCard, err error)
	cardRemoveHandler func()
	stopChan          chan bool
	monitoring        bool
}

func NewPCSCReader() (*PCSCReader, error) {
	ctx, err := scard.EstablishContext()
	if err != nil {
		return nil, fmt.Errorf("failed to establish context: %w", err)
	}

	return &PCSCReader{
		context:  ctx,
		stopChan: make(chan bool),
	}, nil
}

func (r *PCSCReader) StartMonitoring() error {
	if r.monitoring {
		return fmt.Errorf("already monitoring")
	}

	r.monitoring = true
	go r.monitorLoop()

	return nil
}

func (r *PCSCReader) StopMonitoring() {
	if r.monitoring {
		r.stopChan <- true
		r.monitoring = false
	}
}

func (r *PCSCReader) OnCardInserted(handler func(card *domain.ThaiIdCard, err error)) {
	r.cardInsertHandler = handler
}

func (r *PCSCReader) OnCardRemoved(handler func()) {
	r.cardRemoveHandler = handler
}

func (r *PCSCReader) monitorLoop() {
	lastState := make(map[string]bool)

	for {
		select {
		case <-r.stopChan:
			return
		default:
			readers, err := r.context.ListReaders()
			if err != nil {
				log.Printf("Error listing readers: %v", err)
				time.Sleep(2 * time.Second)
				continue
			}

			if len(readers) == 0 {
				if r.cardInsertHandler != nil {
					r.cardInsertHandler(nil, fmt.Errorf("%s", domain.ErrMsgReaderNotFound))
				}
				time.Sleep(2 * time.Second)
				continue
			}

			for _, reader := range readers {
				// Use exclusive mode for more stable connection
				card, err := r.context.Connect(reader, scard.ShareExclusive, scard.ProtocolT0|scard.ProtocolT1)

				if err == nil {
					if !lastState[reader] {
						lastState[reader] = true

						if r.cardInsertHandler != nil {
							// Add retry logic for card reading
							var cardData *domain.ThaiIdCard
							var readErr error

							for retry := 0; retry < 3; retry++ {
								cardData, readErr = r.readCard(card)
								if readErr == nil {
									break
								}

								// If applet not found, try to reconnect
								if retry < 2 && readErr != nil &&
									(readErr.Error() == "applet not found" ||
										readErr.Error() == "select applet failed: SW=6A82") {
									card.Disconnect(scard.ResetCard)
									time.Sleep(200 * time.Millisecond)
									card, err = r.context.Connect(reader, scard.ShareExclusive, scard.ProtocolT0|scard.ProtocolT1)
									if err != nil {
										break
									}
								}

								// Wait a bit before retry
								time.Sleep(100 * time.Millisecond)
							}

							r.cardInsertHandler(cardData, readErr)
						}
					}
					_ = card.Disconnect(scard.LeaveCard)
				} else {
					if lastState[reader] {
						lastState[reader] = false

						if r.cardRemoveHandler != nil {
							r.cardRemoveHandler()
						}
					}
				}
			}

			time.Sleep(500 * time.Millisecond)
		}
	}
}

func (r *PCSCReader) readCard(card *scard.Card) (*domain.ThaiIdCard, error) {
	// Add small delay before applet selection
	time.Sleep(50 * time.Millisecond)

	if err := r.selectApplet(card); err != nil {
		return nil, fmt.Errorf("%s: %w", domain.ErrMsgUnsupportedCard, err)
	}

	thaiCard := &domain.ThaiIdCard{}

	// Read CID
	data, err := r.readBinary(card, 0x00, 0x04, 0x0D)
	if err == nil {
		thaiCard.CitizenID = string(bytes.Trim(data, "\x00"))
	} else {
		log.Printf("Failed to read CID: %v", err)
	}

	// Read Thai Fullname
	data, err = r.readBinary(card, 0x00, 0x11, 0x64)
	if err == nil {
		names := r.decodeThaiString(data)
		// Thai names are space-separated
		parts := bytes.Split([]byte(names), []byte("#"))
		if len(parts) >= 2 {
			thaiCard.FirstNameTH = string(bytes.Trim(parts[0], " \x00"))
			thaiCard.LastNameTH = string(bytes.Trim(parts[1], " \x00"))
		}
	}

	// Read English Fullname
	data, err = r.readBinary(card, 0x00, 0x75, 0x64)
	if err == nil {
		names := string(bytes.Trim(data, "\x00"))
		// English names are space-separated
		parts := bytes.Split([]byte(names), []byte("#"))
		if len(parts) >= 2 {
			thaiCard.FirstNameEN = string(bytes.Trim(parts[0], " \x00"))
			thaiCard.LastNameEN = string(bytes.Trim(parts[1], " \x00"))
		}
	}

	// Read Date of Birth
	data, err = r.readBinary(card, 0x00, 0xD9, 0x08)
	if err == nil {
		thaiCard.DateOfBirth = r.formatDate(string(data))
	}

	// Read Gender
	data, err = r.readBinary(card, 0x00, 0xE1, 0x01)
	if err == nil && len(data) >= 1 {
		switch data[0] {
		case '1':
			thaiCard.Gender = "male"
		case '2':
			thaiCard.Gender = "female"
		}
	}

	// Read Issue Date
	data, err = r.readBinary(card, 0x01, 0x67, 0x08)
	if err == nil {
		thaiCard.IssueDate = r.formatDate(string(data))
	}

	// Read Expire Date
	data, err = r.readBinary(card, 0x01, 0x6F, 0x08)
	if err == nil {
		thaiCard.ExpireDate = r.formatDate(string(data))
	}

	// Read Address
	data, err = r.readBinary(card, 0x15, 0x79, 0x64)
	if err == nil {
		addressStr := r.decodeThaiString(data)
		thaiCard.Address = domain.ParseThaiAddress(addressStr)
	}

	// Read Photo
	photoData, err := r.readPhoto(card)
	if err == nil && len(photoData) > 0 {
		thaiCard.PhotoBase64 = base64.StdEncoding.EncodeToString(photoData)
	}

	return thaiCard, nil
}

func (r *PCSCReader) selectApplet(card *scard.Card) error {
	cmd := []byte{0x00, 0xa4, 0x04, 0x00, 0x08, 0xa0, 0x00, 0x00, 0x00, 0x54, 0x48, 0x00, 0x01}

	rsp, err := card.Transmit(cmd)
	if err != nil {
		return err
	}

	if len(rsp) < 2 {
		return fmt.Errorf("invalid response")
	}

	sw1, sw2 := rsp[len(rsp)-2], rsp[len(rsp)-1]

	// Handle GET RESPONSE if needed
	if sw1 == 0x61 {
		// sw2 contains the length of data available
		getResponseCmd := []byte{0x00, 0xC0, 0x00, 0x00, sw2}
		rsp, err = card.Transmit(getResponseCmd)
		if err != nil {
			return fmt.Errorf("GET RESPONSE failed: %w", err)
		}

		if len(rsp) < 2 {
			return fmt.Errorf("invalid GET RESPONSE")
		}

		sw1, sw2 = rsp[len(rsp)-2], rsp[len(rsp)-1]
	}

	// Accept multiple success status codes
	if (sw1 == 0x90 && sw2 == 0x00) || (sw1 == 0x97 && sw2 == 0x10) {
		return nil
	}

	// 6A82 means file/application not found - might need to reset card
	if sw1 == 0x6A && sw2 == 0x82 {
		return fmt.Errorf("applet not found (SW=%02X%02X) - card may need reset", sw1, sw2)
	}

	return fmt.Errorf("select applet failed: SW=%02X%02X", sw1, sw2)
}

func (r *PCSCReader) readBinary(card *scard.Card, p1, p2, le byte) ([]byte, error) {
	// Send READ BINARY command for Thai ID card
	cmd := []byte{0x80, 0xB0, p1, p2, 0x02, 0x00, le}

	rsp, err := card.Transmit(cmd)
	if err != nil {
		return nil, err
	}

	if len(rsp) < 2 {
		return nil, fmt.Errorf("invalid response")
	}

	sw1, sw2 := rsp[len(rsp)-2], rsp[len(rsp)-1]

	// Check if we need to GET RESPONSE
	if sw1 == 0x61 {
		// sw2 contains the length of data available
		getResponseCmd := []byte{0x00, 0xC0, 0x00, 0x00, sw2}
		rsp, err = card.Transmit(getResponseCmd)
		if err != nil {
			return nil, err
		}

		if len(rsp) < 2 {
			return nil, fmt.Errorf("invalid GET RESPONSE")
		}

		sw1, sw2 = rsp[len(rsp)-2], rsp[len(rsp)-1]
	}

	if sw1 != 0x90 || sw2 != 0x00 {
		return nil, fmt.Errorf("read binary failed: SW=%02X%02X", sw1, sw2)
	}

	return rsp[:len(rsp)-2], nil
}

func (r *PCSCReader) readPhoto(card *scard.Card) ([]byte, error) {
	var photoData []byte

	// Photo is split into 20 parts
	photoCommands := []struct{ p1, p2 byte }{
		{0x01, 0x7B}, {0x02, 0x7A}, {0x03, 0x79}, {0x04, 0x78}, {0x05, 0x77},
		{0x06, 0x76}, {0x07, 0x75}, {0x08, 0x74}, {0x09, 0x73}, {0x0A, 0x72},
		{0x0B, 0x71}, {0x0C, 0x70}, {0x0D, 0x6F}, {0x0E, 0x6E}, {0x0F, 0x6D},
		{0x10, 0x6C}, {0x11, 0x6B}, {0x12, 0x6A}, {0x13, 0x69}, {0x14, 0x68},
	}

	for _, cmd := range photoCommands {
		data, err := r.readBinary(card, cmd.p1, cmd.p2, 0xFF)
		if err != nil {
			// Some cards might not have all photo parts
			break
		}
		photoData = append(photoData, data...)
	}

	// Find the end of JPEG data (FFD9 marker) and trim padding
	jpegEnd := bytes.Index(photoData, []byte{0xFF, 0xD9})
	if jpegEnd != -1 {
		// Include the FFD9 marker
		photoData = photoData[:jpegEnd+2]
	} else {
		// If no JPEG end marker found, trim trailing spaces (0x20)
		photoData = bytes.TrimRight(photoData, " ")
	}

	return photoData, nil
}

func (r *PCSCReader) decodeThaiString(data []byte) string {
	// Thai ID cards use TIS-620 encoding
	decoder := charmap.Windows874.NewDecoder()
	decoded, err := decoder.Bytes(data)
	if err != nil {
		// Fallback to original if decoding fails
		return string(bytes.Trim(data, "\x00"))
	}
	return string(bytes.Trim(decoded, "\x00"))
}

func (r *PCSCReader) formatDate(dateStr string) string {
	dateStr = string(bytes.Trim([]byte(dateStr), "\x00"))
	if len(dateStr) < 8 {
		return ""
	}

	year := dateStr[0:4]
	month := dateStr[4:6]
	day := dateStr[6:8]

	// Convert Buddhist Era to Gregorian
	var thaiYear int
	_, _ = fmt.Sscanf(year, "%d", &thaiYear)
	gregorianYear := thaiYear - 543

	return fmt.Sprintf("%04d-%s-%s", gregorianYear, month, day)
}
