import BrandingSection from './BrandingSection'
import ClaimTypesSection from './ClaimTypesSection'
import ApprovalSection from './ApprovalSection'
import NotificationsSection from './NotificationsSection'
import SlaSection from './SlaSection'
import CustomFieldsSection from './CustomFieldsSection'

/**
 * @typedef {Object} SectionProps
 * @property {import('../../api/tenants').TenantConfig} config
 * @property {(path: string, value: any) => void} update      set one value by path
 * @property {(fn: (config: any) => any) => void} replace    derive the whole config
 * @property {import('../../api/tenants').Issue[]} issues
 * @property {import('../../api/configs').TenantConfigMeta} meta
 */

/**
 * One entry per backend config module (same keys as component/tenantconfig).
 * A new config dimension = one section file + one line here.
 */
export const SECTIONS = [
  { key: 'branding', Component: BrandingSection },
  { key: 'claim_types', Component: ClaimTypesSection },
  { key: 'approval', Component: ApprovalSection },
  { key: 'notifications', Component: NotificationsSection },
  { key: 'sla', Component: SlaSection },
  { key: 'custom_fields', Component: CustomFieldsSection },
]
