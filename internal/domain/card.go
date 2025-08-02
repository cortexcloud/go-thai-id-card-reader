package domain

type ThaiIdCard struct {
	CitizenID   string `json:"citizenId"`
	FirstNameTH string `json:"firstNameTh"`
	LastNameTH  string `json:"lastNameTh"`
	FirstNameEN string `json:"firstNameEn"`
	LastNameEN  string `json:"lastNameEn"`
	DateOfBirth string `json:"dateOfBirth"`
	Gender      string `json:"gender"`
	Address     string `json:"address"`
	IssueDate   string `json:"issueDate"`
	ExpireDate  string `json:"expireDate"`
	PhotoBase64 string `json:"photoBase64"`
}

type CardReaderService interface {
	StartMonitoring() error
	StopMonitoring()
	OnCardInserted(handler func(card *ThaiIdCard, err error))
	OnCardRemoved(handler func())
}