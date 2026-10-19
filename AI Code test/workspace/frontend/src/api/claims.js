import { apiRequest } from './client'

/**
 * @typedef {Object} Claim
 * @property {string} claim_type
 * @property {number} amount
 * @property {string} [submitted_at]   RFC 3339
 * @property {Object<string, any>} [custom_fields]
 * @property {string[]} [documents]
 *
 * @typedef {Object} ProcessResult
 * @property {string} [claim_id]          set when the claim was stored
 * @property {boolean} accepted
 * @property {import('./tenants').Issue[]} errors
 * @property {{ id: string, slug: string, name: string, config_version: number }} tenant
 * @property {string} claim_type
 * @property {number} amount
 * @property {{ required: string[], optional: string[], missing: string[] }} documents
 * @property {{ routing: 'auto_approved'|'manual_review'|'rejected', auto_approved: boolean, approver_role?: string, tier?: import('./tenants').Tier, threshold: number }} approval
 * @property {{ event: string, channels: string[], template: string, default_template: boolean }[]} notifications
 * @property {{ business_days: number, timezone: string, submitted_at: string, deadline: string, escalation: { after_breach_days: number, notify: string[] }[] }|null} sla
 * @property {{ key: string, label: string, type: string, required: boolean, value: any, valid: boolean, code?: string, text?: string }[]} custom_fields
 * @property {{ module: string, code: string, params?: Object, text: string }[]} trace
 */

/** Preview for a saved tenant or an unsaved config; nothing is stored. @returns {Promise<ProcessResult>} */
export const previewClaim = ({ tenantId, config, claim }) =>
  apiRequest('/claims/preview', { method: 'POST', body: { tenant_id: tenantId || undefined, config: config || undefined, claim } })

/** processClaim(tenantId, claimData): stored. @returns {Promise<ProcessResult>} */
export const processClaim = (tenantId, claim) => apiRequest('/claims/process', { method: 'POST', body: { tenant_id: tenantId, claim } })

/** The same claim against several tenants, each stored. @returns {Promise<ProcessResult[]>} */
export const processBatch = async (tenantIds, claim) =>
  (await apiRequest('/claims/process-batch', { method: 'POST', body: { tenant_ids: tenantIds, claim } })).results

export const listClaims = async (tenantId, limit = 50) => {
  const q = new URLSearchParams({ limit: String(limit) })
  if (tenantId) q.set('tenant_id', tenantId)
  return (await apiRequest(`/claims?${q}`)).claims
}
