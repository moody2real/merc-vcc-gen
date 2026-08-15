package main

import (
	"fmt"
	"os"
	"time"

	"github.com/moody2real/merc-vcc-gen/pkg/fluz"
)

func main() {
	c := fluz.New(fluz.Config{
		APIKey:    os.Getenv("FLUZ_API_KEY"),
		UserID:    os.Getenv("FLUZ_USER_ID"),
		AccountID: os.Getenv("FLUZ_ACCOUNT_ID"),
	})

	step := func(name string, err error) {
		if err != nil {
			fmt.Printf("[FAIL] %s: %v\n", name, err)
		} else {
			fmt.Printf("[ ok ] %s\n", name)
		}
	}

	step("EnsureToken", c.EnsureToken())

	w, err := c.Balance()
	step("Balance", err)
	if w != nil {
		fmt.Printf("       rewards=%s cash=%s bankAccounts=%d userCashBalances=%d\n",
			w.Balances.RewardsBalance.AvailableBalance, w.Balances.CashBalance.AvailableBalance,
			len(w.BankAccounts), len(w.Balances.UserCashBalances))
	}

	revealID := os.Getenv("FLUZ_CARD_ID")
	if revealID != "" {
		d, err := c.Reveal(revealID)
		step("Reveal(existing)", err)
		if err == nil {
			fmt.Printf("       ...%s exp=%s holder=%s\n", last4(d.CardNumber), d.ExpiryMMYY, d.CardHolderName)
		}
		step("Lock", c.Lock(revealID))
		step("Unlock", c.Unlock(revealID))
	}

	if os.Getenv("FLUZ_CREATE") != "1" {
		fmt.Println("skipping create (set FLUZ_CREATE=1 to run a real, possibly-charged create)")
		fmt.Println("done.")
		return
	}

	var bankID string
	for _, b := range w.BankAccounts {
		if b.Type == "CHECKING" {
			bankID = b.BankAccountID
			break
		}
	}
	offerID, err := c.OfferQuote("amazon", 5)
	step("OfferQuote", err)

	order, err := c.CreateAndWait(fluz.CreateRequest{
		OfferID: offerID,
		Items: []fluz.OrderItem{{
			Quantity:             1,
			SpendLimit:           5,
			SpendLimitDuration:   fluz.DurationDaily,
			PrimaryFundingSource: fluz.FundingBankAccount,
			BankAccountID:        bankID,
			CardNickname:         "smoke-test",
			LockCardNextUse:      true,
		}},
	}, 90*time.Second)
	step("CreateAndWait", err)
	if order != nil {
		fmt.Printf("       status=%s created=%d/%d\n", order.OrderStatus, order.Successful, order.Total)
		for _, card := range order.VirtualCards {
			fmt.Printf("       CARD id=%s ...%s exp=%s cvv=%s\n", card.VirtualCardID, last4(card.CardNumber), card.ExpiryMMYY, card.CVV)
		}
	}
	fmt.Println("done.")
}

func last4(s string) string {
	if len(s) < 4 {
		return s
	}
	return s[len(s)-4:]
}
