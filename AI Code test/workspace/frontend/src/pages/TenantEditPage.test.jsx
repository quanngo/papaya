import { describe, expect, it } from 'vitest'
import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import { renderApp } from '../test/renderApp'
import { backend } from '../test/server'

describe('tenant configuration form', () => {
  it('validates live and blocks saving an invalid config', async () => {
    renderApp('/tenants/new', { as: 'operator@papaya.com' })
    const threshold = await screen.findByLabelText('Ngưỡng tự động duyệt')
    fireEvent.change(threshold, { target: { value: '-5' } })
    expect(await screen.findByText('Ngưỡng tự động duyệt phải lớn hơn hoặc bằng 0.', {}, { timeout: 2000 })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Tạo công ty' })).toBeDisabled()
    expect(screen.getByText('Còn 1 lỗi cần sửa trước khi lưu.')).toBeInTheDocument()
  })

  it('onboards a 4th tenant through the form (no code change)', async () => {
    const { user } = renderApp('/tenants/new', { as: 'operator@papaya.com' })
    await user.type(await screen.findByLabelText('Mã định danh (slug)'), 'carelink')
    const name = screen.getByLabelText('Tên công ty')
    await user.clear(name)
    await user.type(name, 'CareLink Insurance')
    await user.click(screen.getByRole('checkbox', { name: 'Nhãn khoa' }))
    await waitFor(() => expect(screen.getByText('Cấu hình hợp lệ.')).toBeInTheDocument(), { timeout: 2000 })
    await user.click(screen.getByRole('button', { name: 'Tạo công ty' }))

    await waitFor(() => expect(screen.getByTestId('location')).toHaveTextContent('/tenants/carelink'))
    expect(await screen.findByText('Đã tạo công ty (phiên bản 1).')).toBeInTheDocument()
    const created = backend.tenants.find((t) => t.slug === 'carelink')
    expect(created.versions[0].config.claim_types.enabled).toEqual(['OUTPATIENT', 'OPTICAL'])
  })

  it('saves a new version and shows a conflict when someone saved first', async () => {
    const { user } = renderApp('/tenants/safeguard', { as: 'admin@papaya.com' })
    const threshold = await screen.findByLabelText('Ngưỡng tự động duyệt')
    await user.clear(threshold)
    await user.type(threshold, '30000')
    await waitFor(() => expect(screen.getByRole('button', { name: 'Lưu phiên bản mới' })).toBeEnabled(), { timeout: 2000 })
    await user.click(screen.getByRole('button', { name: 'Lưu phiên bản mới' }))
    expect(await screen.findByText('Đã lưu phiên bản 2.')).toBeInTheDocument()
    expect(backend.tenants[0].current_version).toBe(2)
    // tiers stay contiguous: the first tier now starts at the new threshold
    expect(backend.tenants[0].versions[1].config.approval.tiers[0].min_amount).toBe(30000)

    backend.forceConflict = true
    await user.type(threshold, '1')
    await waitFor(() => expect(screen.getByRole('button', { name: 'Lưu phiên bản mới' })).toBeEnabled(), { timeout: 2000 })
    await user.click(screen.getByRole('button', { name: 'Lưu phiên bản mới' }))
    expect(await screen.findByText(/Người khác vừa lưu cấu hình này/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Tải lại cấu hình mới nhất' })).toBeInTheDocument()
  })

  it('previews a claim with the unsaved config', async () => {
    const { user } = renderApp('/tenants/healthfirst', { as: 'admin@papaya.com' })
    const threshold = await screen.findByLabelText('Ngưỡng tự động duyệt')
    await user.clear(threshold)
    await user.type(threshold, '20000')
    await waitFor(() => expect(screen.getByRole('button', { name: 'Xem trước' })).toBeEnabled(), { timeout: 2000 })
    await user.click(screen.getByRole('button', { name: 'Xem trước' }))
    const card = await screen.findByRole('article', { name: 'HealthFirst' })
    expect(within(card).getByText('Tự động duyệt')).toBeInTheDocument()
  })

  it('is read-only for a viewer', async () => {
    renderApp('/tenants/govhealth', { as: 'viewer@papaya.com' })
    expect(await screen.findByText('Bạn chỉ có quyền xem cấu hình này.')).toBeInTheDocument()
    expect(screen.getByLabelText('Tên công ty')).toBeDisabled()
    expect(screen.queryByRole('button', { name: 'Lưu phiên bản mới' })).not.toBeInTheDocument()
  })
})
