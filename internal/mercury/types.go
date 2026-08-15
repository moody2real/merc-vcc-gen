package mercury

type Status struct {
	Tag string `json:"tag"`
}

type Card struct {
	ID               string `json:"id"`
	AccountID        string `json:"accountId"`
	UserID           string `json:"userId"`
	Nickname         string `json:"nickname"`
	Last4Digits      string `json:"last4Digits"`
	CardType         string `json:"cardType"`
	CardRealm        string `json:"cardRealm"`
	FullStatus       Status `json:"fullStatus"`
	CreatedAtDateStr string `json:"createdAtDateStr"`
}

type cardEntry struct {
	Tag      string `json:"tag"`
	Contents Card   `json:"contents"`
}

type listResponse struct {
	Cards []cardEntry `json:"cards"`
}

type cardLimitContents struct {
	TransactionLimit int `json:"transactionLimit"`
}

type cardLimits struct {
	Tag      string            `json:"tag"`
	Contents cardLimitContents `json:"contents"`
}

type issuingAddress struct {
	Tag string `json:"tag"`
}

type issueRequest struct {
	CardRealm      string         `json:"cardRealm"`
	CardLimits     cardLimits     `json:"cardLimits"`
	AccountID      string         `json:"accountId"`
	UserID         string         `json:"userId"`
	Nickname       string         `json:"nickname"`
	IssuingAddress issuingAddress `json:"issuingAddress"`
	CategoryLocks  []string       `json:"categoryLocks"`
}

type issueResponse struct {
	DebitCardID   string `json:"debitCardId"`
	DebitCardData Card   `json:"debitCardData"`
}

type revealResponse struct {
	PresignedURL string `json:"presigned_url"`
}

type initialData struct {
	OrgSpecifics struct {
		GlobalData struct {
			MercuryDepositoryAccounts []struct {
				ID               string  `json:"id"`
				Name             string  `json:"name"`
				Nickname         string  `json:"nickname"`
				AvailableBalance float64 `json:"availableBalance"`
				CurrentBalance   float64 `json:"currentBalance"`
			} `json:"mercuryDepositoryAccounts"`
		} `json:"globalData"`
	} `json:"orgSpecifics"`
	UserDetails struct {
		ID string `json:"id"`
	} `json:"userDetails"`
}

type AccountBalance struct {
	Name             string
	Nickname         string
	AvailableBalance float64
	CurrentBalance   float64
}

type CardDetails struct {
	PAN    string
	Expiry string
	CVV    string
	Zip    string
}

func (d CardDetails) Line() string {
	return d.PAN + "," + d.Expiry + "," + d.CVV + "," + d.Zip
}
