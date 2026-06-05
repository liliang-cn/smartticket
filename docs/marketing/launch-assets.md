# SmartTicket — Launch Assets (Q1)

> Built with the `launch` skill (ORB framework + Product Hunt tactics). Copy-paste-ready. Brand voice: technical, honest, anti-lock-in — never overclaim, name the trade-offs.
> **Live demo:** https://smartticket.superleo.app · **Repo:** https://github.com/liliang-cn/smartticket (MIT)
> **One-line positioning:** *Chatwoot/Zammad's self-hosting, an order of magnitude lighter — Intercom Fin's AI auto-resolve, but on your model and your server.*

---

## 0. Launch sequencing (solo / $0 reality)

Run these **staggered over one launch week**, all funneling to the **owned** channels (GitHub star + the email/waitlist + the live demo). Per the ORB model: Reddit/HN/PH/X are *rented* — capture the attention into owned.

| Day | Channel (type) | Asset |
|---|---|---|
| **T-7 → T-1** | Owned/Borrowed | Build-in-public teasers on X/LinkedIn; line up a few people who'll engage; add an email-capture to the landing; pre-write everything below |
| **T-0 (Tue, 12:01am PT)** | Product Hunt (rented) | PH listing + maker's first comment |
| **T-0 morning** | Hacker News (rented) | Show HN post |
| **T-0 same day** | Reddit (rented) | r/selfhosted post (then r/opensource, r/SaaS spaced out) |
| **T-0 all day** | X + LinkedIn (rented) | Launch thread + LinkedIn post — link to PH + demo |
| **T-0** | Email (owned) | Announcement to any list/waitlist |
| **T+1 → T+7** | Owned | Reply to every comment; convert traffic → stars + email; publish the comparison pages |

**Best PH day:** Tue–Thu. **HN:** post Show HN in the US morning (≈8–10am ET) on a weekday; don't cross-link PH↔HN in the posts. **Reddit:** value-first, no marketing tone, disclose you're the author.

---

## 1. Product Hunt

**Name:** SmartTicket

**Tagline** (≤60 chars — pick one):
1. `Self-hosted AI helpdesk — your model, your server` (49)
2. `Open-source Zendesk alternative with BYO-AI` (43)
3. `The helpdesk that keeps your customer data yours` (49)
4. `Self-hosted helpdesk + AI that runs on your LLM` (47)

→ **Recommended:** #1 (states all three wedge points: self-hosted, AI, data ownership).

**Topics/Tags:** Customer Success, SaaS, Open Source, Artificial Intelligence, Privacy, Developer Tools

**Description** (≤260 chars):
> SmartTicket is a single-binary, self-hosted helpdesk: tickets, knowledge base, web chat widget, SLA, and AI auto-resolve that runs on *your* LLM (OpenAI, Claude, DeepSeek, or a local model). MIT-licensed. Zero customer data leaves your server.

**Gallery (in order — the story sells, not the feature list):**
1. **Hero:** the chat widget on a site + a ticket in the console side-by-side. Caption: "A website chat that becomes a tracked ticket — on your server."
2. **GIF (the money shot):** AI auto-resolve — a customer message → AI drafts a reply from the knowledge base with a confidence score. Caption: "AI auto-resolve on your own model — confidence-gated, sources cited."
3. **Deploy:** a terminal showing `docker run … smartticket` → running in seconds. Caption: "One binary. SQLite built in. Runs in <512MB. 5-minute deploy."
4. **Data sovereignty diagram:** your server box containing data + LLM, nothing leaving. Caption: "BYO-LLM. Nothing leaves your infrastructure. MIT-licensed."
5. **Console:** tickets list + automations/macros/CSAT. Caption: "SLA, RBAC, automations, macros, CSAT, ticket merge — the full desk."

**Maker's first comment** (post immediately when live):
> Hey Product Hunt 👋 I'm the maker.
>
> I built SmartTicket because every helpdesk worth using is a SaaS that holds your customers' data, and every self-hosted one (Chatwoot, Zammad) makes you run Postgres + Redis + Elasticsearch + a worker fleet. I wanted the middle: a **real** helpdesk you can run as a **single binary** on a small box, where the **AI runs on your own model** and **no customer data ever leaves your server**.
>
> What's in it today:
> • Tickets, knowledge base, SLA, RBAC, audit logs
> • An embeddable **web chat widget** (one `<script>` tag)
> • **AI auto-resolve** — bring your own LLM (OpenAI / Claude / DeepSeek / Ollama / local). It drafts replies from your KB with a confidence score and can auto-send above a threshold, or hand off to a human.
> • Automations/triggers, macros, CSAT surveys, teams, ticket merge
> • Two-way email, 7-language UI, full data export, MCP tools so an AI agent can drive the desk
>
> Single binary, embedded SQLite, runs in <512MB, MIT-licensed.
>
> **Honest about what it's not (yet):** no WhatsApp/Slack channels, single-instance (no multi-region HA yet), and it's young — I'd love your feedback to prioritize.
>
> Try the live demo (no install): https://smartticket.superleo.app · Code: https://github.com/liliang-cn/smartticket
>
> I'll be here all day — ask me anything about the architecture, the BYO-AI design, or self-hosting it.

**Hunter note (if you find a hunter):** keep it short — "Self-hosted, single-binary AI helpdesk. BYO-LLM, MIT, data never leaves your server. Open-source alternative to Zendesk/Intercom for teams that can't put customer data in a SaaS."

---

## 2. Hacker News — Show HN

**Title** (HN is allergic to hype — plain + specific):
> `Show HN: SmartTicket – Self-hosted AI helpdesk that runs on your own LLM (MIT)`

**URL:** https://github.com/liliang-cn/smartticket (link the repo, not the marketing page)

**Body (first comment):**
> I'm a solo dev and I built SmartTicket, a self-hosted helpdesk (tickets + knowledge base + web chat widget + SLA/RBAC) with AI auto-resolve.
>
> The two things that made me build it instead of using something existing:
>
> 1. **Data sovereignty without the ops tax.** Chatwoot/Zammad are great but want Postgres + Redis + Elasticsearch + Sidekiq/worker processes. SmartTicket is a single Go binary with embedded SQLite (pure-Go modernc driver, CGO-free), runs in <512MB, and deploys in ~5 minutes. The whole thing — DB, vector store for RAG, web UI — is in one process.
>
> 2. **AI that's actually yours.** The AI auto-resolve runs on a model *you* configure — OpenAI, Anthropic, DeepSeek, or a local one via Ollama/vLLM. It does RAG over your own knowledge base, returns a structured draft `{reply, confidence, needs_clarification, sources}`, and can either suggest to an agent or (above a confidence threshold you set) auto-reply. No customer data is sent anywhere except the LLM endpoint you point it at. Default is suggest-only.
>
> Also in there: an embeddable chat widget (one script tag, shadow-DOM isolated), automations/triggers, macros, CSAT, teams, ticket merge, two-way email, full data export, and MCP tools so an AI agent can operate the desk programmatically.
>
> **Stack:** Go + Gin + GORM, SQLite (WAL), agent-go for the AI layer, a small in-process WebSocket hub for real-time. No external services required.
>
> **What it's not:** no social channels (WhatsApp/Slack) yet, single-instance only (the WebSocket hub is in-process, so no multi-replica HA yet), monetization undecided (it's MIT and free to self-host). It's early.
>
> Live demo (creates a throwaway ticket): https://smartticket.superleo.app
> Code + deploy docs: https://github.com/liliang-cn/smartticket
>
> Happy to go deep on the architecture — the CGO-free SQLite + embedded RAG, the BYO-LLM structured-output design, or the single-binary tradeoffs. Feedback very welcome, especially on what would make you actually self-host it.

*(HN tips: respond fast and technically; never argue; if someone says "why not X", thank them and answer honestly; the structured-output + CGO-free-SQLite details are your credibility hooks.)*

---

## 3. Reddit — r/selfhosted

**Title:**
> `I built a single-binary, self-hosted helpdesk with AI that runs on your own LLM (MIT, no Postgres/Redis needed)`

**Body:**
> Hey r/selfhosted — author here, sharing a thing I built and use.
>
> Most self-hosted helpdesks (Chatwoot, Zammad) are excellent but heavy — they want Postgres + Redis + Elasticsearch + worker processes. I wanted something I could run on a $5 box without a stack, so I built **SmartTicket**:
>
> - **Single Go binary**, embedded SQLite, runs in <512MB, `docker run` and you're up in ~5 min. No external services.
> - **Tickets + knowledge base + SLA + RBAC + audit logs** — a real desk, not a toy.
> - **Embeddable web chat widget** (one `<script>` tag) so your site visitors can chat → becomes a ticket.
> - **AI auto-resolve on your own model** — point it at OpenAI/Claude/DeepSeek or a local Ollama model. It answers from your KB, shows a confidence score, and only auto-replies if you turn that on. Nothing leaves your server except the call to the LLM endpoint you chose.
> - Automations, macros, CSAT, teams, ticket merge, two-way email, 7-language UI, full data export.
> - **MIT licensed.**
>
> **Honest caveats:** single-instance (the realtime hub is in-process — no multi-node HA yet), no WhatsApp/Slack channels, and it's young.
>
> Live demo: https://smartticket.superleo.app · Code: https://github.com/liliang-cn/smartticket
>
> Would genuinely love feedback from this crowd on deployment ergonomics and what's missing. Not selling anything — it's open source.

*(Reddit rules: check each sub's self-promo policy, post value-first, disclose authorship, don't drop the same text in 5 subs the same hour. Space r/opensource and r/SaaS a day or two apart, retitled.)*

---

## 4. X / Twitter — launch thread

**Tweet 1 (hook):**
> Every helpdesk worth using is a SaaS that holds your customers' data.
> Every self-hosted one makes you run Postgres + Redis + Elasticsearch.
>
> So I built the middle: a single-binary, self-hosted AI helpdesk where the AI runs on YOUR model. 🧵
> 👉 [PH link]

**Tweet 2:**
> SmartTicket = tickets + knowledge base + web chat widget + SLA/RBAC + AI auto-resolve.
>
> One Go binary. Embedded SQLite. <512MB RAM. 5-minute deploy. MIT.
> No external services. Nothing leaves your server.

**Tweet 3 (the differentiator + GIF):**
> The AI runs on YOUR LLM — OpenAI, Claude, DeepSeek, or a local Ollama model.
>
> It drafts replies from your knowledge base, with a confidence score + sources. Suggest-only by default; flip a switch and it auto-resolves above your threshold.
> [auto-resolve GIF]

**Tweet 4 (embed):**
> The chat widget is one `<script>` tag. Shadow-DOM isolated, ~14KB.
> Your site visitor chats → it becomes a tracked ticket → your agent (or the AI) replies live.
> [widget GIF]

**Tweet 5 (honest close + CTA):**
> Honest: it's young, single-instance, no social channels yet. But it's a real desk you fully own.
>
> Live demo (no install): https://smartticket.superleo.app
> Code (MIT): https://github.com/liliang-cn/smartticket
> We're live on Product Hunt today — would love your support 👉 [PH link]

---

## 5. LinkedIn — launch post

> **I shipped SmartTicket: a self-hosted AI helpdesk where your customer data never leaves your server.**
>
> The problem: if you want a modern helpdesk with AI, you put your customers' data into a SaaS (Zendesk, Intercom). If you refuse to — for compliance, data-residency, or principle — your options are heavy self-hosted stacks (Postgres + Redis + Elasticsearch + workers).
>
> SmartTicket is the middle path:
> → One Go binary, embedded SQLite, runs in <512MB, deploys in 5 minutes
> → Tickets, KB, web chat widget, SLA, RBAC, automations, macros, CSAT
> → **AI auto-resolve that runs on your own LLM** (OpenAI, Claude, DeepSeek, or local) — RAG over your KB, confidence-gated, nothing leaving your infra
> → MIT-licensed, full data export, 7-language UI
>
> It's for teams that can't (or won't) put customer data in a SaaS — regulated industries, EU data-residency, gov — and for indie founders who want their own support desk without per-seat SaaS pricing.
>
> Honest about the edges: it's early, single-instance, no social channels yet. But it's a real desk you completely own.
>
> Live demo (no install): https://smartticket.superleo.app
> Open source: https://github.com/liliang-cn/smartticket
>
> Live on Product Hunt today — feedback and a follow mean a lot. 🙏
>
> #opensource #selfhosted #customersupport #AI #dataprivacy

---

## 6. Launch email (owned — to list/waitlist)

**Subject lines (A/B):**
- A: `SmartTicket is live: a helpdesk that keeps your data yours`
- B: `I shipped it — self-hosted AI helpdesk, your model, your server`

**Body:**
> Hi {first_name},
>
> Quick one: **SmartTicket is live.**
>
> It's a self-hosted helpdesk — tickets, knowledge base, web chat widget, SLA — with **AI auto-resolve that runs on your own LLM**. One binary, 5-minute deploy, MIT-licensed, and zero customer data leaves your server.
>
> Two ways to see it in 60 seconds:
> 1. **Live demo (no install):** https://smartticket.superleo.app
> 2. **Self-host it:** https://github.com/liliang-cn/smartticket
>
> We're on Product Hunt today — if it's useful to you, an upvote/comment genuinely helps a solo project get seen: [PH link]
>
> And just reply to this email if you want — I read every one. Tell me the one thing that would make you actually deploy it.
>
> — {your name}
>
> P.S. Honest disclaimer: it's young. No WhatsApp/Slack yet, single-instance. But it's a real, complete desk you fully own.

---

## 7. Launch-week checklist (from the `launch` skill, trimmed to solo/$0)

**Pre-launch (T-7 → T-1)**
- [ ] Landing has a clear value prop + email capture (owned-channel funnel)
- [ ] Primary CTA = "Try live demo" + "Deploy in 5 min"
- [ ] All assets above finalized; PH listing drafted (not submitted)
- [ ] Record the 2 GIFs: (a) AI auto-resolve, (b) widget round-trip
- [ ] Line up 5–10 people who'll engage on launch morning
- [ ] Tease build-in-public on X/LinkedIn for a few days
- [ ] Comparison pages drafted (publish T+1 for the post-launch SEO catch)

**Launch day (T-0)**
- [ ] PH live 12:01am PT + maker's first comment posted
- [ ] Show HN posted (US morning); reply to every comment fast
- [ ] r/selfhosted posted (value-first, authorship disclosed)
- [ ] X thread + LinkedIn post live, linking PH + demo
- [ ] Email to list
- [ ] Watch the demo instance + server (you'll get a traffic spike — confirm it holds)
- [ ] Respond to EVERY comment everywhere, all day

**Post-launch (T+1 → T+7)**
- [ ] Convert traffic → GitHub stars + email signups (the only durable asset)
- [ ] Publish the 4 comparison pages (vs Chatwoot / Zendesk / Intercom / Zammad)
- [ ] Submit to awesome-selfhosted, AlternativeTo, libhunt
- [ ] Follow up with everyone who engaged; thank them
- [ ] Write a short "what I learned launching" post (reusable content + seeds the next launch)
- [ ] Note what resonated → that's the angle for launch #2 (a lifetime deal or a v2 Show HN)

---

## Pre-launch fixes worth doing first (gates the launch quality)
1. **Landing email capture + "Try live demo" primary CTA** — without an owned funnel, launch traffic evaporates.
2. **Record the two GIFs** — the auto-resolve GIF is the single most persuasive asset; PH/Reddit/X all need it.
3. **Widget intake hardening** (domain allowlist / rate-limit) — before you invite a crowd to poke the public demo.
4. **Make sure the live demo survives a spike** — it's your zero-install conversion surface on launch day.
