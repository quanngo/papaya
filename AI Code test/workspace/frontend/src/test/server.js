import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'
import { CONFIGS, DEFAULTS, META, USERS } from './fixtures'

/**
 * In-memory stand-in for the Go backend with the REAL envelope
 * ({code, message, error_text, data[]}, always HTTP 200). Tests flip the
 * toggles on `backend`; resetBackend() runs before each test.
 */
export const backend = {}

export function resetBackend() {
  backend.user = null // signed-in user (null = anonymous)
  backend.tenants = Object.entries(CONFIGS).map(([slug, config], i) => ({
    id: `id-${slug}`,
    slug,
    name: config.branding.company_name,
    current_version: 1,
    archived_at: null,
    created_at: '2026-10-17T08:00:00.000Z',
    updated_at: '2026-10-17T08:00:0' + i + '.000Z',
    versions: [{ version: 1, note: 'Seed data', source: 'create', rolled_back_from: null, created_by_email: null, created_at: '2026-10-17T08:00:00.000Z', config: structuredClone(config) }],
  }))
  backend.claims = []
  backend.claimsList = [] // rows returned by GET /claims
  backend.savedLocale = null
  backend.forceConflict = false
}
resetBackend()

const ok = (...data) => HttpResponse.json({ code: 200, message: 'OK', data })
const fail = (code, message, data) => HttpResponse.json({ code, message, error_text: `backend:${message}`, data: data === undefined ? [] : [data] })

const sessionUser = (email) => ({
  id: 1,
  email,
  display_name: email.split('@')[0],
  auth_method: 'password',
  role: USERS[email].role,
  permissions: USERS[email].permissions,
  permission_names: {},
  locale: backend.savedLocale || 'vi',
})

const findTenant = (ref) => backend.tenants.find((t) => t.id === ref || t.slug === ref)
const current = (t) => t.versions.find((v) => v.version === t.current_version).config
const publicTenant = ({ versions, ...t }) => t
const detail = (t) => ({ ...publicTenant(t), config: structuredClone(current(t)) })

// The few rules the tests exercise; the real validation lives in the backend.
function validate(config) {
  const issues = []
  if (config.approval.auto_approval_threshold < 0) issues.push({ path: 'approval.auto_approval_threshold', code: 'THRESHOLD_NEGATIVE', text: 'Ngưỡng tự động duyệt phải lớn hơn hoặc bằng 0.' })
  if (!config.claim_types.enabled.length) issues.push({ path: 'claim_types.enabled', code: 'NO_CLAIM_TYPE', text: 'Phải bật ít nhất một loại yêu cầu bồi thường.' })
  if (!config.branding.company_name.trim()) issues.push({ path: 'branding.company_name', code: 'COMPANY_NAME_REQUIRED', text: 'Tên công ty là bắt buộc.' })
  return issues
}

function engine(tenant, config, claim) {
  const accepted = config.claim_types.enabled.includes(claim.claim_type)
  const auto = accepted && claim.amount < config.approval.auto_approval_threshold
  const tier = accepted && !auto ? config.approval.tiers.find((t) => claim.amount >= t.min_amount && (t.max_amount === null || claim.amount < t.max_amount)) : null
  const days = config.sla.targets[claim.claim_type] || config.sla.default_days || 0
  return {
    accepted,
    errors: accepted ? [] : [{ path: 'claim.claim_type', code: 'CLAIM_TYPE_NOT_ENABLED', text: `Công ty không hỗ trợ loại ${claim.claim_type}.` }],
    tenant: { id: tenant?.id || '', slug: tenant?.slug || '', name: tenant?.name || config.branding.company_name, config_version: tenant ? tenant.current_version : 0 },
    claim_type: claim.claim_type,
    amount: claim.amount,
    documents: { required: config.claim_types.documents[claim.claim_type]?.required || [], optional: [], missing: [] },
    approval: { routing: !accepted ? 'rejected' : auto ? 'auto_approved' : 'manual_review', auto_approved: auto, approver_role: tier?.role, tier: tier || undefined, threshold: config.approval.auto_approval_threshold },
    notifications: accepted ? [{ event: 'claim_submitted', channels: config.notifications.events.claim_submitted.channels, template: 'x', default_template: true }] : [],
    sla: accepted ? { business_days: days, timezone: 'Asia/Ho_Chi_Minh', submitted_at: '2026-10-16T09:00:00+07:00', deadline: '2026-10-23T09:00:00+07:00', escalation: [] } : null,
    custom_fields: [],
    trace: [{ module: 'approval', code: 'X', text: `trace ${tenant?.slug || 'unsaved'}` }],
  }
}

const requireUser = () => (backend.user ? null : fail(401, 'UNAUTHENTICATED'))
const API = '*/api/v1'

export const handlers = [
  http.post(`${API}/auth/login`, async ({ request }) => {
    const { email, password } = await request.json()
    const u = USERS[email?.toLowerCase()]
    if (!u || u.password !== password) return fail(401, 'INVALID_CREDENTIALS')
    backend.user = email.toLowerCase()
    return ok(sessionUser(backend.user))
  }),
  http.post(`${API}/auth/logout`, () => {
    backend.user = null
    return ok()
  }),
  http.get(`${API}/auth/me`, () => requireUser() || ok(sessionUser(backend.user))),
  http.put(`${API}/auth/me/locale`, async ({ request }) => {
    backend.savedLocale = (await request.json()).locale
    return requireUser() || ok(sessionUser(backend.user))
  }),

  http.get(`${API}/meta/tenant-config`, () => requireUser() || ok(META)),
  http.get(`${API}/configs/defaults`, () => requireUser() || ok(structuredClone(DEFAULTS))),
  http.post(`${API}/configs/validate`, async ({ request }) => {
    const issues = validate((await request.json()).config)
    return requireUser() || ok({ valid: issues.length === 0, issues })
  }),

  http.get(`${API}/tenants`, ({ request }) => {
    const all = new URL(request.url).searchParams.get('include_archived') === '1'
    return requireUser() || ok({ tenants: backend.tenants.filter((t) => all || !t.archived_at).map(publicTenant) })
  }),
  http.post(`${API}/tenants`, async ({ request }) => {
    const { slug, config, note } = await request.json()
    if (requireUser()) return requireUser()
    const issues = validate(config)
    if (issues.length) return fail(400, 'CONFIG_INVALID', { valid: false, issues })
    if (findTenant(slug)) return fail(409, 'TENANT_SLUG_TAKEN')
    const t = { id: `id-${slug}`, slug, name: config.branding.company_name, current_version: 1, archived_at: null, created_at: 'x', updated_at: 'x', versions: [{ version: 1, note, source: 'create', rolled_back_from: null, created_by_email: backend.user, created_at: 'x', config }] }
    backend.tenants.push(t)
    return ok(detail(t))
  }),
  http.get(`${API}/tenants/diff`, ({ request }) => {
    const q = new URL(request.url).searchParams
    const side = (ref) => {
      const [id, v] = ref.split('@')
      const t = findTenant(id)
      return t && { tenant_id: t.id, slug: t.slug, name: t.name, version: Number(v) || t.current_version }
    }
    const a = side(q.get('a'))
    const b = side(q.get('b'))
    if (!a || !b) return fail(404, 'TENANT_NOT_FOUND')
    return ok({ a, b, changes: [{ path: 'approval.auto_approval_threshold', module: 'approval', kind: 'changed', left: 20000, right: 0 }, { path: 'claim_types.enabled', module: 'claim_types', kind: 'removed', left: 'DENTAL', right: null }] })
  }),
  http.get(`${API}/tenants/:id`, ({ params }) => {
    const t = findTenant(params.id)
    return requireUser() || (t ? ok(detail(t)) : fail(404, 'TENANT_NOT_FOUND'))
  }),
  http.put(`${API}/tenants/:id`, async ({ params, request }) => {
    const t = findTenant(params.id)
    const { config, note, base_version } = await request.json()
    if (backend.forceConflict || base_version !== t.current_version) return fail(409, 'VERSION_CONFLICT')
    t.current_version += 1
    t.versions.push({ version: t.current_version, note, source: 'update', rolled_back_from: null, created_by_email: backend.user, created_at: 'x', config })
    return ok(detail(t))
  }),
  http.delete(`${API}/tenants/:id`, ({ params }) => {
    findTenant(params.id).archived_at = 'x'
    return ok()
  }),
  http.get(`${API}/tenants/:id/versions`, ({ params }) => {
    const t = findTenant(params.id)
    return ok({ versions: [...t.versions].reverse().map(({ config, ...v }) => ({ ...v, tenant_id: t.id, is_current: v.version === t.current_version })) })
  }),
  http.get(`${API}/tenants/:id/versions/:v`, ({ params }) => {
    const t = findTenant(params.id)
    const v = t.versions.find((x) => x.version === Number(params.v))
    return v ? ok({ ...v, tenant_id: t.id, is_current: v.version === t.current_version }) : fail(404, 'VERSION_NOT_FOUND')
  }),
  http.post(`${API}/tenants/:id/rollback`, async ({ params, request }) => {
    const t = findTenant(params.id)
    const { version } = await request.json()
    const old = t.versions.find((x) => x.version === version)
    t.current_version += 1
    t.versions.push({ version: t.current_version, note: `Rollback to v${version}`, source: 'rollback', rolled_back_from: version, created_by_email: backend.user, created_at: 'x', config: old.config })
    return ok(detail(t))
  }),

  http.post(`${API}/claims/preview`, async ({ request }) => {
    const { tenant_id, config, claim } = await request.json()
    const t = tenant_id ? findTenant(tenant_id) : null
    return requireUser() || ok(engine(t, config || current(t), claim))
  }),
  http.post(`${API}/claims/process-batch`, async ({ request }) => {
    const { tenant_ids, claim } = await request.json()
    const results = tenant_ids.map((id) => {
      const t = findTenant(id)
      return { ...engine(t, current(t), claim), claim_id: `claim-${backend.claims.length}` }
    })
    backend.claims.push(...results)
    return ok({ results })
  }),
  http.get(`${API}/claims`, () => requireUser() || ok({ claims: backend.claimsList })),
]

export const server = setupServer(...handlers)
