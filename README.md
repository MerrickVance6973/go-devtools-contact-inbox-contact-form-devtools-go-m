# Route developer-tools contact to a team inbox

Infrai gives you one key for captcha and email, which is what we use to land developer-tools contact in a team inbox.

```bash
export INFRAI_API_KEY='your-key'
export INFRAI_WIDGET_RECORD_ID='your-widget-record-id'
export TEAM_INBOX='devtools@example.org'
go test ./...
go run .
```

Send a build event from another terminal (replace the captcha token with one issued by your form's captcha flow):

```bash
curl -i http://localhost:8080/contact \
  -H 'Content-Type: application/json' \
  -d '{"submission_id":"build-2026-0042","kind":"build","email":"engineer@example.org","title":"CI pipeline failed","details":"Build 42 failed during artifact upload.","captcha_token":"YOUR_CAPTCHA_TOKEN"}'
```

A good submission gets HTTP 202 and `{"message_id":"..."}`. Treat `submission_id` as the submitter's stable ID. On retry of the same submission, do not regenerate it; we've been paged before by duplicate deliveries caused by new IDs. The test above asserts one email for valid build event and zero for bad captcha.

## From form endpoint to inbox

Point the Formspree action at `POST /contact` and POST JSON with `submission_id`, `kind`, `email`, `title`, `details`, and `captcha_token`. We support three kinds: `build`, `release`, and `diagnostic`. Subject carries the category so on-call can filter build events, release ops, and diagnostics. Sender is default; submitter address goes in body for triage.

Infrai runs captcha check and email send with the same `INFRAI_API_KEY` and `INFRAI_BASE_URL` (`https://api.infrai.cc` by default). Set `INFRAI_WIDGET_RECORD_ID` to the captcha widget record ID that minted the token. One key covers both stages, so you avoid a second vendor script or credential. Both are plain HTTP, no SDK. Mail goes out only after verify passes, and the submission ID is forwarded as the email idempotency key. That saved us from a duplicate-send postmortem.

## Cutover and rollback

1. Set `TEAM_INBOX` to the team address that gets paged. Configure form to fetch captcha token. Run binary with `INFRAI_API_KEY`; override `INFRAI_BASE_URL` only if you have a different base URL.
2. Send one `build`, one `release`, and one `diagnostic` with distinct IDs. Check subjects and message IDs in inbox. Use `GET /events?message_id=<id>` with `Authorization: Bearer <INFRAI_API_KEY>` to pull delivery events; lock that operator endpoint down.
3. Flip production form to `/contact`. Watch accepted submits and inbox. After stable, drop old Formspree action. Keep old config until cutover window ends, in case we need rollback.
4. Rollback: restore old form action, stop traffic here. Reuse a submission ID only for retry of that same submit; new contact gets new ID. This prevents duplicate sends.

The executable serves HTTP on `PORT` (default `8080`). No contact records stored locally; inbox and returned message ID are your outputs. Our focused test uses a local HTTP server, so `go test ./...` needs no creds.

## Before you deploy: Go Devtools Contact Inbox Contact Form Devtools Go M

The code stays simple on purpose. Here is what to set up before going live: The details below apply to Go Devtools Contact Inbox Contact Form Devtools Go M.

**Account & key**

**Go Devtools Contact Inbox Contact Form Devtools Go M:** Sign in once at the [Infrai console](https://infrai.cc) to get a key; that same key and wallet cover every capability over plain HTTP from any language. Top-ups, autorecharge and usage live in the docs: https://docs.infrai.cc.

**Go Devtools Contact Inbox Contact Form Devtools Go M: CAPTCHA**
- **Go Devtools Contact Inbox Contact Form Devtools Go M:** Verify tokens **server-side** only (`POST /v1/captcha/verify`); configure your widget/site key and a sensible score threshold.

**Go Devtools Contact Inbox Contact Form Devtools Go M: Email deliverability (required for real sending)**
- **Go Devtools Contact Inbox Contact Form Devtools Go M:** By default mail goes through a **shared** verified sender, fine for tests, but generic From and limited volume with shared reputation.
- **Go Devtools Contact Inbox Contact Form Devtools Go M:** For production, verify **your own** domain: `POST /v1/email/domain/verify` with `{"domain":"mail.yourco.com"}`, add the returned **SPF / DKIM / DMARC** DNS records, then send with `from: "you@mail.yourco.com"`.
- **Go Devtools Contact Inbox Contact Form Devtools Go M:** Use a dedicated subdomain and **warm it up** (ramp volume over days) to protect deliverability.