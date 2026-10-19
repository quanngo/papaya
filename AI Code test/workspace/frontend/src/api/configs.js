import { apiRequest } from './client'

/**
 * @typedef {Object} TenantConfigMeta
 * @property {string[]} claim_types
 * @property {string[]} notification_events
 * @property {string[]} channels
 * @property {string[]} custom_field_types
 * @property {string[]} modules
 */

/** @returns {Promise<TenantConfigMeta>} */
export const fetchMeta = () => apiRequest('/meta/tenant-config')

/** @returns {Promise<import('./tenants').TenantConfig>} */
export const fetchDefaults = () => apiRequest('/configs/defaults')

/** @returns {Promise<{ valid: boolean, issues: import('./tenants').Issue[] }>} */
export const validateConfig = (config, signal) => apiRequest('/configs/validate', { method: 'POST', body: { config }, signal })
