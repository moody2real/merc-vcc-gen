package main

import (
	"fmt"
	"os"

	"github.com/moody2real/merc-vcc-gen/internal/config"
	"github.com/moody2real/merc-vcc-gen/pkg/fluz"
	"github.com/moody2real/merc-vcc-gen/pkg/fluzweb"
)

func main() {
	cfg, err := config.Load("config.json")
	if err != nil {
		fmt.Println("config:", err)
		return
	}

	cookieFile := os.Getenv("FLUZ_COOKIE_FILE")
	if cookieFile == "" {
		cookieFile = "/tmp/fluz_cookie.txt"
	}
	cookie, err := os.ReadFile(cookieFile)
	if err != nil {
		fmt.Println("read cookie:", err)
		return
	}

	web := fluzweb.New(fluzweb.Config{
		Cookie:      string(cookie),
		PIN:         os.Getenv("FLUZ_PIN"),
		ProxyURL:    cfg.Proxy.URL(),
		SessionPath: "fluz_session.json",
	})

	before := web.Cookie()
	if err := web.Heartbeat(); err != nil {
		fmt.Println("[FAIL] Heartbeat:", err)
	} else {
		fmt.Printf("[ ok ] Heartbeat: __session rotated=%v, saved to fluz_session.json\n", before != web.Cookie())
	}

	if os.Getenv("FLUZ_CREATE") != "1" {
		fmt.Println("skipping create (set FLUZ_CREATE=1 to create a real card)")
		return
	}

	fmt.Println("creating card through proxy...")
	card, err := web.Create(fluzweb.CreateParams{
		SeatID:            "dc0e84b9-0ccc-41ac-9657-f5ca45a82247",
		OfferID:           "26a63dc0-74a8-46d9-960b-2f6f373efab5",
		PurchaseAmount:    500,
		FluzpayAmount:     1,
		BankAccountID:     "da619fdd-8888-427a-a331-cccb25dddfbd",
		AddressID:         "12b9ac94-58b7-4dce-90b5-034391836a5a",
		UserCashBalanceID: "edf14c92-4ede-4e5c-848f-e51c11685a3f",
		CardNickname:      "web-mod-test",
		LockCardNextUse:   true,
	})
	if err != nil {
		fmt.Println("[FAIL] Create:", err)
		return
	}
	fmt.Printf("[ ok ] Create: id=%s last4=%s status=%s balance=%d\n",
		card.VirtualCardID, card.Last4, card.Status, card.InitialBalance)

	api := fluz.New(fluz.Config{
		APIKey:    os.Getenv("FLUZ_API_KEY"),
		UserID:    "0a033ad7-89f2-4785-973c-3505caf8f756",
		AccountID: "25dc4479-a2c0-4ea3-bb6e-d3c085613162",
	})
	d, err := api.Reveal(card.VirtualCardID)
	if err != nil {
		fmt.Println("[FAIL] Reveal:", err)
		return
	}
	fmt.Printf("[ ok ] Reveal: %s\n", d.Line())
	fmt.Println("done.")
}
