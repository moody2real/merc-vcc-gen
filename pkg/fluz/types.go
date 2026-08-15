package fluz

const (
	FundingBankAccount = "BANK_ACCOUNT"
	FundingFluzBalance = "FLUZ_BALANCE"

	DurationDaily    = "DAILY"
	DurationWeekly   = "WEEKLY"
	DurationMonthly  = "MONTHLY"
	DurationAnnual   = "ANNUAL"
	DurationLifetime = "LIFETIME"
)

type Balance struct {
	AvailableBalance string `json:"availableBalance"`
	TotalBalance     string `json:"totalBalance"`
	PendingBalance   string `json:"pendingBalance"`
	LifetimeBalance  string `json:"lifetimeBalance"`
}

type UserCashBalance struct {
	UserCashBalanceID    string `json:"userCashBalanceId"`
	TotalCashBalance     string `json:"totalCashBalance"`
	AvailableCashBalance string `json:"availableCashBalance"`
	LifetimeCashBalance  string `json:"lifetimeCashBalance"`
	Nickname             string `json:"nickname"`
	Status               string `json:"status"`
	CreatedAt            string `json:"createdAt"`
}

type BankAccount struct {
	BankAccountID string `json:"bankAccountId"`
	Type          string `json:"type"`
	AccountName   string `json:"accountName"`
	Status        string `json:"status"`
	LastFour      string `json:"lastFour"`
	Nickname      string `json:"nickname"`
}

type Wallet struct {
	Balances struct {
		RewardsBalance   Balance           `json:"rewardsBalance"`
		CashBalance      Balance           `json:"cashBalance"`
		UserCashBalances []UserCashBalance `json:"userCashBalances"`
	} `json:"balances"`
	BankAccounts []BankAccount `json:"bankAccounts"`
}

type Address struct {
	StreetAddress string `json:"streetAddress"`
	City          string `json:"city"`
	State         string `json:"state"`
	PostalCode    string `json:"postalCode"`
}

type Card struct {
	VirtualCardID  string `json:"virtualCardId"`
	CardNumber     string `json:"cardNumber"`
	ExpiryMMYY     string `json:"expiryMMYY"`
	CVV            string `json:"cvv"`
	CardHolderName string `json:"cardHolderName"`
}

func (c Card) Line() string {
	return c.CardNumber + "," + c.ExpiryMMYY + "," + c.CVV
}

type CardDetails struct {
	VirtualCardID  string  `json:"-"`
	CardNumber     string  `json:"cardNumber"`
	ExpiryMMYY     string  `json:"expiryMMYY"`
	CVV            string  `json:"cvv"`
	CardHolderName string  `json:"cardHolderName"`
	BillingAddress Address `json:"billingAddress"`
}

func (d CardDetails) Line() string {
	return d.CardNumber + "," + d.ExpiryMMYY + "," + d.CVV + "," + d.BillingAddress.PostalCode
}

type Order struct {
	OrderID      string `json:"orderId"`
	OrderStatus  string `json:"orderStatus"`
	VirtualCards []Card `json:"virtualCards"`
	Successful   int    `json:"successfulCardCreations"`
	Failed       int    `json:"failedCardCreations"`
	Total        int    `json:"totalCards"`
}

type OrderItem struct {
	Quantity             int
	SpendLimit           float64
	SpendLimitDuration   string
	PrimaryFundingSource string
	BankAccountID        string
	CardNickname         string
	LockCardNextUse      bool
	LockDate             string
}

type CreateRequest struct {
	OfferID string
	Items   []OrderItem
}

type EditRequest struct {
	VirtualCardID      string
	CardNickname       string
	SpendLimit         float64
	SpendLimitDuration string
	LockDate           string
}
