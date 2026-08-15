package fluzweb

import "fmt"

func (c *Client) CheckPin(pin string) (pinAuthToken, encryptedPin string, err error) {
	stream, err := c.post("/check-pin.data", map[string]any{"pin": pin})
	if err != nil {
		return "", "", err
	}
	pinAuthToken = extractString(stream, "token")
	encryptedPin = extractString(stream, "encryptedPin")
	if pinAuthToken == "" || encryptedPin == "" {
		return "", "", fmt.Errorf("check-pin failed: %s", extractString(stream, "msg"))
	}
	return pinAuthToken, encryptedPin, nil
}

func (c *Client) Create(p CreateParams) (*WebCard, error) {
	pinAuthToken, encryptedPin, err := c.CheckPin(c.pin)
	if err != nil {
		return nil, err
	}

	channel := p.Channel
	if channel == "" {
		channel = DefaultChannel
	}
	device := defaultDevice()
	if p.Device != nil {
		device = *p.Device
	}

	body := map[string]any{
		"seat_id":              p.SeatID,
		"offer_id":             p.OfferID,
		"purchase_amount":      p.PurchaseAmount,
		"fluzpay_amount":       p.FluzpayAmount,
		"bank_account_id":      p.BankAccountID,
		"channel":              channel,
		"addressId":            p.AddressID,
		"isMultiUseVCPurchase": true,
		"lockCardNextUse":      p.LockCardNextUse,
		"cardNickname":         p.CardNickname,
		"pinAuthToken":         pinAuthToken,
		"deviceDetails":        device,
		"brandLocked":          false,
		"user_cash_balance_id": p.UserCashBalanceID,
		"virtualCardPIN":       encryptedPin,
		"fluzpay_options":      map[string]any{"use_prepayment_balance": true, "use_rewards_balance": false},
		"isTokenized":          true,
		"expenseCategory":      "",
		"expenseMemo":          "",
	}

	stream, err := c.post("/virtual-cards/create.data", body)
	if err != nil {
		return nil, err
	}

	id := extractString(stream, "virtual_card_id")
	if id == "" {
		return nil, fmt.Errorf("create failed: %s", extractString(stream, "msg"))
	}
	return &WebCard{
		VirtualCardID:  id,
		Last4:          extractString(stream, "virtual_card_last_4"),
		Status:         extractString(stream, "status"),
		InitialBalance: extractInt(stream, "initial_card_balance"),
		ExpiresAt:      extractString(stream, "expires_at"),
		CardholderName: extractString(stream, "cardholder_name"),
		OrderNumber:    extractString(stream, "order_number"),
	}, nil
}
