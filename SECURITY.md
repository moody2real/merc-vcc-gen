# Security

This document explains what data the tool stores, where it goes, and how to report a problem.

## What data is stored

- **`config.json`** — your credentials. For Mercury: email, password, and TOTP secret. For Fluz: apiKey, userId, accountId, and optional seatId. Also your card settings and optional proxy.
- **Session files** — after a successful Mercury login, the session cookie is saved locally so you do not have to log in every time.
- **`data/cards_<timestamp>.txt`** — the virtual cards you generate.

## Where it is stored

Everything stays **on your local machine**, inside the tool's own folder. Sensitive files are written with `0600` permissions, which means only your user account can read them.

Nothing is uploaded to any server, database, or third party.

## What is transmitted, and where

The tool only sends network traffic to:

- **Mercury** — to log in and manage your Mercury cards.
- **Fluz** — to manage your Fluz cards.
- **An optional proxy** — only if *you* configure one in `config.json`. If you leave the proxy blank, traffic goes directly to Mercury/Fluz.

That is the complete list. Your credentials never go anywhere except to the provider they belong to.

## Never commit your secrets

`config.json` and session files contain live credentials. **Never** commit them to git or share them.

These files are already listed in `.gitignore`, but double-check before pushing:

```bash
git status
```

If you see `config.json` or any session/data file listed, do **not** commit it.

## Reporting a vulnerability

If you find a security issue, please report it privately. Do **not** open a public issue that reveals the problem.

- Open a [GitHub security advisory](../../security/advisories/new) on this repository, **or**
- Email the maintainer.

Please include steps to reproduce and the impact. You will get a response as soon as possible.

## Responsible use

This tool is for managing **your own** Mercury and Fluz accounts, using **your own** credentials. Do not use it with accounts or credentials that are not yours. You are responsible for following Mercury's and Fluz's terms of service.
