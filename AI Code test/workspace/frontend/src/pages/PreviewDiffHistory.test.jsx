import { describe, expect, it } from 'vitest'
import { screen, waitFor, within } from '@testing-library/react'
import { renderApp } from '../test/renderApp'
import { backend } from '../test/server'

describe('claim preview', () => {
  it('shows a different outcome per tenant for the same claim', async () => {
    const { user } = renderApp('/preview', { as: 'operator@papaya.com' })
    await user.click(await screen.findByRole('button', { name: 'Xem trước (3 công ty)' }))
    const sg = await screen.findByRole('article', { name: 'SafeGuard Insurance' })
    expect(within(sg).getByText('Tự động duyệt')).toBeInTheDocument()
    expect(within(screen.getByRole('article', { name: 'HealthFirst' })).getByText('Chuyên viên thẩm định')).toBeInTheDocument()
    expect(within(screen.getByRole('article', { name: 'GovHealth' })).getByText('Hội đồng xét duyệt')).toBeInTheDocument()
    expect(backend.claims).toHaveLength(0) // preview stores nothing
  })

  it('asks for every selected tenant custom field', async () => {
    renderApp('/preview', { as: 'operator@papaya.com' })
    expect(await screen.findByLabelText(/Employee ID/)).toBeInTheDocument()
    expect(await screen.findByLabelText(/Budget Code/)).toBeInTheDocument()
  })

  it('processes for real and stores the claims', async () => {
    const { user } = renderApp('/preview', { as: 'operator@papaya.com' })
    await user.click(await screen.findByRole('button', { name: 'Gửi xử lý thật (3 công ty)' }))
    expect(await screen.findByRole('heading', { name: 'Kết quả xử lý (đã lưu)' })).toBeInTheDocument()
    expect(backend.claims).toHaveLength(3)
  })

  it('hides processing from a viewer', async () => {
    renderApp('/preview', { as: 'viewer@papaya.com' })
    await screen.findByRole('button', { name: 'Xem trước (3 công ty)' })
    expect(screen.queryByRole('button', { name: /Gửi xử lý thật/ })).not.toBeInTheDocument()
  })
})

describe('config diff', () => {
  it('compares two tenants from the URL', async () => {
    renderApp('/diff?a=safeguard&b=govhealth', { as: 'viewer@papaya.com' })
    expect(await screen.findByRole('heading', { name: '2 điểm khác nhau' })).toBeInTheDocument()
    // plain words instead of config paths and raw values
    const threshold = screen.getByText('Ngưỡng tự động duyệt').closest('tr')
    expect(within(threshold).getByText('Thay đổi')).toBeInTheDocument()
    expect(within(threshold).getByText('20.000')).toBeInTheDocument()
    const removed = screen.getByText('Loại yêu cầu được hỗ trợ').closest('tr')
    expect(within(removed).getByText('Nha khoa')).toBeInTheDocument()
    expect(within(removed).getByText('Chưa có')).toBeInTheDocument()
    expect(screen.queryByText('approval.auto_approval_threshold')).not.toBeInTheDocument()
  })

  it('shows the full configs side by side, highlighting differences', async () => {
    const { user } = renderApp('/diff?a=safeguard&b=govhealth', { as: 'viewer@papaya.com' })
    await user.click(await screen.findByLabelText('Hiện toàn bộ cấu hình song song'))
    const row = screen.getByText('Ngưỡng tự động duyệt').closest('tr')
    expect(row).toHaveClass('differs')
    expect(within(row).getByText('20.000')).toBeInTheDocument()
    const tier = screen.getByText('Cấp duyệt 1 › Vai trò duyệt').closest('tr')
    expect(within(tier).getByText('Chuyên viên thẩm định')).toBeInTheDocument()
    expect(within(tier).getByText('Hội đồng xét duyệt')).toBeInTheDocument()
  })
})

describe('config history', () => {
  it('rolls back to a previous version after confirmation', async () => {
    const t = backend.tenants[0]
    t.versions.push({ ...t.versions[0], version: 2, note: 'raise', source: 'update' })
    t.current_version = 2
    const { user } = renderApp('/tenants/safeguard/history', { as: 'admin@papaya.com' })
    await user.click(await screen.findByRole('button', { name: 'Khôi phục v1' }))
    const dialog = await screen.findByRole('dialog')
    await user.click(within(dialog).getByRole('button', { name: 'Khôi phục v1' }))
    expect(await screen.findByText('Đã khôi phục v1 thành phiên bản mới v3.')).toBeInTheDocument()
    expect(t.current_version).toBe(3)
    await waitFor(() => expect(screen.getByText('khôi phục v1')).toBeInTheDocument())
  })
})
