import { apiRequest } from './client'

/**
 * @typedef {Object} SessionUser
 * @property {number} id
 * @property {string} email
 * @property {string} display_name
 * @property {string} auth_method
 * @property {{ code: string, name: string }} role
 * @property {string[]} permissions
 * @property {Object<string, string>} permission_names
 * @property {'vi'|'en'} locale
 */

/** @returns {Promise<SessionUser>} */
export const login = (email, password, rememberMe) =>
  apiRequest('/auth/login', { method: 'POST', body: { email, password, remember_me: rememberMe } })

export const logout = () => apiRequest('/auth/logout', { method: 'POST' })

/** @returns {Promise<SessionUser>} */
export const fetchMe = () => apiRequest('/auth/me')

/** @returns {Promise<SessionUser>} */
export const saveLocale = (locale) => apiRequest('/auth/me/locale', { method: 'PUT', body: { locale } })
