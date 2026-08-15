# pkg/fluzweb

Creates Fluz virtual cards through the **consumer web endpoints** (the same calls
`fluz.app` makes), for when the official API's `createVirtualCardBulkOrder` is
blocked (`VC-0019`). Pairs with `pkg/fluz` for reveal/balance/lock via the API.

## Why this exists

The official API (`pkg/fluz`) works for token, balance, reveal, lock, unlock,
edit — but `create` returns a server-side `VC-0019 UnableToGetVirtualCards` for
this account. The consumer flow below works. **Verified live:** create → reveal
end-to-end, through the config proxy.

## Auth model

Cookie-based, not API key:
- `__session` (your login) + `cf_clearance` (Cloudflare) cookies.
- Obtained once via a real browser login (`Capture`), then reused for HTTP calls.
- Not strictly IP-bound (verified: a cookie made on one IP worked through the
  proxy), but run login through the **same proxy** you'll use for calls to be safe.

Card creation also requires your 4-digit app **passcode** (PIN) — the client
exchanges it at `/check-pin.data` for a short-lived `pinAuthToken` + `encryptedPin`
on every create.

## Flow

```
Capture (browser, one-time)  ->  __session + cf_clearance cookie string
Create:
  check-pin.data  {pin}          -> pinAuthToken, encryptedPin
  virtual-cards/create.data {..} -> virtual_card_id, last4, status
Reveal (via pkg/fluz API)        -> full PAN / CVV / expiry
```

## Usage

```go
import (
    "merc-vcc-gen/pkg/fluz"
    "merc-vcc-gen/pkg/fluzweb"
)

// 1. one-time browser login (opens Chrome through the proxy; you complete
//    email + 2FA + passcode; it captures the cookie)
cookie, _ := fluzweb.Capture(fluzweb.LoginConfig{
    UserDataDir: "fluz_profile",
    ProxyServer: "http://your-proxy-host:PORT",
    ProxyUser:   "user", ProxyPass: "pass",
})
// persist `cookie` somewhere (it lasts hours); re-run Capture when it expires.

// 2. create cards over HTTP through the same proxy
web := fluzweb.New(fluzweb.Config{
    Cookie:   cookie,
    PIN:      "YOUR_PIN",
    ProxyURL: "http://user:pass@your-proxy-host:PORT",
})
card, _ := web.Create(fluzweb.CreateParams{
    SeatID:            "...",   // account-stable
    OfferID:           "...",   // from fluz.OfferQuote(merchantSlug, denom)
    PurchaseAmount:    500,     // multi-use spend limit (does not charge upfront)
    FluzpayAmount:     1,
    BankAccountID:     "...",   // from fluz.FundingSources()
    AddressID:         "...",   // account-stable
    UserCashBalanceID: "...",   // from fluz.Balance() userCashBalances
    CardNickname:      "my card",
    LockCardNextUse:   true,
})

// 3. reveal the PAN via the official API
api := fluz.New(fluz.Config{APIKey: "...", UserID: "...", AccountID: "..."})
d, _ := api.Reveal(card.VirtualCardID)
fmt.Println(d.Line())   // number,MM/YY,CVV,ZIP
```

## Where the IDs come from
- `OfferID` — `fluz.OfferQuote("amazon", 25)` (API).
- `BankAccountID`, `UserCashBalanceID` — `fluz.Balance()` (API).
- `SeatID`, `AddressID` — account-stable; from the consumer `.data` endpoints
  (`auth-users-list.data`, `last-virtual-card-address-id.data`) or your capture.

## Session persistence (won't drop out of nowhere)

Two independent clocks govern the session:

- **`__session`** — carries a ~10-min inner token, but the server refreshes it via
  the long-lived refresh token on every `GET /session/heartbeat`, returning a new
  cookie in `Set-Cookie`. The client captures every rolling `Set-Cookie` and, with
  `SessionPath` set, persists it to disk. So `__session` survives indefinitely as
  long as you heartbeat and the refresh token isn't revoked.
- **`cf_clearance`** — Cloudflare. Long cookie TTL but can be invalidated by
  Cloudflare. It **cannot** be renewed over pure HTTP; when it dies, requests get
  challenged. The client detects this (`ErrChallenge`) and calls your `Relogin`
  hook to re-solve it via the browser, then retries automatically.

Wire it up so it self-heals and never silently dies:

```go
web := fluzweb.New(fluzweb.Config{
    PIN:         "YOUR_PIN",
    ProxyURL:    proxyURL,
    SessionPath: "fluz_session.json",  // rolling cookie persisted here
    Relogin: func() (string, error) {  // called on Cloudflare challenge / 401
        return fluzweb.Capture(fluzweb.LoginConfig{
            UserDataDir: "fluz_profile", ProxyServer: proxyServer,
            ProxyUser: u, ProxyPass: p,
        })
    },
})

// keep __session fresh in the background (heartbeat every 5 min)
ctx, cancel := context.WithCancel(context.Background())
defer cancel()
go web.KeepAlive(ctx, 5*time.Minute)
```

On restart, `New` loads `fluz_session.json` automatically. The `fluz_profile`
Chrome dir also keeps the browser logged in, so `Relogin` usually needs no manual
2FA — it just re-solves Cloudflare and returns fresh cookies.

## API

- `New(Config) *Client` — Config: Cookie, PIN, ProxyURL, Endpoint, SessionPath, Relogin
- `Capture(LoginConfig) (cookie string, err error)` — browser login, returns cookie
- `CheckPin(pin) (pinAuthToken, encryptedPin string, err error)`
- `Create(CreateParams) (*WebCard, error)` — check-pin + create in one call
- `Heartbeat() error` — refresh + persist the session (self-relogins on challenge)
- `KeepAlive(ctx, interval)` — background heartbeat loop
- `Cookie() string` — current cookie header

## Notes
- Responses are Remix turbo-stream; the client extracts the scalar fields it needs.
- The consumer endpoints can change without notice (unlike the documented API) —
  prefer the API for everything except `create`.
