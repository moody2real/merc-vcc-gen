![Build](https://img.shields.io/badge/build-passing-brightgreen)
![License](https://img.shields.io/badge/license-MIT-blue)
![Go](https://img.shields.io/badge/go-1.22%2B-00ADD8)

# merc-vcc-gen

Generate virtual cards on your own Mercury and Fluz accounts, straight from the terminal.

---

## Is this safe?

Short answer: yes, and you can prove it yourself. Here is why you can trust it.

- **It is open source.** Every line of code is public in this repository. Read it before you run it.
- **Builds are automated.** Release binaries are built by GitHub Actions from the public source. No one uploads a binary by hand, so what you download matches what you can read.
- **Every release is verifiable.** Each release includes:
  - A `SHA256` checksums file so you can confirm your download was not tampered with.
  - A **VirusTotal scan link** so you can see a third-party antivirus report before you run anything.
- **You can build it yourself.** If you do not want to trust the prebuilt binary at all, compile from source:
  ```bash
  go build ./cmd/merc
  ```
  The binary you build is the tool. Nothing hidden.
- **Your secrets stay on your machine.** Credentials are written to a local `config.json` file with `0600` permissions (only your user can read it). The tool only talks to Mercury and Fluz (and an optional proxy that *you* configure). Nothing is ever sent to any third party.

This is a personal automation tool. You log in with **your own** credentials to manage **your own** cards.

---

## Install

Pick one of the three options below.

### Option 1: Download a prebuilt binary (recommended)

1. Go to the [Releases](../../releases) page and download the file for your system.
2. Verify the checksum before running it.

**Linux / macOS:**
```bash
# from the folder where you downloaded the binary and the checksums file
sha256sum -c merc-vcc-gen_checksums.txt
```

**macOS (if `sha256sum` is missing):**
```bash
shasum -a 256 -c merc-vcc-gen_checksums.txt
```

**Windows (PowerShell):**
```powershell
# compare this output against the value in the checksums file
Get-FileHash .\merc.exe -Algorithm SHA256
```

3. Make it runnable (Linux / macOS):
```bash
chmod +x ./merc
```

### Option 2: `go install`

Requires Go 1.22 or newer.
```bash
go install github.com/moody2real/merc-vcc-gen/cmd/merc@latest
```
The `merc` binary lands in your `$GOPATH/bin` (usually `~/go/bin`). Make sure that folder is on your `PATH`.

### Option 3: Build from source

```bash
git clone https://github.com/moody2real/merc-vcc-gen.git
cd merc-vcc-gen
go build ./cmd/merc
```

This produces a `merc` binary (or `merc.exe` on Windows) in the current folder.

---

## Quick start

1. Run the tool:
   ```bash
   ./merc
   ```
   On Windows: `.\merc.exe`
2. On the **first run**, an interactive setup wizard starts. It asks for your provider choice and credentials (passwords are typed hidden).
3. The wizard writes `config.json` locally with `0600` permissions.
4. Choose your provider (**Mercury** or **Fluz**) and how many cards to create.
5. The tool creates the cards and writes them to a file under `data/`. Done.

---

## Getting credentials

You need account credentials before the wizard can do anything.

- **Mercury:** see [docs/mercury.md](docs/mercury.md) — email, password, and your TOTP (2FA) secret.
- **Fluz:** see [docs/fluz.md](docs/fluz.md) — API key, userId, and accountId.

---

## Usage

After setup, running `./merc` shows a menu:

- **Create cards** — generate one or more virtual cards.
- **Delete card** — remove a virtual card.
- **List cards** — show your existing virtual cards.

### Where output goes

Created cards are written to:
```
data/cards_<timestamp>.txt
```
The file is created with `0600` permissions (only your user can read it).

Each line uses this format:
```
pan,mm/yy,cvv,zip
```
Example:
```
4111111111111111,08/29,123,10001
```

---

## Config reference

The wizard creates `config.json`. You can edit it by hand later if you want.

| Field | Description |
|---|---|
| `provider` | Which provider to use: `mercury` or `fluz`. |
| `mercury` | Mercury credentials: `email`, `password`, `totpSecret`. |
| `fluz` | Fluz credentials: `apiKey`, `userId`, `accountId` (optional `seatId`). |
| `proxy` | Optional. Proxy URL for all outgoing traffic. Leave empty to go direct. |
| `card.dailyLimit` | Spending limit per day for new cards. |
| `card.zip` | ZIP / postal code attached to new cards. |
| `card.nickname` | Nickname / label for new cards. |
| `card.delaySeconds` | Seconds to wait between creating each card. |
| `browser.headless` | Mercury only. `true` runs Chrome hidden, `false` shows the login window. |

---

## FAQ / troubleshooting

**A browser window opened during Mercury login. Is that normal?**
Yes. Mercury logs in through a real Chrome browser. If you set `browser.headless` to `false`, you will see the window. Set it to `true` to keep it hidden.

**Login fails with a "wrong code" or 2FA error.**
Your TOTP secret is probably wrong or your computer clock is off. TOTP codes depend on the correct time. Re-check your `totpSecret` (see [docs/mercury.md](docs/mercury.md)) and make sure your system clock is set to sync automatically.

**Nothing connects / everything times out.**
If you set a `proxy`, check that the proxy URL is correct and the proxy is online. Remove it to test a direct connection.

**It logged me in before, but now it asks again or fails.**
Your saved session cookie expired. This is normal. Just run the tool again and it will log in fresh.

**Where is my data stored?**
`config.json` (your credentials) and any session files live in the tool's folder, all with `0600` permissions. Cards are written to `data/`. Nothing leaves your machine except calls to Mercury/Fluz. See [SECURITY.md](SECURITY.md).

---

## License

MIT. See [LICENSE](LICENSE).
