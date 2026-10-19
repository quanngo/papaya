import i18n from './index'

// The single place that maps the UI language to an Intl locale.
const INTL_LOCALE = { vi: 'vi-VN', en: 'en-US' }

const intlLocale = () => INTL_LOCALE[i18n.language] || INTL_LOCALE.vi

/** 15000 -> "15.000" (vi) / "15,000" (en) */
export function formatNumber(value) {
  if (value === null || value === undefined || value === '') return '—'
  const n = Number(value)
  return Number.isFinite(n) ? new Intl.NumberFormat(intlLocale(), { maximumFractionDigits: 2 }).format(n) : String(value)
}

/** Money amounts: the challenge has no currency setting, so they are plain numbers. */
export const formatMoney = formatNumber

/** ISO/RFC 3339 -> localized date */
export function formatDate(iso) {
  if (!iso) return '—'
  const d = new Date(iso)
  return Number.isNaN(d.getTime()) ? iso : new Intl.DateTimeFormat(intlLocale(), { dateStyle: 'medium' }).format(d)
}

/** ISO/RFC 3339 -> localized date and time, optionally in a given IANA time zone */
export function formatDateTime(iso, timeZone) {
  if (!iso) return '—'
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  try {
    return new Intl.DateTimeFormat(intlLocale(), { dateStyle: 'medium', timeStyle: 'short', timeZone }).format(d)
  } catch {
    return new Intl.DateTimeFormat(intlLocale(), { dateStyle: 'medium', timeStyle: 'short' }).format(d)
  }
}
