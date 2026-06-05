# SmartTicket — Marketing Plan v1

> Generated with the `marketing-plan` skill (fCMO-level, AARRR-structured). Date: 2026-06-05.
> **Intake basis (assumptions — correct any that are wrong):** bootstrapped solo founder; ~$0 paid budget; product just shipped (pre-revenue, $0–10K ARR phase); open-source (MIT); live at smartticket.superleo.app. Idea numbers in `[#NN]` reference the 139-idea `marketing-ideas` library.

---

## Section 1 — Executive summary

SmartTicket is a **self-hosted, single-tenant AI helpdesk** — tickets + knowledge base + bring-your-own-LLM AI in one binary you run on your own server. The market is crowded, so the plan does **not** chase feature parity with Zendesk. It bets the entire go-to-market on one narrow, real wedge: **"Chatwoot/Zammad's open-source self-hosting, but an order of magnitude lighter; Intercom Fin's AI auto-resolve, but the model and data are 100% yours."**

The buyer is two-headed: (a) **data-sovereignty teams** (regulated / 政企 / air-gapped, who can't or won't put customer data in a SaaS), and (b) **indie & SaaS founders** who want a self-hosted support desk for their own product. Both are reachable for ~$0 through the motions open-source/indie products win with: **content & comparison SEO, a credible launch, open-source community, integration/"powered-by" loops, and developer marketing** — not paid ads.

**The single thesis (north star):** *self-hosted installs that activate (first ticket resolved with AI) → GitHub stars + word-of-mouth → a paid managed/license tier later.* Distribution is the constraint, not the product. The next 90 days exist to (1) make the product trivially try-able and credibly trustworthy (1-command deploy, comparison pages, a real launch), and (2) stand up the two repeatable content engines (knowledge-base/SEO + "vs Chatwoot/Zendesk" pages) that compound.

**Top 3 moves this quarter:** ① a real **launch** (Product Hunt + Show HN + r/selfhosted) `[#77, #38, #80]`; ② **comparison + migration pages** ("SmartTicket vs Chatwoot/Zendesk/Intercom", "Import from Zendesk") `[#11, #16, #91]`; ③ **founder-led content** documenting the build + the data-sovereignty/BYO-AI angle `[#39, #45, #6]`.

---

## Section 2 — Strategic frame

**What SmartTicket is, in one sentence:** A single-binary, self-hosted helpdesk (tickets, knowledge base, web chat widget, SLA, RBAC) with AI auto-resolve that runs on *your* LLM and *your* infrastructure — zero data leaves your server.

**The category we're claiming:** *Self-hosted / data-sovereign AI customer support.* We are not "another Zendesk alternative" generically — we own the intersection nobody else sits in: **lightweight self-host + data sovereignty + swappable/local AI.** (Per the competitive matrix, no competitor is full-marks on all three at once.)

**Who we're for (ICP, distilled):**
- **Segment A — Data-sovereignty teams.** Regulated SMB, gov/政企, healthcare, finance, EU data-residency, internal/air-gapped IT. Pain: "we need a real helpdesk but legal/compliance won't allow a SaaS." They evaluate on: can it run offline, who holds the data, can we audit it.
- **Segment B — Indie / SaaS founders.** Bootstrappers who want a support desk *for their own product* without paying Intercom/Zendesk per-seat, and who like that it's open-source and embeddable. Pain: "Intercom is $X00/mo, I just want a chat bubble + tickets I control." They evaluate on: 5-minute deploy, embeddable widget, BYO-AI cost.

**The business model logic:** Open-source (MIT) self-host = free and frictionless = top of funnel + trust. Money comes later from a **managed/hosted tier** and/or **a license/support tier** (and for indies, possibly per-deployment / lifetime deals). The free OSS install is the distribution engine; monetization rides on convenience and support, not on locking the core. *(Pricing/packaging is the #1 open decision — Section 13.)*

**Brand voice (non-negotiable):** Technical, honest, anti-lock-in, developer-credible. Show the code, name the trade-offs, never overclaim. Tone is "the self-hosted, data-sovereign answer to a SaaS-fatigued market" — confident about the wedge, candid about what it *doesn't* do (no social channels, single-instance, young). Honesty is the moat for the skeptical self-host buyer.

---

## Section 3 — Current state

**Team composition (marketing surface area):** Solo founder, π-shaped (deep engineering, developing marketing). Marketing owner = founder, tactical time only. → Implication: every chosen tactic must be **low-maintenance and compounding** (SEO assets, launches, community) rather than daily-grind channels (paid, high-frequency social).

**Marketing budget (current):** ~$0 cash. Real budget = founder time + AI tooling (these marketing skills + the product's own AI). Plan must be executable at $0 paid spend.

**Phase of SaaS growth:** **$0–10K ARR (pre-revenue).** Binding constraint = *distribution + proof*, not product. Dominant growth pattern at this phase = founder-led, content + launches + community; manual, unscalable, trust-building.

**What's already done (acknowledge, then build on):**
- Product: full helpdesk + 7 competitive-parity features shipped (web widget, AI auto-resolve, automations, macros, CSAT, teams, merge).
- A live, multilingual (7-lang) SEO landing with JSON-LD, sitemap, hreflang.
- Public GitHub repo (MIT) + GitHub Pages landing.
- Live production deployment (smartticket.superleo.app) + the chat widget dogfooded on the landing.
- A competitive-analysis doc with sharp positioning.

**What's in-flight (drafted, not shipped):** MCP tools for AI-agent operation of the helpdesk (just added). Intake-hardening for the widget.

**What's stuck (unstick this quarter):** **Zero distribution.** The product is excellent and invisible. No launch, no comparison pages, no community presence, no content cadence, no install analytics. *Pricing/monetization undefined.*

**Audit rubric snapshot (0–5):**

| # | Area | Score | Note |
|---|---|---|---|
| 1 | Positioning / category | 4 | Sharp wedge already articulated; just needs to be everywhere |
| 2 | ICP clarity | 4 | Two clear segments; messaging not yet tailored per segment |
| 3 | Website / conversion | 3 | Good landing; lacks comparison/migration pages, social proof, clear CTA to deploy |
| 4 | SEO foundation | 2 | Multilingual + schema in place; no content, no target keywords, no KB indexing |
| 5 | Content engine | 1 | None yet |
| 6 | Launch / PR | 0 | Never launched |
| 7 | Community / social | 1 | Repo exists; no presence on Reddit/HN/X/LinkedIn |
| 8 | Activation / onboarding | 2 | Deploys, but no guided first-run / "aha" path documented |
| 9 | Retention | 2 | Product is sticky (self-hosted), but no changelog/email/community loop |
| 10 | Referral / virality | 1 | "Powered by" + OSS stars latent, not wired |
| 11 | Email | 0 | No list, no founder newsletter |
| 12 | Analytics / measurement | 1 | No install telemetry, no funnel metrics |
| 13 | Monetization | 1 | OSS only; no paid tier defined |
| 14 | Integrations ecosystem | 1 | Importers/integrations are marketing assets, not built |
| 15 | Developer marketing | 2 | OSS + MCP is a great base; no DevRel motion |
| 16 | Brand voice consistency | 4 | Strong, consistent, credible |
| 17 | Trust / social proof | 1 | No stars/case studies/testimonials yet |

**Shape interpretation:** Strengths cluster in **product, positioning, and voice** (the hard parts are done). Gaps cluster in **distribution, content, launch, and proof** — i.e., everything that turns a great product into a known one. This is the classic "built it, didn't market it" profile. The plan is almost entirely a **distribution + trust** plan.

---

## Section 4 — Acquisition

**Current state:** Inbound = near zero (direct/GitHub only).

**The plan — pick the 4 channels that compound at $0 and fit an OSS/dev buyer:**
1. **Comparison & category SEO** — "SmartTicket vs Chatwoot / vs Zendesk / vs Intercom / vs Zammad", "open source Zendesk alternative", "self-hosted Intercom alternative", "self-hosted helpdesk", "data residency helpdesk" `[#11, #1, #3, #4]`. High-intent, evergreen, exactly where this buyer searches.
2. **Knowledge-base / problem-solution SEO** — publish guides the ICP googles: "self-host customer support", "GDPR-compliant helpdesk", "run a helpdesk on your own LLM", "Zendesk data residency". Dogfood: host them in SmartTicket's own KB `[#9, #6]`.
3. **Launches + community** — Product Hunt, Show HN, r/selfhosted, r/SaaS, r/opensource, Indie Hackers `[#77, #38, #80]`. One-time spikes that also seed backlinks + stars.
4. **Open-source distribution** — be present where self-hosters discover tools: awesome-selfhosted lists, AlternativeTo, libhunt, OSS directories `[#123, #76, #10]`.

**90-day acquisition moves:**
- Ship 4 comparison pages + 1 "open-source Zendesk alternative" pillar.
- Submit to awesome-selfhosted, AlternativeTo, libhunt, 5+ directories `[directory-submissions skill]`.
- Execute the launch (Section 9).
- Start a founder content cadence (1 substantial post/week: build-in-public + the data-sovereignty/BYO-AI thesis) on X + LinkedIn + Indie Hackers `[#39, #41, #6]`.

**12-month acquisition outlook:** Comparison/KB SEO becomes the compounding base; launches punctuate (PH → Show HN → lifetime deal later); "powered-by" + integration pages add a product-led loop; developer marketing (MCP, API, importers) opens the dev channel.

**Skills + tools:** `seo-audit`, `ai-seo`, `programmatic-seo`, `competitor-profiling`, `competitors`, `directory-submissions`, `content-strategy`, `copywriting`. Tools: DataForSEO/Ahrefs/Semrush CLIs (keyword research), Firecrawl/Exa (competitor + SERP scraping), GSC.

---

## Section 5 — Activation

**The "aha":** *A self-hoster has SmartTicket running and watches the AI resolve (or draft) a real ticket from their own knowledge base — on their own model.* That moment proves all three wedge claims at once.

**The plan:**
- **5-minute deploy is the activation funnel.** Make the path from "git clone / docker run" → "first ticket + AI reply" frictionless and *documented as a guided first-run*. A quickstart that ends at the aha, not at a running binary.
- **Seed data + a demo mode** so a new installer sees the widget + AI working in 60 seconds without wiring an LLM.
- **The hosted demo (smartticket.superleo.app) is the zero-install activation** for evaluators who won't deploy first — make "try the live demo" a primary landing CTA.
- Instrument activation: opt-in anonymous telemetry for install → first-ticket → first-AI-reply (privacy-respecting, off by default, transparent — consistent with the brand).

**Skills + tools:** `onboarding`, `signup`, `free-tools`. Tools: the product's own analytics; Plausible (privacy-friendly) for the site.

---

## Section 6 — Retention

**Why it's already strong:** self-hosted = switching cost is high; the product runs on their infra. The risk isn't churn-out, it's **install-then-forget** (never reaching aha) and **silent abandonment** of the repo.

**The plan:**
- **Changelog + release cadence** (ship-and-tell). Each release = a reason to re-engage + a content unit `[#97, #8]`.
- **Founder newsletter** to anyone who stars/deploys/opts-in: monthly, build-in-public + tips `[#45]`.
- **Community home** (GitHub Discussions first; Discord only if demand) so installers help each other and stick `[#35]`.
- Dogfood retention features inside the product (CSAT, automations) and publish the results as proof.

**Skills + tools:** `churn-prevention`, `emails`, `community-marketing`, `onboarding`. Tools: Resend (already wired) for the newsletter; GitHub Discussions.

---

## Section 7 — Referral

**The latent loops, wired:**
- **"Powered by SmartTicket"** on the embeddable widget (optional, on by default for free tier) → every deployed widget is a distribution surface `[#88]`.
- **Open-source stars + social proof** — explicitly ask happy installers to star + share; surface star count and testimonials on the landing `[#137, #43]`.
- **Free migrations** — "Import from Zendesk/Freshdesk/Intercom" tooling doubles as a referral/switching magnet and a comparison-page CTA `[#16, #91]`.
- **Integration / co-marketing** — when SmartTicket integrates with a tool (LLM providers, Resend, etc.), co-market with them `[#54, #57]`.

**Skills + tools:** `referrals`, `co-marketing`, `product-marketing`. Tools: the widget; importer scripts.

---

## Section 8 — Revenue

**Current:** $0; OSS only. **This is the biggest open decision (Section 13).**

**Recommended model (to validate):**
- **Free:** self-host OSS (MIT) forever — the distribution engine.
- **Paid tier 1 — Managed/Hosted:** for teams who want the data-sovereignty story *without* running infra (single-tenant managed instance, EU/region of choice). Monthly per-instance, not per-seat (per-seat is the thing they're fleeing).
- **Paid tier 2 — License/Support:** priority support, SSO/SAML, SLA, white-glove migration for regulated/政企 buyers (who pay for support + compliance, not features).
- **Indie angle:** a one-time / lifetime / per-deployment license for founders who hate subscriptions (matches that ICP's psychology) `[pricing skill]`.

**Unit economics (unknown — must establish):** ARPC, CAC (here mostly *time*, not cash), retention. Flag all three in Section 13; every projection depends on them.

**Skills + tools:** `pricing`, `paywalls`, `revops`, `product-marketing`. Tools: Stripe/Paddle (when monetization starts).

---

## Section 9 — 90-day roadmap

**Weeks 1–2 — Unblock (decisions + foundation)**
- Decide the **monetization model** (at least v1 pricing page hypothesis) — unblocks Sections 8/10.
- Pick **10 target keywords** (comparison + category) via DataForSEO/Ahrefs.
- Set up **Plausible** + opt-in product telemetry; define the activation funnel metric.
- Tighten the landing: primary CTA = "Deploy in 5 min" + secondary "Try live demo"; add a comparison nav.

**Weeks 3–4 — Foundation (assets that compound)**
- Publish **4 comparison pages** (vs Chatwoot, Zendesk, Intercom, Zammad) + **1 pillar** ("The open-source, data-sovereign helpdesk") `[#11, #1]`.
- Write a **5-minute quickstart** that ends at the AI-resolves-a-ticket aha; add demo seed data.
- Submit to **awesome-selfhosted, AlternativeTo, libhunt + 5 directories** `[directory-submissions]`.
- Draft the **launch assets** (PH tagline/gallery/first comment, Show HN post, demo GIF of the widget + AI auto-resolve).

**Weeks 5–8 — Velocity (launch + content engine)**
- **LAUNCH:** Product Hunt + Show HN (same week, staggered) + r/selfhosted + Indie Hackers `[#77, #38, #80]`. Mobilize for stars + feedback.
- Start the **weekly founder post** (build-in-public + wedge thesis) on X/LinkedIn/IH `[#39]`.
- Stand up **GitHub Discussions** + a monthly **founder newsletter** (Resend) `[#35, #45]`.
- Ship **"Import from Zendesk"** as the first migration magnet `[#16, #91]`.

**Weeks 9–12 — Compound (measure + double down)**
- Publish 4 more KB/problem-solution SEO posts targeting the validated keywords `[#9, #6]`.
- Add **"Powered by" referral** + social-proof (stars, first testimonials) to the landing `[#88, #137]`.
- Review funnel metrics (installs → activation → stars → inbound); kill what didn't move, double the winner.
- First **pricing experiment** (managed-tier waitlist or "contact for hosted") to test willingness-to-pay.

---

## Section 10 — 12-month outlook

**Framing:** Year 1 goal is not revenue — it's **becoming the known answer to "self-hosted AI helpdesk"** with a compounding SEO base, a launch track record, and the first paying validation. Sequence the move from $0 → first $10K ARR by stacking distribution, then turning on monetization once activation + audience exist.

**Q1 (M1–3) — Foundation + launch.** Comparison/category SEO live; first launch executed; content + community cadence started; activation instrumented; pricing hypothesis set. *Target: launch done, baseline traffic + stars established, activation funnel measured.*

**Q2 (M4–6) — Engine + first revenue test.** Double the SEO content; ship 2–3 importers/integrations `[#16, #54]`; second launch beat (e.g. a lifetime deal or a "v2" Show HN) `[#80]`; open the **managed/hosted tier** to a waitlist → first paid pilots. *Target: repeatable inbound, first paying customers, pricing validated.*

**Q3 (M7–9) — Developer + ecosystem.** Lean into developer marketing: MCP/API/importers as DevRel, certification/quickstarts, "build on SmartTicket" `[#133, #22, #54]`. Integration co-marketing with LLM providers / tools `[#57]`. *Target: dev channel contributing installs; ecosystem flywheel starting.*

**Q4 (M10–12) — Compound + expand.** Annual "state of self-hosted support" data report from aggregate (opt-in) usage `[#6, #97]`; international (the multilingual SEO already exists — activate it) `[#131]`; scale the winning channels. *Target: SEO base compounding, multiple revenue tiers live, a real pipeline.*

---

## Section 11 — Marketing operations stack (skills → AARRR)

| AARRR stage | What executes it | marketing skills | Tools / MCP |
|---|---|---|---|
| **Acquisition** | Comparison/category + KB SEO, launches, directories, founder content | `seo-audit`, `ai-seo`, `programmatic-seo`, `competitor-profiling`, `competitors`, `directory-submissions`, `content-strategy`, `copywriting`, `launch`, `social`, `ad-creative` | DataForSEO/Ahrefs/Semrush, Firecrawl/Exa, GSC, GA4/Plausible |
| **Activation** | 5-min quickstart, demo mode, hosted demo, onboarding instrumentation | `onboarding`, `signup`, `free-tools`, `cro` | Product analytics, Plausible |
| **Retention** | Changelog cadence, founder newsletter, community | `churn-prevention`, `emails`, `community-marketing` | Resend, GitHub Discussions |
| **Referral** | Powered-by, OSS stars, migrations, co-marketing | `referrals`, `co-marketing`, `product-marketing` | Widget, importer scripts |
| **Revenue** | Pricing/packaging, managed tier, license/support | `pricing`, `paywalls`, `revops` | Stripe/Paddle |
| **Cross-cutting** | Positioning, ICP, messaging, customer language | `product-marketing`, `customer-research`, `copy-editing`, `marketing-psychology` | — |

**Thesis:** the founder is the whole marketing team, so the stack is "AI skills do the labor, founder does the judgment + the human/community parts (launch, replies, relationships)."

---

## Section 12 — Tactical idea bank (139 ideas → SmartTicket status)

Status: **Now (Q1)** · **Q2** · **Q3+** · **Q4+** · **Skip** (off-brand/no-fit).

**Acquisition / Content & SEO (1–10):** Now → `#1 Easy keyword ranking`, `#3 Glossary marketing` (helpdesk/ITSM terms), `#4 Programmatic SEO` (comparison/integration templates), `#6 Proprietary data content`, `#9 Knowledge base SEO`. Q2 → `#2 SEO audit (public)`, `#5 Repurposing`, `#7 Internal linking`, `#8 Content refreshing`, `#10 Parasite SEO`.
**Competitor (11–13):** Now → `#11 Comparison pages`, `#12 Marketing jiu-jitsu` (SaaS price hikes / data-residency fear → our affordability + sovereignty). Q2 → `#13 Competitive ad research`.
**Free tools (14–22):** Q2 → `#16 Importers as marketing` (Zendesk/Freshdesk), `#22 Public APIs` (already have). Q3+ → `#15 Engineering as marketing` (e.g. a "helpdesk SaaS cost calculator" `#18`, a "is your support data leaving the EU?" scanner `#21`). 
**Paid ads (23–34):** **Skip** at $0 budget — revisit `#28 LinkedIn`, `#29 Reddit`, `#31 Google (brand+competitor terms)` only after revenue.
**Social & community (35–44):** Now → `#35 Community marketing` (GitHub Discussions), `#38 Reddit marketing` (r/selfhosted), `#39 LinkedIn audience`, `#41 X audience`, `#44 Comment marketing`. Q2 → `#37 Reddit keyword research`, `#42 short-form video` (deploy-in-5-min demo).
**Email (45–53):** Now → `#45 Founder emails/newsletter`. Q2 → `#46 onboarding sequences`, `#53 win-back`.
**Partnerships (54–64):** Q2 → `#54 Affiliate/integration marketing`, `#57 Integration marketing` (LLM providers, Resend). Q3+ → `#60 Newsletter swaps`.
**Events (65–72):** Q3+ → `#65 Webinars` ("self-hosting a compliant helpdesk"). Mostly later.
**PR & Media (73–76):** Q2 → `#76 review sites` (AlternativeTo, G2 when eligible).
**Launches (77–86):** Now → `#77 Product Hunt`. Q2 → `#80 Lifetime deal` (indie ICP). Q3+ → `#82 giveaways`.
**Product-led (87–96):** Now → `#88 Powered-by marketing` (widget). Q2 → `#91 Free migrations`, `#87 viral loops`. 
**Content formats (97–109):** Q2 → `#97 Changelog/release notes as content`. Q4 → `#104 Annual report` (state of self-hosted support).
**Developer (133–136):** Q3+ → `#133 DevRel`, `#134 certifications/quickstarts`, MCP/API as dev surface.
**International (131–132):** Q4 → `#131 Expansion` (multilingual SEO already built — just activate).
**Audience-specific (137–139):** Now → `#139 Customer language` (mine Reddit/competitor reviews for exact pain phrasing → feeds all copy). Q2 → `#137 Referrals`.

**Explicit Skips (off-brand or no-fit at this stage):** most paid ads (no budget), influencer/giveaway-heavy plays, anything that compromises the "honest, technical, no-overclaim" voice.

---

## Section 13 — Measurement, RACI, open decisions, appendix

**North star (proposed):** **Activated self-hosted installs** (install → first AI-assisted ticket resolved) — it captures distribution *and* the wedge proof in one number. Secondary: GitHub stars (trust proxy) and, once live, managed-tier MRR.

**Leading indicators by stage:**

| Stage | Leading indicators |
|---|---|
| Acquisition | Landing sessions, comparison-page impressions/clicks (GSC), directory referrals, launch traffic, new stars |
| Activation | Install → first-ticket rate, first-AI-reply rate, live-demo → deploy rate |
| Retention | Returning installs, newsletter opens, Discussions activity, release re-engagement |
| Referral | Powered-by clickthroughs, star/share rate, migration-tool runs |
| Revenue | Managed-tier waitlist signups, pilot conversions, ARPC |

**RACI (solo reality):** Founder = R/A on everything; the **AI marketing skills = the "team" doing the labor** (drafting SEO, comparison pages, launch copy, newsletters); founder owns judgment, community, and relationships (launch day replies, partnerships).

**Open decisions (ranked):**
1. **Monetization model + v1 pricing** — blocks Sections 8 & 10. Decide before Q1 ends.
2. **Telemetry** — will you ship opt-in anonymous install analytics? Without it, the north star is unmeasurable.
3. **Hosted tier scope** — managed single-tenant: which regions, what's in/out vs OSS.
4. **Capacity** — solo founder can't run all channels; the plan assumes ~1 compounding asset/week. Confirm realistic cadence.
5. **Intake hardening** before pushing the widget hard (domain allowlist / abuse) — small eng task, gates the "embed everywhere" referral loop.

**Appendix / source docs:** `docs/competitive-analysis.md` (positioning + competitor matrix), the product itself (feature truth), `marketing-ideas` library (the 139 ideas referenced above). Channel execution lives in the per-skill docs (`launch`, `seo-audit`, `copywriting`, `pricing`, etc.).

---

*Next step options: (a) start executing Q1 — I can draft the 4 comparison pages + the launch assets next; (b) revise any section; (c) deep-dive one channel with its dedicated skill (e.g. `launch` for Product Hunt/Show HN assets, or `copywriting` to rewrite the landing).*
