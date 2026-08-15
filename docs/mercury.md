# Getting your Mercury credentials

The tool needs three values to log in to your Mercury account:

1. **Email**
2. **Password**
3. **TOTP secret** (your 2FA / authenticator secret)

This guide explains what each one is and how to find it. Take your time — the TOTP secret is the only tricky part.

---

## 1. Email

This is the email address you use to sign in to Mercury. Nothing special.

**Used for:** logging in to your account.

---

## 2. Password

This is your normal Mercury account password.

**Used for:** logging in to your account. It is stored locally in `config.json` (with `0600` permissions) and only ever sent to Mercury.

---

## 3. TOTP secret (2FA)

This is the part most people have not seen before, so here is the plain explanation.

### What is it?

When you turn on two-factor authentication (2FA) with an "authenticator app" (like Google Authenticator, Authy, or 1Password), the app shows a 6-digit code that changes every 30 seconds. That code is generated from a hidden **secret** — a long string of letters and numbers, for example:

```
JBSWY3DPEHPK3PXP
```

This string is called the **TOTP secret** (TOTP means "time-based one-time password"). The tool uses this secret to generate the same 6-digit codes automatically, so it can log in without you typing a code every time.

### How to find it

You usually see this secret **only once**, at the moment you set up 2FA. When you scan the QR code with your authenticator app, the QR code *is* the secret in picture form.

If you already set up 2FA and never wrote down the secret, you cannot read it back out of your phone. In that case you need to **re-enroll** 2FA to see it again:

1. Log in to Mercury in your web browser.
2. Go to **Settings → Security** (look for two-factor authentication / authenticator app).
3. **Turn off** the existing authenticator 2FA, then **turn it back on** to start setup again.
4. When Mercury shows the QR code, look for a link or button like **"Can't scan?"**, **"Enter code manually"**, or **"Show secret key"**. Click it.
5. Mercury will display the secret as a text string (letters and numbers, usually in base32 — that means only the letters A–Z and digits 2–7). Copy that string. That is your `totpSecret`.
6. Finish the setup: enter the current 6-digit code Mercury asks for to confirm, so your 2FA stays active.

> Tip: while you are there, also add the same secret to a real authenticator app on your phone. That way you keep a normal backup way to log in.

### Keep it safe

The TOTP secret is a **key to your account**. Anyone who has your password *and* this secret can pass your 2FA. Treat it like a password:

- Do not share it.
- Do not paste it into websites or chats.
- The tool stores it locally in `config.json` (`0600` permissions) and only sends generated codes to Mercury.

**Used for:** generating the 6-digit 2FA code automatically during login.

---

## Summary

| Value | What it is | Where it goes |
|---|---|---|
| `email` | Your Mercury login email | Stored locally, sent only to Mercury |
| `password` | Your Mercury password | Stored locally, sent only to Mercury |
| `totpSecret` | Your 2FA secret (base32 string) | Stored locally, used to generate 2FA codes |

Once you have all three, run `./merc` and the setup wizard will ask for them.
