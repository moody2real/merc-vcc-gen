# Getting your Fluz credentials

The tool talks to Fluz through its GraphQL API. To do that it needs these values:

1. **apiKey** (required)
2. **userId** (required)
3. **accountId** (required)
4. **seatId** (optional)

Being honest up front: these are **API credentials**, not something you can copy off a normal screen in the Fluz app. You must obtain them from your Fluz account's API / developer access. If you do not have API access to your Fluz account, you cannot use the Fluz provider — that is a limit on the Fluz side, not this tool.

---

## What each value is

### apiKey

A secret token that authenticates your API requests to Fluz. It acts like a password for the API. Anyone who has it can act as your account through the API, so keep it private.

**Used for:** authenticating every request to the Fluz GraphQL API.

### userId

The unique ID of your Fluz user account. It identifies *who* you are in the Fluz system.

**Used for:** telling Fluz which user is making the request.

### accountId

The unique ID of the Fluz account (wallet / funding account) that the virtual cards are created under. A user can be linked to an account; this ID says *which* one to use.

**Used for:** telling Fluz which account the new cards belong to.

### seatId (optional)

Some Fluz setups (for example, team or business accounts) use "seats" to separate members inside one account. If your account uses seats, you may need to provide the `seatId`. If you do not know what this is, you most likely do not need it — leave it blank.

**Used for:** targeting a specific seat inside an account, when the account uses seats.

---

## How to get them

These values come from the Fluz API layer, so the exact steps depend on the access Fluz gives you. In general:

1. Confirm your Fluz account has **API / developer access**. If it does not, request it from Fluz.
2. Generate or copy your **apiKey** from wherever Fluz exposes it (a developer settings page, an API dashboard, or a value Fluz support provides).
3. Find your **userId** and **accountId**. These are typically returned by Fluz API responses (for example, the response from a "me" / account query) or shown alongside your API key.
4. If your account uses seats and you need a specific one, note the **seatId** too.

If you are unsure where to find these, contact Fluz support or check their API documentation — this tool cannot create these values for you, it only uses them.

---

## Summary

| Value | Required? | What it is | Where it goes |
|---|---|---|---|
| `apiKey` | Yes | API secret token | Stored locally, sent only to Fluz |
| `userId` | Yes | Your Fluz user ID | Stored locally, sent only to Fluz |
| `accountId` | Yes | The account cards are created under | Stored locally, sent only to Fluz |
| `seatId` | No | A specific seat inside the account | Stored locally, sent only to Fluz |

All values are stored locally in `config.json` (`0600` permissions) and are only ever sent to Fluz. Once you have them, run `./merc` and the setup wizard will ask for them.
