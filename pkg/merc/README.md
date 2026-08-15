# pkg/merc

Reusable Mercury virtual-card client. Handles login/session, listing, creating
(issue + reveal), deleting, and balance — with no CLI or UI dependencies.

## Import from another module

This module is named `merc-vcc-gen`. In your other project's `go.mod`:

```
require merc-vcc-gen v0.0.0

replace merc-vcc-gen => C:/Users/Moody/Desktop/merc-vcc-gen
```

Then:

```go
import "merc-vcc-gen/pkg/merc"
```

## Usage

```go
package main

import (
	"fmt"
	"log"

	"merc-vcc-gen/pkg/merc"
)

func main() {
	cfg, err := merc.LoadConfig("config.json")
	if err != nil {
		log.Fatal(err)
	}

	client := merc.New(cfg, "session.json")

	// Reuses session.json if valid, otherwise logs in via headless Chrome
	// (Chrome is auto-installed if not found) and saves a fresh session.
	if err := client.EnsureSession(); err != nil {
		log.Fatal(err)
	}

	// Balance
	balances, err := client.Balance()
	if err != nil {
		log.Fatal(err)
	}
	for _, b := range balances {
		fmt.Printf("%s: available $%.2f, current $%.2f\n", b.Name, b.AvailableBalance, b.CurrentBalance)
	}

	// Create one card (issue + reveal PAN/exp/CVV)
	card, details, err := client.Create()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("created:", card.Last4Digits, details.Line())

	// List active cards
	active, _ := client.ActiveCards()
	fmt.Println("active cards:", len(active))

	// Delete one by id
	_ = client.Cancel(card.ID)
}
```

## API

Config
- `LoadConfig(path string) (*Config, error)` — load `config.json`.

Session
- `New(cfg *Config, sessionPath string) *Client`
- `EnsureSession() error` — reuse a saved session, else log in and save.
- `Login() error` — force a fresh browser login.

Cards
- `Balance() ([]AccountBalance, error)`
- `List() ([]Card, error)`
- `ActiveCards() ([]Card, error)`
- `Issue() (Card, error)`
- `Reveal(id string) (CardDetails, error)`
- `Create() (Card, CardDetails, error)` — issue + reveal in one call.
- `Cancel(id string) error`
- `DeleteAllActive() (int, error)`

`CardDetails.Line()` returns `PAN,MM/YY,CVV,ZIP`.

## Notes

- All card operations return an error until `EnsureSession`/`Login` has succeeded.
- Login uses the proxy and browser settings from `config.json`; Chrome is
  located automatically (or downloaded) and stale Chrome instances holding the
  profile are killed first.
- The internal `charmbracelet/log` logger still emits at its configured level;
  set the global level in your app if you want it quieter.
