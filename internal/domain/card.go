package domain

import "strings"

type Address struct {
	HouseNo     string `json:"houseNo"`
	Moo         string `json:"moo"`
	Soi         string `json:"soi"`
	Street      string `json:"street"`
	Subdistrict string `json:"subdistrict"`
	District    string `json:"district"`
	Province    string `json:"province"`
	FullAddress string `json:"fullAddress"`
}

type ThaiIdCard struct {
	CitizenID    string   `json:"citizenId"`
	PrefixNameTH string   `json:"prefixNameTh"`
	FirstNameTH  string   `json:"firstNameTh"`
	MiddleNameTH string   `json:"middleNameTh"`
	LastNameTH   string   `json:"lastNameTh"`
	PrefixNameEN string   `json:"prefixNameEN"`
	FirstNameEN  string   `json:"firstNameEn"`
	MiddleNameEN string   `json:"middleNameEN"`
	LastNameEN   string   `json:"lastNameEn"`
	DateOfBirth  string   `json:"dateOfBirth"`
	Gender       string   `json:"gender"`
	Address      *Address `json:"address"`
	IssueDate    string   `json:"issueDate"`
	ExpireDate   string   `json:"expireDate"`
	PhotoBase64  string   `json:"photoBase64"`
}

type CardReaderService interface {
	StartMonitoring() error
	StopMonitoring()
	OnCardInserted(handler func(card *ThaiIdCard, err error))
	OnCardRemoved(handler func())
}

// ParseThaiAddress parses a Thai address string into structured format
func ParseThaiAddress(addressStr string) *Address {
	if addressStr == "" {
		return nil
	}

	parts := strings.Split(addressStr, "#")
	if len(parts) == 0 {
		return &Address{FullAddress: addressStr}
	}

	addr := &Address{}

	// Extract house number from first part
	if len(parts) > 0 && parts[0] != "" {
		addr.HouseNo = strings.TrimSpace(parts[0])
	}

	// Extract province from last part first (may or may not have prefix)
	if len(parts) > 1 {
		lastPart := strings.TrimSpace(parts[len(parts)-1])
		if lastPart != "" {
			if strings.HasPrefix(lastPart, "จังหวัด") {
				addr.Province = strings.TrimSpace(strings.TrimPrefix(lastPart, "จังหวัด"))
			} else {
				// Assume last part is province even without prefix
				addr.Province = lastPart
			}
		}
	}

	// Process middle parts (skip first and last)
	endIdx := len(parts) - 1
	if endIdx < 1 {
		endIdx = len(parts)
	}

	for i := 1; i < endIdx; i++ {
		part := strings.TrimSpace(parts[i])
		if part == "" {
			continue
		}

		// Check for Moo (village)
		if strings.HasPrefix(part, "หมู่ที่") {
			addr.Moo = strings.TrimSpace(strings.TrimPrefix(part, "หมู่ที่"))
		} else if strings.HasPrefix(part, "ซอย") {
			// Check for Soi (alley)
			addr.Soi = strings.TrimSpace(strings.TrimPrefix(part, "ซอย"))
		} else if strings.HasPrefix(part, "ตำบล") || strings.HasPrefix(part, "แขวง") {
			// Check for Subdistrict
			addr.Subdistrict = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(part, "ตำบล"), "แขวง"))
		} else if strings.HasPrefix(part, "อำเภอ") || strings.HasPrefix(part, "เขต") {
			// Check for District
			addr.District = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(part, "อำเภอ"), "เขต"))
		} else if strings.HasPrefix(part, "จังหวัด") {
			// If province appears in middle parts with prefix, override the last part
			addr.Province = strings.TrimSpace(strings.TrimPrefix(part, "จังหวัด"))
		} else if addr.Street == "" {
			// If no prefix, assume it's a street name
			addr.Street = part
		}
	}

	// Build full address
	var fullAddressParts []string
	if addr.HouseNo != "" {
		fullAddressParts = append(fullAddressParts, addr.HouseNo)
	}
	if addr.Moo != "" {
		fullAddressParts = append(fullAddressParts, "หมู่ที่ "+addr.Moo)
	}
	if addr.Soi != "" {
		fullAddressParts = append(fullAddressParts, "ซอย"+addr.Soi)
	}
	if addr.Street != "" {
		fullAddressParts = append(fullAddressParts, addr.Street)
	}
	if addr.Subdistrict != "" {
		prefix := "ตำบล"
		if strings.Contains(addressStr, "แขวง") {
			prefix = "แขวง"
		}
		fullAddressParts = append(fullAddressParts, prefix+addr.Subdistrict)
	}
	if addr.District != "" {
		prefix := "อำเภอ"
		if strings.Contains(addressStr, "เขต") {
			prefix = "เขต"
		}
		fullAddressParts = append(fullAddressParts, prefix+addr.District)
	}
	if addr.Province != "" {
		fullAddressParts = append(fullAddressParts, "จังหวัด"+addr.Province)
	}

	addr.FullAddress = strings.Join(fullAddressParts, " ")
	return addr
}
