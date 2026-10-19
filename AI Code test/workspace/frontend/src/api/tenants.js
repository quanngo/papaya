import { apiRequest } from './client'

/**
 * @typedef {Object} Tier
 * @property {number} min_amount
 * @property {number|null} max_amount   null = no upper bound (last tier only)
 * @property {string} role
 *
 * @typedef {Object} TenantConfig
 * @property {{ company_name: string, logo_url: string, primary_color: string, secondary_color: string }} branding
 * @property {{ enabled: string[], documents: Object<string, { required: string[], optional: string[] }> }} claim_types
 * @property {{ auto_approval_threshold: number, tiers: Tier[] }} approval
 * @property {{ webhook_url?: string, events: Object<string, { enabled: boolean, channels: string[], template?: string }> }} notifications
 * @property {{ timezone: string, targets: Object<string, number>, default_days?: number, escalation: { after_breach_days: number, notify: string[] }[] }} sla
 * @property {{ key: string, label: string, type: 'text'|'number'|'select', required: boolean, pattern?: string, options?: string[] }[]} custom_fields
 *
 * @typedef {Object} Tenant
 * @property {string} id
 * @property {string} slug
 * @property {string} name
 * @property {number} current_version
 * @property {string|null} archived_at
 * @property {string} created_at
 * @property {string} updated_at
 *
 * @typedef {Tenant & { config: TenantConfig }} TenantDetail
 *
 * @typedef {Object} TenantVersion
 * @property {number} version
 * @property {string} note
 * @property {'create'|'update'|'rollback'} source
 * @property {number|null} rolled_back_from
 * @property {string|null} created_by_email
 * @property {string} created_at
 * @property {boolean} is_current
 * @property {TenantConfig} [config]
 *
 * @typedef {Object} Issue
 * @property {string} path   e.g. approval.tiers[1].min_amount
 * @property {string} code   e.g. TIER_GAP
 * @property {Object} [params]
 * @property {string} text   translated by the backend in the user's language
 *
 * @typedef {Object} Change
 * @property {string} path
 * @property {string} module
 * @property {'added'|'removed'|'changed'} kind
 * @property {any} left
 * @property {any} right
 */

/** @returns {Promise<Tenant[]>} */
export const listTenants = async (includeArchived = false) =>
  (await apiRequest(`/tenants${includeArchived ? '?include_archived=1' : ''}`)).tenants

/** @returns {Promise<TenantDetail>} */
export const getTenant = (id) => apiRequest(`/tenants/${encodeURIComponent(id)}`)

/** @returns {Promise<TenantDetail>} */
export const createTenant = (slug, config, note) => apiRequest('/tenants', { method: 'POST', body: { slug, config, note } })

/** @returns {Promise<TenantDetail>} */
export const updateTenant = (id, config, note, baseVersion) =>
  apiRequest(`/tenants/${encodeURIComponent(id)}`, { method: 'PUT', body: { config, note, base_version: baseVersion } })

export const archiveTenant = (id) => apiRequest(`/tenants/${encodeURIComponent(id)}`, { method: 'DELETE' })

/** @returns {Promise<TenantVersion[]>} */
export const listVersions = async (id) => (await apiRequest(`/tenants/${encodeURIComponent(id)}/versions`)).versions

/** @returns {Promise<TenantVersion>} */
export const getVersion = (id, version) => apiRequest(`/tenants/${encodeURIComponent(id)}/versions/${version}`)

/** @returns {Promise<TenantDetail>} */
export const rollbackTenant = (id, version, note) =>
  apiRequest(`/tenants/${encodeURIComponent(id)}/rollback`, { method: 'POST', body: { version, note } })

/**
 * @param {string} a  "<id|slug>" or "<id|slug>@<version>"
 * @param {string} b
 * @returns {Promise<{ a: Object, b: Object, changes: Change[] }>}
 */
export const diffTenants = (a, b) => apiRequest(`/tenants/diff?a=${encodeURIComponent(a)}&b=${encodeURIComponent(b)}`)
