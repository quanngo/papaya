-- 004: Claims submitted through the runtime processClaim API.
--
-- Each row is one POST /claims/process call: the input claim plus the outcome
-- the engine computed from the tenant's config *at that moment*
-- (tenant_id + config_version pins the exact config used, so the result stays
-- explainable after the tenant's config changes or is rolled back).
-- Preview (POST /claims/preview) never writes here.
--
-- The headline outcome is denormalised into columns for listing/filtering;
-- the full ProcessResult (documents, notifications, custom-field checks, trace)
-- is kept as JSON in `result`.

CREATE TABLE IF NOT EXISTS claim (
  id              TEXT    PRIMARY KEY,                          -- UUID
  tenant_id       TEXT    NOT NULL REFERENCES tenant (id),
  config_version  INTEGER NOT NULL,
  claim_type      TEXT    NOT NULL
                  CHECK (claim_type IN ('OUTPATIENT', 'INPATIENT', 'DENTAL', 'MATERNITY', 'OPTICAL')),
  amount          REAL    NOT NULL CHECK (amount >= 0),
  submitted_at    TEXT    NOT NULL,
  custom_fields   TEXT    NOT NULL DEFAULT '{}' CHECK (json_valid(custom_fields) AND json_type(custom_fields) = 'object'),
  documents       TEXT    NOT NULL DEFAULT '[]' CHECK (json_valid(documents) AND json_type(documents) = 'array'),

  -- outcome
  accepted        INTEGER NOT NULL CHECK (accepted IN (0, 1)),
  routing         TEXT    NOT NULL
                  CHECK (routing IN ('auto_approved', 'manual_review', 'rejected')),
                  -- rejected = claim not accepted (type not enabled, invalid custom fields...)
  approver_role   TEXT,                                         -- tier role when manual_review
  sla_deadline    TEXT,                                         -- NULL when rejected
  result          TEXT    NOT NULL CHECK (json_valid(result) AND json_type(result) = 'object'),

  created_by      INTEGER REFERENCES app_user (id),
  created_at      TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),

  FOREIGN KEY (tenant_id, config_version) REFERENCES tenant_config_version (tenant_id, version),
  CHECK ((routing = 'rejected') = (accepted = 0)),
  CHECK ((routing = 'manual_review') = (approver_role IS NOT NULL)),
  CHECK (routing = 'rejected' OR sla_deadline IS NOT NULL)
);

CREATE INDEX IF NOT EXISTS idx_claim_tenant_created ON claim (tenant_id, created_at DESC);

-- Open claims ordered by deadline: what an SLA-escalation job would scan.
CREATE INDEX IF NOT EXISTS idx_claim_sla_open ON claim (sla_deadline) WHERE routing = 'manual_review';
