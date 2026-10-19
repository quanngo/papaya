#!/usr/bin/env bash
# API demo of the four flows the challenge asks for, against a running server:
#   1. create a tenant      (onboarding = configuration only, no code change)
#   2. configure it         (an invalid save is rejected with field-level issues, then a valid save -> v2)
#   3. process claims       (ONE identical claim to 4 tenants -> 4 different outcomes, plus a rejected claim)
#   4. compare configs      (v1 vs v2 of the new tenant, the new tenant vs SafeGuard), then roll back
#
# Usage: scripts/demo.sh [base-url]        default http://localhost:8080
# Needs curl and node. Safe to re-run: each run creates a new tenant slug.
set -euo pipefail
BASE="${1:-http://localhost:8080}"
API="$BASE/api/v1"
JAR="$(mktemp)"; trap 'rm -f "$JAR"' EXIT
RUN="$(date +%H%M%S)"
SLUG="carelink-$RUN"

call() { # method path [json-body]  -> envelope JSON
  curl -sS -b "$JAR" -c "$JAR" -X "$1" "$API$2" -H 'Content-Type: application/json' ${3:+-d "$3"}
}
show() { # print a JS expression over d = data[0]; on an error envelope print its code and text
  node -e 'let s="";process.stdin.on("data",c=>s+=c).on("end",()=>{const e=JSON.parse(s);const d=e.data[0];
    if(e.code!==200&&!process.argv[2]){console.log("  !! "+e.code+" "+e.message+" - "+e.error_text);process.exit(1)}
    console.log(eval(process.argv[1]))})' "$@"
}
edit() { # read a TenantDetail envelope, apply a JS mutation to cfg, print an update body
  node -e 'let s="";process.stdin.on("data",c=>s+=c).on("end",()=>{const d=JSON.parse(s).data[0];const cfg=d.config;
    eval(process.argv[1]);console.log(JSON.stringify({config:cfg,note:process.argv[2],base_version:d.current_version}))})' "$@"
}
title() { printf '\n\033[1m== %s\033[0m\n' "$1"; }

title "0. Sign in as the operations user ($BASE)"
call POST /auth/login '{"email":"operator@papaya.com","password":"Operator@2026"}' \
  | show '"  "+d.email+" ("+d.role.code+"): "+d.permissions.join(", ")'
# Messages come back in the account's saved language; DEMO_LOCALE=vi keeps Vietnamese.
call PUT /auth/me/locale "{\"locale\":\"${DEMO_LOCALE:-en}\"}" >/dev/null

title "1. Create tenant '$SLUG' from the default config"
DEFAULTS="$(call GET /configs/defaults)"
BODY="$(echo "$DEFAULTS" | node -e 'let s="";process.stdin.on("data",c=>s+=c).on("end",()=>{const cfg=JSON.parse(s).data[0];
  cfg.branding.company_name="CareLink '"$RUN"'";
  console.log(JSON.stringify({slug:"'"$SLUG"'",config:cfg,note:"Onboard CareLink"}))})')"
call POST /tenants "$BODY" \
  | show '"  created "+d.slug+" v"+d.current_version+": types="+d.config.claim_types.enabled+", auto-approve below "+d.config.approval.auto_approval_threshold+", tiers="+d.config.approval.tiers.map(t=>t.role)'

title "2. Configure it"
echo "  2a. invalid save (negative threshold, OPTICAL enabled without documents or SLA) is rejected:"
BAD="$(call GET "/tenants/$SLUG" | edit 'cfg.approval.auto_approval_threshold=-1; cfg.claim_types.enabled.push("OPTICAL")' "bad")"
call PUT "/tenants/$SLUG" "$BAD" \
  | show '"  "+e.message+"\n"+d.issues.map(i=>"    - "+i.path+": "+i.text).join("\n")' allow-error
echo "  2b. valid save -> v2:"
GOOD="$(call GET "/tenants/$SLUG" | edit '
  cfg.claim_types.enabled=["OUTPATIENT","OPTICAL"];
  cfg.claim_types.documents.OPTICAL={required:["invoice","eye_prescription"],optional:[]};
  cfg.approval={auto_approval_threshold:10000,tiers:[
    {min_amount:10000,max_amount:200000,role:"assessor"},{min_amount:200000,max_amount:null,role:"medical_director"}]};
  cfg.notifications.webhook_url="https://hooks.carelink.example/claims";
  cfg.notifications.events.claim_submitted.channels=["email","webhook"];
  cfg.sla.targets={OUTPATIENT:3,OPTICAL:3};
  cfg.sla.escalation=[{after_breach_days:1,notify:["medical_director"]}];
  cfg.custom_fields=[{key:"policy_number",label:"Policy Number",type:"text",required:true,pattern:"^CL-[0-9]{6}$"}];
' "Enable OPTICAL, 2 approval tiers, webhook, Policy Number")"
call PUT "/tenants/$SLUG" "$GOOD" \
  | show '"  saved v"+d.current_version+": types="+d.config.claim_types.enabled+", auto-approve below "+d.config.approval.auto_approval_threshold+", tiers="+d.config.approval.tiers.map(t=>t.role).join(" / ")'

title "3. Process claims: the same claim (OUTPATIENT 15,000, Friday) sent to 4 tenants"
CLAIM='{"claim_type":"OUTPATIENT","amount":15000,"submitted_at":"2026-10-16T09:00:00+07:00",
  "custom_fields":{"employee_id":"EMP-12345","department":"Finance","budget_code":"FIN-2026","policy_number":"CL-123456"},
  "documents":["invoice","prescription"]}'
call POST /claims/process-batch "{\"tenant_ids\":[\"safeguard\",\"healthfirst\",\"govhealth\",\"$SLUG\"],\"claim\":$CLAIM}" \
  | show '"  "+"tenant".padEnd(18)+"routing".padEnd(15)+"approver".padEnd(12)+"notifications".padEnd(42)+"SLA due\n"+
      d.results.map(r=>"  "+r.tenant.slug.padEnd(18)+r.approval.routing.padEnd(15)+(r.approval.approver_role||"-").padEnd(12)
      +r.notifications.map(n=>n.event+":"+n.channels.join("+")).join(" ").padEnd(42)+r.sla.deadline.slice(0,10)
      +"  (docs: "+r.documents.required.join(",")+")").join("\n")'
echo "  a claim the tenant does not offer is rejected with a reason (DENTAL -> GovHealth):"
call POST /claims/process '{"tenant_id":"govhealth","claim":{"claim_type":"DENTAL","amount":1000,"custom_fields":{"department":"Finance","budget_code":"FIN-2026"}}}' \
  | show '"  accepted="+d.accepted+", routing="+d.approval.routing+": "+d.errors.map(e=>e.text).join("; ")'

title "4. Compare configs"
echo "  4a. $SLUG v1 vs v2 (what step 2 changed):"
call GET "/tenants/diff?a=$SLUG@1&b=$SLUG@2" \
  | show 'd.changes.map(c=>"    "+c.kind.padEnd(8)+c.path.padEnd(50)+JSON.stringify(c.left)+" -> "+JSON.stringify(c.right)).join("\n")'
echo "  4b. $SLUG vs SafeGuard:"
call GET "/tenants/diff?a=$SLUG&b=safeguard" \
  | show '"    "+d.changes.length+" differences, e.g.\n"+d.changes.filter(c=>c.module!=="branding").slice(0,6).map(c=>"    "+c.kind.padEnd(8)+c.path).join("\n")'
echo "  4c. roll back to v1 (creates v3; history is never rewritten):"
call POST "/tenants/$SLUG/rollback" '{"version":1,"note":"Demo rollback"}' >/dev/null
call GET "/tenants/$SLUG/versions" \
  | show 'd.versions.map(v=>"    v"+v.version+" "+v.source.padEnd(9)+(v.is_current?"(current) ":"          ")+v.note).join("\n")'

printf '\nDone. Open %s/tenants/%s (UI) or %s/swagger/index.html (API).\n' "$BASE" "$SLUG" "$BASE"
