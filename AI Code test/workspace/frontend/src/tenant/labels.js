/**
 * Readable names for codes that tenants put in their config: document codes
 * (invoice, medical_report…), approver roles (assessor, team_lead…) and time
 * zones. Known codes come from the locale catalogs; unknown ones are tidied
 * ("policy_check" -> "Policy check") so custom values still read well.
 */
import i18n from '../i18n'

function tidy(code) {
  if (typeof code !== 'string' || !/^[a-z0-9]+(_[a-z0-9]+)*$/.test(code)) return code
  const words = code.replace(/_/g, ' ')
  return words.charAt(0).toUpperCase() + words.slice(1)
}

const lookup = (prefix) => (t, code) => {
  if (!code) return code
  const key = `${prefix}.${code}`
  return i18n.exists(key) ? t(key) : tidy(code)
}

/** invoice -> "Hóa đơn" / "Invoice" */
export const docLabel = lookup('documentNames')
/** team_lead -> "Trưởng nhóm" / "Team lead" */
export const roleLabel = lookup('roleNames')

/** Asia/Ho_Chi_Minh -> "Giờ Việt Nam (GMT+7)"; unknown zones stay as written. */
export function timezoneLabel(t, zone) {
  if (!zone) return zone
  const key = `timezones.${zone.replace(/\//g, '_')}`
  return i18n.exists(key) ? t(key) : zone
}

export const TIMEZONES = ['Asia/Ho_Chi_Minh', 'Asia/Bangkok', 'Asia/Singapore', 'Asia/Jakarta', 'Asia/Manila', 'Asia/Hong_Kong', 'UTC']
