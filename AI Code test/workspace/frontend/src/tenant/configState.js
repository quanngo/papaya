/**
 * Helpers for editing a TenantConfig immutably and matching backend issue
 * paths (e.g. "approval.tiers[1].min_amount") to form fields.
 */

/** "approval.tiers[1].min_amount" -> ['approval', 'tiers', 1, 'min_amount'] */
export function parsePath(path) {
  const out = []
  for (const part of path.split('.')) {
    const m = part.match(/^([^[\]]+)((?:\[\d+\])*)$/)
    if (!m) {
      out.push(part)
      continue
    }
    out.push(m[1])
    for (const idx of m[2].matchAll(/\[(\d+)\]/g)) out.push(Number(idx[1]))
  }
  return out
}

/** Returns a copy of obj with value set at path (string or array of keys). */
export function setIn(obj, path, value) {
  const keys = Array.isArray(path) ? path : parsePath(path)
  if (keys.length === 0) return value
  const [head, ...rest] = keys
  const base = obj ?? (typeof head === 'number' ? [] : {})
  const copy = Array.isArray(base) ? [...base] : { ...base }
  copy[head] = setIn(base[head], rest, value)
  return copy
}

export function getIn(obj, path) {
  const keys = Array.isArray(path) ? path : parsePath(path)
  return keys.reduce((acc, k) => (acc == null ? undefined : acc[k]), obj)
}

/** Issues whose path is exactly `path`, or starts with it when `nested` is true. */
export function issuesAt(issues, path, nested = false) {
  return (issues || []).filter((i) => i.path === path || (nested && (i.path.startsWith(path + '.') || i.path.startsWith(path + '['))))
}

/** Number input -> number; empty input -> fallback (0 or null). */
export function toNumber(raw, fallback = 0) {
  if (raw === '' || raw === null || raw === undefined) return fallback
  const n = Number(raw)
  return Number.isFinite(n) ? n : fallback
}

/**
 * Keeps tiers contiguous as the user edits: the first tier starts at the
 * auto-approval threshold and each tier starts where the previous one ends.
 */
export function chainTiers(approval) {
  const tiers = (approval.tiers || []).map((tier, i, all) => {
    if (i === 0) return { ...tier, min_amount: approval.auto_approval_threshold }
    const prevMax = all[i - 1].max_amount
    return prevMax === null || prevMax === undefined ? tier : { ...tier, min_amount: prevMax }
  })
  return { ...approval, tiers }
}

/** Enables/disables a claim type, keeping meta order and a documents entry for enabled types. */
export function toggleClaimType(config, type, on, order) {
  const current = new Set(config.claim_types.enabled || [])
  if (on) current.add(type)
  else current.delete(type)
  const enabled = order.filter((t) => current.has(t))
  const documents = { ...(config.claim_types.documents || {}) }
  if (on && !documents[type]) documents[type] = { required: ['invoice'], optional: [] }
  return { ...config, claim_types: { ...config.claim_types, enabled, documents } }
}

/** Fills nullable lists/maps so the form never reads undefined. */
export function normalizeConfig(config) {
  const c = structuredClone(config)
  c.claim_types.enabled ||= []
  c.claim_types.documents ||= {}
  for (const k of Object.keys(c.claim_types.documents)) {
    c.claim_types.documents[k].required ||= []
    c.claim_types.documents[k].optional ||= []
  }
  c.approval.tiers ||= []
  c.notifications.events ||= {}
  for (const k of Object.keys(c.notifications.events)) c.notifications.events[k].channels ||= []
  c.sla.targets ||= {}
  c.sla.escalation ||= []
  for (const e of c.sla.escalation) e.notify ||= []
  c.custom_fields ||= []
  return c
}
