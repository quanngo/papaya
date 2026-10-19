import { describe, expect, it } from 'vitest'
import i18n from '../i18n'
import { describePath, describeValue, flatten } from './humanize'

const t = i18n.t.bind(i18n)
const path = (p) => {
  const d = describePath(p, t)
  return d.label ? `${d.section}: ${d.label}` : d.section
}

describe('humanize', () => {
  it('turns config paths into plain words', () => {
    expect(path('approval.auto_approval_threshold')).toBe('Quy tắc duyệt: Ngưỡng tự động duyệt')
    expect(path('approval.tiers[role=assessor].max_amount')).toBe('Quy tắc duyệt: Cấp duyệt "Chuyên viên thẩm định" › Đến dưới')
    expect(path('claim_types.documents.OPTICAL.required')).toBe('Loại yêu cầu & chứng từ: Chứng từ cho Nhãn khoa › Bắt buộc')
    expect(path('notifications.events.claim_submitted.channels')).toBe('Thông báo: Đã nộp yêu cầu › Kênh thông báo')
    expect(path('sla.targets.OUTPATIENT')).toBe('SLA: Thời gian xử lý – Ngoại trú')
    expect(path('sla.escalation[0].notify')).toBe('SLA: Quy tắc leo thang 1 › Thông báo cho')
    expect(path('custom_fields[key=policy_number]')).toBe('Trường bổ sung: Trường "policy_number"')
    expect(path('branding.primary_color')).toBe('Thương hiệu: Màu chính')
  })

  it('formats values for people, not machines', () => {
    expect(describeValue('approval.auto_approval_threshold', 20000, t)).toBe('20.000')
    expect(describeValue('approval.tiers[role=x].max_amount', null, t)).toBe('Không giới hạn')
    expect(describeValue('approval.tiers[1].max_amount', undefined, t)).toBe('Chưa có') // tier missing on this side
    expect(describeValue('sla.targets.OUTPATIENT', 5, t)).toBe('5 ngày làm việc')
    expect(describeValue('claim_types.enabled', ['OUTPATIENT', 'DENTAL'], t)).toBe('Ngoại trú, Nha khoa')
    expect(describeValue('notifications.events.approved.channels', ['email', 'sms'], t)).toBe('Email, SMS')
    expect(describeValue('custom_fields[0].required', true, t)).toBe('Có')
    expect(describeValue('approval.tiers[role=director]', { min_amount: 500000, max_amount: null, role: 'director' }, t)).toBe('Giám đốc: từ 500.000 trở lên')
    expect(describeValue('claim_types.documents.OPTICAL', { required: ['invoice'], optional: [] }, t)).toBe('Bắt buộc: Hóa đơn · Tùy chọn: Không có')
    expect(describeValue('custom_fields[key=x]', { key: 'x', label: 'Policy Number', type: 'text', required: true }, t)).toBe('Policy Number (Văn bản, bắt buộc)')
    expect(describeValue('sla.escalation', { after_breach_days: 1, notify: ['team_lead'] }, t)).toBe('Sau 1 ngày quá hạn → báo Trưởng nhóm')
  })

  it('keeps lists of plain values as one row in the full view', () => {
    const rows = flatten({ claim_types: { enabled: ['OUTPATIENT'] }, approval: { tiers: [{ role: 'a', min_amount: 0, max_amount: null }] } })
    expect(rows.get('claim_types.enabled')).toEqual(['OUTPATIENT'])
    expect(rows.get('approval.tiers[0].role')).toBe('a')
    expect(rows.has('approval.tiers[0].max_amount')).toBe(true)
  })
})

describe('readable codes', () => {
  it('names known documents and roles, tidies unknown ones', async () => {
    const { docLabel, roleLabel, timezoneLabel } = await import('./labels')
    expect(docLabel(t, 'medical_report')).toBe('Báo cáo y tế')
    expect(docLabel(t, 'policy_check')).toBe('Policy check')
    expect(roleLabel(t, 'team_lead')).toBe('Trưởng nhóm')
    expect(roleLabel(t, 'Chuyên viên A')).toBe('Chuyên viên A') // free text stays as typed
    expect(timezoneLabel(t, 'Asia/Ho_Chi_Minh')).toBe('Giờ Việt Nam (GMT+7)')
  })
})
