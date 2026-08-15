package fluz

import (
	"fmt"
	"time"
)

func (c *Client) Balance() (*Wallet, error) {
	const q = `query { getWallet { balances { rewardsBalance { availableBalance totalBalance lifetimeBalance } cashBalance { availableBalance totalBalance pendingBalance lifetimeBalance } userCashBalances(paginate: { limit: 20, offset: 0 }) { userCashBalanceId totalCashBalance availableCashBalance lifetimeCashBalance nickname status createdAt } } bankAccounts { bankAccountId type accountName status lastFour nickname } } }`
	var out struct {
		GetWallet Wallet `json:"getWallet"`
	}
	if err := c.gql(q, nil, &out); err != nil {
		return nil, err
	}
	return &out.GetWallet, nil
}

func (c *Client) FundingSources() ([]BankAccount, error) {
	w, err := c.Balance()
	if err != nil {
		return nil, err
	}
	return w.BankAccounts, nil
}

func (c *Client) OfferQuote(merchantSlug string, denomination float64) (string, error) {
	const q = `query($input: GetOfferQuoteInput!) { getOfferQuote(input: $input) { offerId } }`
	vars := map[string]any{"input": map[string]any{"merchantSlug": merchantSlug, "denomination": denomination}}
	var out struct {
		GetOfferQuote struct {
			OfferID string `json:"offerId"`
		} `json:"getOfferQuote"`
	}
	if err := c.gql(q, vars, &out); err != nil {
		return "", err
	}
	return out.GetOfferQuote.OfferID, nil
}

func (c *Client) CreateCards(req CreateRequest) (string, string, error) {
	const q = `mutation($input: CreateVirtualCardBulkOrderInput!) { createVirtualCardBulkOrder(input: $input) { orderId orderStatus } }`

	items := make([]map[string]any, 0, len(req.Items))
	for _, it := range req.Items {
		m := map[string]any{"quantity": it.Quantity, "spendLimit": it.SpendLimit}
		if it.SpendLimitDuration != "" {
			m["spendLimitDuration"] = it.SpendLimitDuration
		}
		if it.PrimaryFundingSource != "" {
			m["primaryFundingSource"] = it.PrimaryFundingSource
		}
		if it.BankAccountID != "" {
			m["bankAccountId"] = it.BankAccountID
		}
		if it.CardNickname != "" {
			m["cardNickname"] = it.CardNickname
		}
		if it.LockCardNextUse {
			m["lockCardNextUse"] = true
		}
		if it.LockDate != "" {
			m["lockDate"] = it.LockDate
		}
		items = append(items, m)
	}

	vars := map[string]any{"input": map[string]any{"offerId": req.OfferID, "orderItems": items}}
	var out struct {
		CreateVirtualCardBulkOrder struct {
			OrderID     string `json:"orderId"`
			OrderStatus string `json:"orderStatus"`
		} `json:"createVirtualCardBulkOrder"`
	}
	if err := c.gql(q, vars, &out); err != nil {
		return "", "", err
	}
	return out.CreateVirtualCardBulkOrder.OrderID, out.CreateVirtualCardBulkOrder.OrderStatus, nil
}

func (c *Client) OrderStatus(orderID string) (*Order, error) {
	const q = `query($orderId: String!) { getVirtualCardBulkOrderStatus(input: { orderId: $orderId }) { orderStatus orderId virtualCards { virtualCardId cardNumber expiryMMYY cvv cardHolderName } successfulCardCreations failedCardCreations totalCards } }`
	var out struct {
		Get Order `json:"getVirtualCardBulkOrderStatus"`
	}
	if err := c.gql(q, map[string]any{"orderId": orderID}, &out); err != nil {
		return nil, err
	}
	return &out.Get, nil
}

func (c *Client) CreateAndWait(req CreateRequest, timeout time.Duration) (*Order, error) {
	orderID, _, err := c.CreateCards(req)
	if err != nil {
		return nil, err
	}

	deadline := time.Now().Add(timeout)
	for {
		order, err := c.OrderStatus(orderID)
		if err != nil {
			return nil, err
		}
		switch order.OrderStatus {
		case "COMPLETED":
			return order, nil
		case "FAILED":
			return order, fmt.Errorf("order %s failed", orderID)
		}
		if time.Now().After(deadline) {
			return order, fmt.Errorf("order %s still %s after timeout", orderID, order.OrderStatus)
		}
		time.Sleep(3 * time.Second)
	}
}

func (c *Client) Reveal(virtualCardID string) (*CardDetails, error) {
	const q = `mutation($id: UUID!) { revealVirtualCardByVirtualCardId(virtualCardId: $id) { cardNumber expiryMMYY cvv cardHolderName billingAddress { streetAddress city state postalCode } } }`
	var out struct {
		Reveal CardDetails `json:"revealVirtualCardByVirtualCardId"`
	}
	if err := c.gql(q, map[string]any{"id": virtualCardID}, &out); err != nil {
		return nil, err
	}
	out.Reveal.VirtualCardID = virtualCardID
	return &out.Reveal, nil
}

func (c *Client) Lock(virtualCardID string) error {
	const q = `mutation($input: LockVirtualCardInput!) { lockVirtualCard(input: $input) { __typename } }`
	return c.gql(q, map[string]any{"input": map[string]any{"virtualCardId": virtualCardID}}, nil)
}

func (c *Client) Unlock(virtualCardID string) error {
	const q = `mutation($input: UnlockVirtualCardInput!) { unlockVirtualCard(input: $input) { __typename } }`
	return c.gql(q, map[string]any{"input": map[string]any{"virtualCardId": virtualCardID}}, nil)
}

func (c *Client) Edit(req EditRequest) error {
	const q = `mutation($input: EditVirtualCardInput!) { editVirtualCard(input: $input) { virtualCardId status } }`
	in := map[string]any{"virtualCardId": req.VirtualCardID}
	if req.CardNickname != "" {
		in["cardNickname"] = req.CardNickname
	}
	if req.SpendLimit > 0 {
		in["spendLimit"] = req.SpendLimit
	}
	if req.SpendLimitDuration != "" {
		in["spendLimitDuration"] = req.SpendLimitDuration
	}
	if req.LockDate != "" {
		in["lockDate"] = req.LockDate
	}
	return c.gql(q, map[string]any{"input": in}, nil)
}
