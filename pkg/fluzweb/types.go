package fluzweb

const (
	DefaultEndpoint = "https://fluz.app"
	DefaultChannel  = "UWP"
)

type DeviceDetails struct {
	DeviceID        string `json:"deviceId"`
	Interface       string `json:"interface"`
	MacAddress      string `json:"macAddress"`
	OS              string `json:"os"`
	OSVersion       string `json:"osVersion"`
	SoftwareVersion string `json:"softwareVersion"`
	Type            string `json:"type"`
	Brand           string `json:"brand"`
	Model           string `json:"model"`
}

func defaultDevice() DeviceDetails {
	return DeviceDetails{
		DeviceID:   "v99mp1y9DB5ubPQv0woB",
		Interface:  "BROWSER",
		MacAddress: "0:0:0:0:0",
		OS:         "Windows",
		OSVersion:  "11",
		Type:       "DESKTOP",
		Brand:      "Brave",
		Model:      "150.0.0",
	}
}

type CreateParams struct {
	SeatID            string
	OfferID           string
	PurchaseAmount    int
	FluzpayAmount     int
	BankAccountID     string
	AddressID         string
	UserCashBalanceID string
	CardNickname      string
	LockCardNextUse   bool
	Channel           string
	Device            *DeviceDetails
}

type WebCard struct {
	VirtualCardID  string
	Last4          string
	Status         string
	InitialBalance int
	ExpiresAt      string
	CardholderName string
	OrderNumber    string
}
