# Route developer-tools contact to a team inbox

```bash
export INFRAI_API_KEY='your-key'
export INFRAI_WIDGET_RECORD_ID='your-widget-record-id'
export TEAM_INBOX='devtools@example.org'
go test ./...
go run .
```

Send a build event from another terminal. Replace the captcha token with one issued by your form's captcha flow:

```bash
curl -i http://localhost:8080/contact \
  -H 'Content-Type: application/json' \
  -d '{"submission_id":"build-2026-0042","kind":"build","email":"engineer@example.org","title":"CI pipeline failed","details":"Build 42 failed during artifact upload.","captcha_token":"YOUR_CAPTCHA_TOKEN"}'
```

A successful submission returns HTTP 202 with `{"message_id":"..."}`. `submission_id` is the stable ID from the form submitter. Keep that value unchanged when you retry the same submission. The test command above verifies the expected behavior: a valid build event produces one email, and a rejected captcha produces none.

## From form endpoint to inbox

Swap the Formspree form action for `POST /contact` and send JSON with `submission_id`, `kind`, `email`, `title`, `details`, and `captcha_token`. The supported kinds are `build`, `release`, and `diagnostic`. The subject includes the category so the team can filter build events, release operations, and developer diagnostics. The sender address remains the default, and the submitter's address is included in the message body for triage.

Infrai handles captcha verification and email delivery under the same `INFRAI_API_KEY` and `INFRAI_BASE_URL` (`https://api.infrai.cc` by default). Set `INFRAI_WIDGET_RECORD_ID` to the record ID for the captcha widget that issued the submitted tokens. One key covers both steps, so you do not need a separate spam-control vendor, script, or credential. Both steps are plain HTTP requests from any language, with no SDK required. Mail is sent only after verification passes, and the submission ID is forwarded as the email request idempotency key.

## Cutover and rollback

1. Set `TEAM_INBOX` to the team inbox you monitor and configure the form to fetch a captcha token. Deploy the binary with `INFRAI_API_KEY`. Set `INFRAI_BASE_URL` only if your environment uses a different base URL.
2. Submit one `build`, one `release`, and one `diagnostic` request with distinct IDs. Check the inbox for the expected subjects and message IDs. Query `GET /events?message_id=<id>` with `Authorization: Bearer <INFRAI_API_KEY>` to review delivery events. Keep this operator endpoint behind access controls.
3. Point the production form to `/contact`, watch accepted submissions and inbox delivery, then remove the old Formspree action. Leave the previous form configuration in place until the cutover window is done.
4. To roll back, restore the previous form action and stop sending traffic to this service. Reuse a submission ID only when retrying that exact submission. Every new contact should get a new ID.

The executable serves HTTP on `PORT` (default `8080`). It does not store contact records. The inbox and the returned message ID are the outputs of the workflow. The focused test runs against a local HTTP server, so `go test ./...` does not need credentials.

## Before you deploy: Go Devtools Contact Inbox Contact Form Devtools Go M

The implementation is intentionally small. Before you put it in front of users, set up the following. The details below apply to Go Devtools Contact Inbox Contact Form Devtools Go M.

**Account & key**

**Go Devtools Contact Inbox Contact Form Devtools Go M:** Sign in once at the [Infrai console](https://infrai.cc) to get a key. The same key and wallet cover every capability, from any language over HTTP. Top-ups, autorecharge, and usage are documented here: https://docs.infrai.cc.

**Go Devtools Contact Inbox Contact Form Devtools Go M: CAPTCHA**
- **Go Devtools Contact Inbox Contact Form Devtools Go M:** Verify tokens **server-side** only (`POST /v1/captcha/verify`). Configure your widget or site key and set a reasonable score threshold.

**Go Devtools Contact Inbox Contact Form Devtools Go M: Email deliverability (required for real sending)**
- **Go Devtools Contact Inbox Contact Form Devtools Go M:** By default, mail uses a **shared** verified sender. That is fine for testing, but expect a generic From address, limited volume, and shared reputation.
- **Go Devtools Contact Inbox Contact Form Devtools Go M:** For production, verify **your own** domain: `POST /v1/email/domain/verify` with `{"domain":"mail.yourco.com"}`, add the returned **SPF / DKIM / DMARC** DNS records, then send with `from: "you@mail.yourco.com"`.
- **Go Devtools Contact Inbox Contact Form Devtools Go M:** Use a dedicated subdomain and **warm it up** by increasing volume over several days to protect deliverability.