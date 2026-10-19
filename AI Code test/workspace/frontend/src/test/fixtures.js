// Minimal copies of the backend seed tenants (workspace/backend/seed/*.json).

const docs = (required, optional = []) => ({ required, optional })
const events = (channels) => ({
  claim_submitted: { enabled: true, channels },
  approved: { enabled: true, channels },
  rejected: { enabled: true, channels },
  payment_sent: { enabled: true, channels },
})

export const CONFIGS = {
  safeguard: {
    branding: { company_name: 'SafeGuard Insurance', logo_url: '', primary_color: '#0B3D91', secondary_color: '#F2A900' },
    claim_types: {
      enabled: ['OUTPATIENT', 'INPATIENT', 'DENTAL'],
      documents: { OUTPATIENT: docs(['invoice', 'prescription', 'medical_report']), INPATIENT: docs(['invoice']), DENTAL: docs(['invoice']) },
    },
    approval: {
      auto_approval_threshold: 20000,
      tiers: [
        { min_amount: 20000, max_amount: 100000, role: 'assessor' },
        { min_amount: 100000, max_amount: 500000, role: 'team_lead' },
        { min_amount: 500000, max_amount: null, role: 'director' },
      ],
    },
    notifications: { events: events(['email']) },
    sla: { timezone: 'Asia/Ho_Chi_Minh', targets: { OUTPATIENT: 5, INPATIENT: 10, DENTAL: 5 }, escalation: [] },
    custom_fields: [{ key: 'employee_id', label: 'Employee ID', type: 'text', required: true, pattern: '^EMP-[0-9]{5}$' }],
  },
  healthfirst: {
    branding: { company_name: 'HealthFirst', logo_url: '', primary_color: '#00897B', secondary_color: '#FF7043' },
    claim_types: { enabled: ['OUTPATIENT', 'INPATIENT'], documents: { OUTPATIENT: docs(['invoice']), INPATIENT: docs(['invoice']) } },
    approval: {
      auto_approval_threshold: 5000,
      tiers: [
        { min_amount: 5000, max_amount: 50000, role: 'assessor' },
        { min_amount: 50000, max_amount: null, role: 'manager' },
      ],
    },
    notifications: { events: events(['email', 'sms']) },
    sla: { timezone: 'Asia/Ho_Chi_Minh', targets: { OUTPATIENT: 7, INPATIENT: 7 }, default_days: 7, escalation: [] },
    custom_fields: [],
  },
  govhealth: {
    branding: { company_name: 'GovHealth', logo_url: '', primary_color: '#B71C1C', secondary_color: '#37474F' },
    claim_types: { enabled: ['OUTPATIENT', 'INPATIENT'], documents: { OUTPATIENT: docs(['invoice', 'referral_letter']), INPATIENT: docs(['invoice']) } },
    approval: { auto_approval_threshold: 0, tiers: [{ min_amount: 0, max_amount: null, role: 'committee' }] },
    notifications: { webhook_url: 'https://hooks.govhealth.example/claims', events: events(['email', 'webhook']) },
    sla: { timezone: 'Asia/Ho_Chi_Minh', targets: { OUTPATIENT: 15, INPATIENT: 15 }, escalation: [] },
    custom_fields: [
      { key: 'department', label: 'Department', type: 'text', required: true },
      { key: 'budget_code', label: 'Budget Code', type: 'text', required: true, pattern: '^[A-Z]{3}-[0-9]{4}$' },
    ],
  },
}

export const META = {
  claim_types: ['OUTPATIENT', 'INPATIENT', 'DENTAL', 'MATERNITY', 'OPTICAL'],
  notification_events: ['claim_submitted', 'approved', 'rejected', 'payment_sent'],
  channels: ['email', 'sms', 'webhook'],
  custom_field_types: ['text', 'number', 'select'],
  modules: ['branding', 'claim_types', 'custom_fields', 'approval', 'sla', 'notifications'],
}

export const DEFAULTS = {
  branding: { company_name: 'New Insurer', logo_url: '', primary_color: '#1E88E5', secondary_color: '#FFC107' },
  claim_types: { enabled: ['OUTPATIENT'], documents: { OUTPATIENT: docs(['invoice']) } },
  approval: { auto_approval_threshold: 0, tiers: [{ min_amount: 0, max_amount: null, role: 'assessor' }] },
  notifications: { events: events(['email']) },
  sla: { timezone: 'Asia/Ho_Chi_Minh', targets: { OUTPATIENT: 5 }, escalation: [] },
  custom_fields: [],
}

const allPerms = ['users.read', 'users.manage', 'tenants.read', 'tenants.manage', 'claims.read', 'claims.process']
export const USERS = {
  'admin@papaya.com': { password: '123456', role: { code: 'admin', name: 'Quản trị viên' }, permissions: allPerms },
  'operator@papaya.com': { password: 'Operator@2026', role: { code: 'operator', name: 'Vận hành' }, permissions: ['tenants.read', 'tenants.manage', 'claims.read', 'claims.process'] },
  'viewer@papaya.com': { password: 'Viewer@2026!', role: { code: 'viewer', name: 'Chỉ xem' }, permissions: ['tenants.read', 'claims.read'] },
}
