# Demo script: AI Challenge 15, Multi-Tenant Configuration Platform

Four flows, about 5 minutes: **create a tenant → configure it → process claims → compare configs**. Do them through the admin UI (A) or with one command against the API (B).

- **Live app:** **https://papaya-2ze0.onrender.com** (admin UI at `/`, Swagger at `/swagger/index.html`). The free host sleeps when idle, so the first request can take up to a minute. It resets to the seed data when it restarts.
- **Local:** `docker compose up --build` in this folder, then http://localhost:8080. Without Docker: `cd workspace/frontend && npm run build`, then `cd workspace/backend && STATIC_DIR=../frontend/dist go run .`
- **Accounts:** `operator@papaya.com` / `Operator@2026` manages tenants and processes claims; `viewer@papaya.com` / `Viewer@2026!` is read-only; `admin@papaya.com` / `123456` has everything.
- **Seed tenants:** SafeGuard Insurance (corporate), HealthFirst (retail), GovHealth (government), as specified in the challenge.

---

## A. Through the admin UI

Sign in as **operator@papaya.com**. Switch the language selector (bottom of the sidebar) to **English** if you prefer. You land on **Overview**: insurer and claim counts, quick actions, recent changes.

### 1. Create a tenant (no code change)
1. **Insurers → + Add insurer**. The form starts from a minimal valid config.
2. **Identifier (slug):** `carelink`. **Company name:** `CareLink Insurance`.
3. Click **Create insurer**. You get *"Insurer created (version 1)"*, and the new insurer appears in the **Insurers** list.

### 2. Configure it
Validation runs live on the server while you type. Problems show under the field and as a count on the section tab, and **Save** stays disabled until the config is valid.

1. **Approval rules:** set **Auto-approval threshold** to `-1`. The error *"The auto-approval threshold must be zero or more."* appears. Change it to `10000`; the first tier's *From* follows automatically.
2. Click **+ Add approval tier**. Set tier 1 *To* = `200000`, tier 2 *Role* = `medical_director`. Tiers stay contiguous: each starts where the previous ends, and the last has no upper bound.
3. **Claim types & documents:** tick **Optical** and add required documents `invoice`, `eye_prescription`.
4. **SLA:** set Outpatient = `3` and Optical = `3` business days. Leaving Optical empty shows *"No SLA is set for OPTICAL"*.
5. **Notifications:** tick **Webhook** for *Claim submitted*. Enter a **Webhook URL** (required once a webhook channel is used).
6. **Custom fields → + Add field:** key `policy_number`, label `Policy Number`, format `^CL-[0-9]{6}$`, required.
7. Optional: in **Preview with this configuration** (right panel), preview a claim against the *unsaved* config.
8. Add a version note and click **Save new version**, or press **Ctrl+S / ⌘S**. The insurer is now at **v2**.

### 3. Process claims: identical claim, different behaviour
1. Open **Claim preview** (sidebar → Operations).
2. Keep all insurers selected. Use claim type **Outpatient**, amount **15000**. Fill the custom fields the form asks for (Employee ID `EMP-12345`, Department `Finance`, Budget Code `FIN-2026`, Policy Number `CL-123456`).
3. Click **Preview**, which stores nothing, or **Process for real**, which runs `processClaim` and stores the claims. Each card shows:

   | Insurer | Approval | Notifications | Required documents | SLA |
   |---|---|---|---|---|
   | SafeGuard | **Auto-approved** (below 20,000) | email: submitted + approved | invoice, prescription, medical_report | 5 business days |
   | HealthFirst | **assessor** (tier 5,000–50,000) | email + SMS | invoice, prescription | 7 business days |
   | GovHealth | **committee** (threshold 0) | email + webhook | + referral_letter | 15 business days |
   | CareLink | **assessor** (tier 10,000–200,000) | email + webhook | invoice | 3 business days |

   Expand **Step-by-step explanation** on a card to see why each decision was made.
4. Change the claim type to **Dental** and preview again. GovHealth answers *Not accepted: "This insurer does not offer DENTAL"*.
5. **Processed claims** lists what was processed, with the config version used. Filter by outcome (Auto-approved / Manual review / Not accepted).

### 4. Compare configs
1. **Compare configs:** pick **A** = CareLink, **B** = SafeGuard. Differences are grouped by section and highlighted (changed / added / removed). Tick **Show full configurations side by side** for the complete view.
2. Pick A = CareLink **v1** and B = CareLink **current** to see exactly what step 2 changed.
3. **Insurers → CareLink → History:** select v1, check its diff against the current version, then **Restore v1**. This creates **v3** with v1's config; history is never rewritten.

---

## B. One command against the API

`workspace/backend/scripts/demo.sh` runs the same four flows through the REST API. It needs `curl` and `node`, and you can re-run it: each run creates a new tenant.

```bash
cd workspace/backend
scripts/demo.sh                 # local server on :8080
scripts/demo.sh https://papaya-2ze0.onrender.com      # deployed app
```

Real output from a fresh database:

```text
== 0. Sign in as the operations user (http://localhost:18082)
  operator@papaya.com (operator): tenants.read, tenants.manage, claims.read, claims.process

== 1. Create tenant 'carelink-125305' from the default config
  created carelink-125305 v1: types=OUTPATIENT, auto-approve below 0, tiers=assessor

== 2. Configure it
  2a. invalid save (negative threshold, OPTICAL enabled without documents or SLA) is rejected:
  CONFIG_INVALID
    - claim_types.documents.OPTICAL: No documents are defined for OPTICAL.
    - approval.auto_approval_threshold: The auto-approval threshold must be zero or more.
    - approval.tiers[0].min_amount: The first tier must start at the auto-approval threshold (-1).
    - sla.targets.OPTICAL: No SLA is set for OPTICAL.
  2b. valid save -> v2:
  saved v2: types=OUTPATIENT,OPTICAL, auto-approve below 10000, tiers=assessor / medical_director

== 3. Process claims: the same claim (OUTPATIENT 15,000, Friday) sent to 4 tenants
  tenant            routing        approver    notifications                             SLA due
  safeguard         auto_approved  -           claim_submitted:email approved:email      2026-10-23  (docs: invoice,prescription,medical_report)
  healthfirst       manual_review  assessor    claim_submitted:email+sms                 2026-10-27  (docs: invoice,prescription)
  govhealth         manual_review  committee   claim_submitted:email+webhook             2026-11-06  (docs: invoice,prescription,medical_report,referral_letter)
  carelink-125305   manual_review  assessor    claim_submitted:email+webhook             2026-10-21  (docs: invoice)
  a claim the tenant does not offer is rejected with a reason (DENTAL -> GovHealth):
  accepted=false, routing=rejected: This insurer does not offer DENTAL (enabled: OUTPATIENT, INPATIENT).

== 4. Compare configs
  4a. carelink-125305 v1 vs v2 (what step 2 changed):
    changed approval.auto_approval_threshold                  0 -> 10000
    added   approval.tiers[role=assessor].max_amount          null -> 200000
    changed approval.tiers[role=assessor].min_amount          0 -> 10000
    added   approval.tiers[role=medical_director]             null -> {"max_amount":null,"min_amount":200000,"role":"medical_director"}
    added   claim_types.documents.OPTICAL                     null -> {"optional":[],"required":["invoice","eye_prescription"]}
    added   claim_types.enabled                               null -> "OPTICAL"
    added   custom_fields                                     null -> [{"key":"policy_number","label":"Policy Number","pattern":"^CL-[0-9]{6}$","required":true,"type":"text"}]
    added   notifications.events.claim_submitted.channels     null -> "webhook"
    added   notifications.webhook_url                         null -> "https://hooks.carelink.example/claims"
    added   sla.escalation                                    null -> [{"after_breach_days":1,"notify":["medical_director"]}]
    added   sla.targets.OPTICAL                               null -> 3
    changed sla.targets.OUTPATIENT                            5 -> 3
  4b. carelink-125305 vs SafeGuard:
    31 differences, e.g.
    changed approval.auto_approval_threshold
    changed approval.tiers[role=assessor].max_amount
    changed approval.tiers[role=assessor].min_amount
    removed approval.tiers[role=medical_director]
    added   approval.tiers[role=team_lead]
    added   approval.tiers[role=director]
  4c. roll back to v1 (creates v3; history is never rewritten):
    v3 rollback (current) Demo rollback
    v2 update             Enable OPTICAL, 2 approval tiers, webhook, Policy Number
    v1 create             Onboard CareLink
```

The SLA dates assume a claim submitted on Friday 2026-10-16 09:00 (Asia/Ho_Chi_Minh): +5, +7, +15 and +3 business days, weekends skipped.

---

## What this demonstrates

| Evaluation criterion | Where to see it |
|---|---|
| The same claim behaves differently per tenant | A.3 table / B step 3 |
| Invalid configurations are rejected | A.2 live errors / B step 2a (`CONFIG_INVALID` with field paths) |
| Preview predicts the real outcome | Preview and *Process for real* call the same engine (`ProcessClaim`); the unsaved-config preview in A.2.7 |
| Diff finds every difference | A.4 / B step 4 (tiers matched by role, custom fields by key, lists compared as sets) |
| History and rollback | A.4.3 / B step 4c (append-only versions) |
| A 4th tenant needs zero code changes | A.1–A.3 / B steps 1–3 |
| Modular config dimensions | Each section is one module (`component/tenantconfig/mod_*.go` + `pages/tenant-sections/*`); see `CLAUDE.md` |
