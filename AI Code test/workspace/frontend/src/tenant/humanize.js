/**
 * Turns config paths and values from the diff API into words a non-technical
 * user understands, e.g.
 *   approval.tiers[role=assessor].max_amount  ->  "Approval tier "assessor" › To (<)"
 *   {min_amount: 10000, max_amount: null, role: 'assessor'} -> "assessor: from 10,000, no upper limit"
 * The raw path stays available to callers (e.g. as a tooltip) for admins.
 */
import { formatMoney, formatNumber } from '../i18n/format'
import { docLabel, roleLabel, timezoneLabel } from './labels'
import { patternText } from './pattern'

const CLAIM_TYPES = ['OUTPATIENT', 'INPATIENT', 'DENTAL', 'MATERNITY', 'OPTICAL']
const CHANNELS = ['email', 'sms', 'webhook']
const FIELD_TYPES = ['text', 'number', 'select']
const MONEY_FIELDS = ['min_amount', 'max_amount', 'auto_approval_threshold']
const DAY_FIELDS = ['default_days', 'after_breach_days']

/** "tiers[role=assessor]" -> { name: 'tiers', idKey: 'role', id: 'assessor' }; "escalation[0]" -> { name, index: 0 } */
function parseSegment(seg) {
  const m = seg.match(/^([^[]+)(?:\[(?:([a-z_]+)=)?([^\]]*)\])?$/)
  if (!m) return { name: seg }
  const [, name, idKey, raw] = m
  if (raw === undefined) return { name }
  return idKey ? { name, idKey, id: raw } : { name, index: Number(raw) }
}

const claimType = (t, code) => (CLAIM_TYPES.includes(code) ? t(`claimTypes.${code}`) : code)
const itemLabel = (t, seg, keyedKey, numberedKey) =>
  seg.id !== undefined ? t(keyedKey, { id: seg.id }) : t(numberedKey, { n: (seg.index ?? 0) + 1 })

const LEAF = {
  branding: (t, k) => t(`branding.${k}`, { defaultValue: k }),
  tier: { min_amount: 'approval.from', max_amount: 'approval.to', role: 'approval.role' },
  event: { enabled: 'notifications.enabled', channels: 'tenants.channels', template: 'notifications.template' },
  escalation: { after_breach_days: 'sla.afterBreachDays', notify: 'sla.notify' },
  field: { key: 'customFields.key', label: 'customFields.label', type: 'customFields.type', required: 'customFields.required', pattern: 'customFields.pattern', options: 'customFields.options' },
}
const leaf = (t, map, name) => (map[name] ? t(map[name]) : name)

/**
 * @returns {{ section: string, label: string }} section = config area, label = the field inside it ('' = the whole area)
 */
export function describePath(path, t) {
  const segs = path.split('.').map(parseSegment)
  const module = segs[0].name
  const section = t(`sections.${module}`, { defaultValue: module })
  const out = []
  const rest = segs.slice(1)

  switch (module) {
    case 'branding':
      if (rest[0]) out.push(LEAF.branding(t, rest[0].name))
      break
    case 'claim_types':
      if (rest[0]?.name === 'enabled') out.push(t('claimTypesSection.enabled'))
      if (rest[0]?.name === 'documents') {
        if (rest[1]) out.push(t('claimTypesSection.documentsFor', { type: claimType(t, rest[1].name) }))
        else out.push(t('diff.label.documents'))
        if (rest[2]) out.push(t(`claimTypesSection.${rest[2].name}`, { defaultValue: rest[2].name }))
      }
      break
    case 'approval':
      if (rest[0]?.name === 'auto_approval_threshold') out.push(t('approval.threshold'))
      if (rest[0]?.name === 'tiers') {
        const tier = rest[0].id !== undefined ? { ...rest[0], id: roleLabel(t, rest[0].id) } : rest[0]
        out.push(tier.id !== undefined || tier.index !== undefined ? itemLabel(t, tier, 'diff.label.tierRole', 'diff.label.tierN') : t('diff.label.tiers'))
        if (rest[1]) out.push(leaf(t, LEAF.tier, rest[1].name))
      }
      break
    case 'notifications':
      if (rest[0]?.name === 'webhook_url') out.push(t('notifications.webhookUrl'))
      if (rest[0]?.name === 'events') {
        if (rest[1]) out.push(t(`events.${rest[1].name}`, { defaultValue: rest[1].name }))
        if (rest[2]) out.push(leaf(t, LEAF.event, rest[2].name))
      }
      break
    case 'sla':
      if (rest[0]?.name === 'timezone') out.push(t('sla.timezone'))
      if (rest[0]?.name === 'default_days') out.push(t('sla.defaultDays'))
      if (rest[0]?.name === 'targets') out.push(rest[1] ? t('diff.label.targetFor', { type: claimType(t, rest[1].name) }) : t('sla.targets'))
      if (rest[0]?.name === 'escalation') {
        out.push(rest[0].index !== undefined || rest[0].id !== undefined ? itemLabel(t, rest[0], 'diff.label.escalationN', 'diff.label.escalationN') : t('sla.escalation'))
        if (rest[1]) out.push(leaf(t, LEAF.escalation, rest[1].name))
      }
      break
    case 'custom_fields':
      if (segs[0].id !== undefined || segs[0].index !== undefined) out.push(itemLabel(t, segs[0], 'diff.label.fieldKey', 'diff.label.fieldN'))
      if (rest[0]) out.push(leaf(t, LEAF.field, rest[0].name))
      break
    default:
      out.push(...rest.map((s) => s.name))
  }
  return { section, label: out.join(' › ') }
}

/** A config value in plain words. `path` gives context (money, days, codes). */
export function describeValue(path, value, t) {
  const field = parseSegment(path.split('.').pop() || '').name
  if (value === undefined) return t('diff.value.notSet') // the setting does not exist on this side
  if (value === null || value === '') {
    return field === 'max_amount' ? t('diff.value.unlimited') : t('diff.value.empty')
  }
  if (typeof value === 'boolean') return value ? t('diff.value.yes') : t('diff.value.no')
  if (typeof value === 'number') {
    if (MONEY_FIELDS.includes(field)) return formatMoney(value)
    if (DAY_FIELDS.includes(field) || path.startsWith('sla.targets')) return t('diff.value.days', { n: formatNumber(value) })
    return formatNumber(value)
  }
  if (typeof value === 'string') {
    if (CLAIM_TYPES.includes(value)) return t(`claimTypes.${value}`)
    if (CHANNELS.includes(value)) return t(`channels.${value}`)
    if (field === 'type' && FIELD_TYPES.includes(value)) return t(`fieldTypes.${value}`)
    if (path.startsWith('claim_types.documents')) return docLabel(t, value)
    if (field === 'role' || field === 'notify' || path.startsWith('sla.escalation')) return roleLabel(t, value)
    if (field === 'timezone') return timezoneLabel(t, value)
    if (field === 'pattern') return patternText(value, t)
    return value
  }
  if (Array.isArray(value)) {
    return value.length ? value.map((v) => describeValue(path, v, t)).join(', ') : t('diff.value.none')
  }
  return describeObject(value, t)
}

function describeObject(o, t) {
  const list = (a, label = (v) => describeValue('', v, t)) => (a?.length ? a.map(label).join(', ') : t('diff.value.none'))
  if ('role' in o && 'min_amount' in o) {
    const role = roleLabel(t, o.role)
    return o.max_amount === null || o.max_amount === undefined
      ? t('diff.value.tierOpen', { role, min: formatMoney(o.min_amount) })
      : t('diff.value.tier', { role, min: formatMoney(o.min_amount), max: formatMoney(o.max_amount) })
  }
  if ('required' in o && 'optional' in o && Array.isArray(o.required)) {
    const doc = (d) => docLabel(t, d)
    return t('diff.value.documents', { required: list(o.required, doc), optional: list(o.optional, doc) })
  }
  if ('after_breach_days' in o) return t('diff.value.escalation', { n: o.after_breach_days, notify: list(o.notify, (r) => roleLabel(t, r)) })
  if ('key' in o && 'type' in o) {
    const base = t(o.required ? 'diff.value.fieldRequired' : 'diff.value.field', { label: o.label || o.key, type: t(`fieldTypes.${o.type}`, { defaultValue: o.type }) })
    return o.pattern ? `${base} · ${patternText(o.pattern, t)}` : base
  }
  if ('enabled' in o && 'channels' in o) {
    return o.enabled ? t('diff.value.eventOn', { channels: list(o.channels) }) : t('diff.value.eventOff')
  }
  return JSON.stringify(o)
}

/** Flattens a config into leaf rows for the full side-by-side view (lists of plain values stay one row). */
export function flatten(value, prefix = '', out = new Map()) {
  if (Array.isArray(value)) {
    if (value.length === 0 || value.every((v) => v === null || typeof v !== 'object')) out.set(prefix, value)
    else value.forEach((v, i) => flatten(v, `${prefix}[${i}]`, out))
  } else if (value && typeof value === 'object') {
    const keys = Object.keys(value)
    if (keys.length === 0) out.set(prefix, null)
    keys.forEach((k) => flatten(value[k], prefix ? `${prefix}.${k}` : k, out))
  } else {
    out.set(prefix, value)
  }
  return out
}
