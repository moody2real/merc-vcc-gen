# pkg/fluz

Reusable client for the official **Fluz GraphQL API** (virtual cards). Pure HTTP —
no browser, no session scraping. Mirrors the shape of `pkg/merc`.

Endpoint: `https://transactional-graph.fluzapp.com/api/v1/graphql` (production).

## Auth

Two-tier, handled for you:
1. Your **API key** (`Authorization: Basic <APIKey>`) mints a short-lived user
   access token via `generateUserAccessToken`.
2. That **Bearer token** authorizes all card operations. The client caches it and
   auto-regenerates on expiry/401.

Credentials come from the Fluz Developer Console (`uni.fluzapp.com/developers`):
API Key, User ID, Account ID.

## Usage

```go
import "merc-vcc-gen/pkg/fluz"

c := fluz.New(fluz.Config{
    APIKey:    "<base64 api key>",
    UserID:    "0a033ad7-...",
    AccountID: "25dc4479-...",
    // SeatID, Endpoint, Scopes are optional
})

// Balance / funding sources
w, _ := c.Balance()
fmt.Println("rewards:", w.Balances.RewardsBalance.AvailableBalance)
banks, _ := c.FundingSources()          // []BankAccount with BankAccountID

// Pick an offer (Fluz cards are created against a merchant offer)
offerID, _ := c.OfferQuote("amazon", 25) // merchantSlug, denomination

// Create (async: submit order, poll until COMPLETED)
order, _ := c.CreateAndWait(fluz.CreateRequest{
    OfferID: offerID,
    Items: []fluz.OrderItem{{
        Quantity:             1,
        SpendLimit:           25,
        SpendLimitDuration:   fluz.DurationDaily,
        PrimaryFundingSource: fluz.FundingBankAccount,
        BankAccountID:        banks[0].BankAccountID,
        CardNickname:         "my card",
    }},
}, 90*time.Second)
for _, card := range order.VirtualCards {
    fmt.Println(card.Line()) // number,MM/YY,CVV
}

// Reveal an existing card's PAN/CVV
d, _ := c.Reveal("<virtualCardId>")
fmt.Println(d.Line())        // number,MM/YY,CVV,ZIP

// Manage
c.Lock("<virtualCardId>")
c.Unlock("<virtualCardId>")
c.Edit(fluz.EditRequest{VirtualCardID: "<id>", CardNickname: "renamed", SpendLimit: 50})
```

## API

- `New(cfg Config) *Client`
- `EnsureToken() error`
- `Balance() (*Wallet, error)` — rewards / cash balances, userCashBalances, bankAccounts
- `FundingSources() ([]BankAccount, error)`
- `OfferQuote(merchantSlug string, denomination float64) (offerId string, err error)`
- `CreateCards(CreateRequest) (orderId, orderStatus string, err error)`
- `OrderStatus(orderId string) (*Order, error)`
- `CreateAndWait(CreateRequest, timeout) (*Order, error)` — create + poll to COMPLETED
- `Reveal(virtualCardId string) (*CardDetails, error)`
- `Lock(virtualCardId) / Unlock(virtualCardId) error`
- `Edit(EditRequest) error`

## Notes (verified live against production)

- Amounts are **strings, 5-decimal** (e.g. `"5.00000"`).
- Offers have a **minimum denomination** (~$5 for Amazon); below it `OfferQuote`
  returns an empty offerId.
- Default scopes: `LIST_PAYMENT, LIST_PURCHASES, LIST_OFFERS, CREATE_VIRTUALCARD,
  REVEAL_VIRTUALCARD, MANAGE_PAYMENT, EDIT_VIRTUALCARD`. `PCI_COMPLIANCE` is a
  Fluz-gated scope your app is not granted; it is **not** required for reveal
  (reveal works with `REVEAL_VIRTUALCARD` alone). Override via `Config.Scopes`.
- `Create` is the only operation that can move money (it draws from the chosen
  funding source). Everything else (balance/reveal/lock/unlock/edit) is free.
- **Create status on this account:** `createVirtualCardBulkOrder` currently
  returns server error `VC-0019 UnableToGetVirtualCards` ("unable to get your
  virtual card history") for every input/funding/seat combination — a backend
  gate, not a client bug. All other operations are verified working live. Fluz
  support must enable card issuance (likely tied to the Fluz-gated
  `PCI_COMPLIANCE` scope) on the app before create succeeds.
- There is no "list all my virtual cards" endpoint in the API; track
  `virtualCardId`s returned from creation.
