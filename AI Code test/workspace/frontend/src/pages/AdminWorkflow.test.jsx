import { describe, expect, it } from 'vitest'
import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import { renderApp } from '../test/renderApp'
import { backend } from '../test/server'

describe('admin workflow', () => {
  it('opens on the overview with key numbers and quick actions', async () => {
    renderApp('/', { as: 'operator@papaya.com' })
    await waitFor(() => expect(screen.getByTestId('location')).toHaveTextContent('/dashboard'))
    const stats = await screen.findByRole('region', { name: 'Số liệu chính' })
    expect(within(stats).getByRole('link', { name: /3\s*Công ty đang hoạt động/ })).toHaveAttribute('href', '/tenants')
    expect(screen.getByRole('link', { name: /Thêm công ty/ })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /Xử lý yêu cầu/ })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'SafeGuard Insurance' })).toBeInTheDocument()
  })

  it('groups the main menu by configuration and operations', async () => {
    renderApp('/dashboard', { as: 'admin@papaya.com' })
    const nav = await screen.findByRole('navigation', { name: 'Menu chính' })
    expect(within(nav).getByText('Cấu hình')).toBeInTheDocument()
    expect(within(nav).getByText('Vận hành')).toBeInTheDocument()
    expect(within(nav).getByRole('link', { name: /Tổng quan/ })).toHaveAttribute('aria-current', 'page')
  })

  it('searches and summarizes insurers in the list', async () => {
    const { user } = renderApp('/tenants', { as: 'admin@papaya.com' })
    const row = (await screen.findByRole('link', { name: 'GovHealth' })).closest('tr')
    // config summary columns fill in once the details arrive
    await waitFor(() => expect(within(row).getByText('Email, Webhook')).toBeInTheDocument())
    expect(within(row).getByText('Ngoại trú')).toBeInTheDocument()
    await user.type(screen.getByLabelText('Tìm công ty'), 'health')
    expect(screen.queryByRole('link', { name: 'SafeGuard Insurance' })).not.toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'HealthFirst' })).toBeInTheDocument()
    expect(screen.getByText('2 / 3 công ty')).toBeInTheDocument()
    expect(screen.getByTestId('location')).toHaveTextContent('/tenants?q=health')
  })

  it('opens the preview for one insurer from the list', async () => {
    const { user } = renderApp('/tenants', { as: 'operator@papaya.com' })
    await user.click(await screen.findByRole('link', { name: 'Xem trước xử lý cho HealthFirst' }))
    expect(await screen.findByRole('button', { name: 'Xem trước (1 công ty)' })).toBeInTheDocument()
    expect(screen.getByRole('checkbox', { name: 'HealthFirst' })).toBeChecked()
    expect(screen.getByRole('checkbox', { name: 'GovHealth' })).not.toBeChecked()
  })

  it('marks unsaved edits and saves with Ctrl+S', async () => {
    renderApp('/tenants/safeguard', { as: 'admin@papaya.com' })
    const threshold = await screen.findByLabelText('Ngưỡng tự động duyệt')
    expect(screen.getByRole('navigation', { name: 'Đi tới phần cấu hình' })).toBeInTheDocument()
    fireEvent.change(threshold, { target: { value: '25000' } })
    expect(screen.getByText('Chưa lưu')).toBeInTheDocument()
    await waitFor(() => expect(screen.getByRole('button', { name: 'Lưu phiên bản mới' })).toBeEnabled(), { timeout: 2000 })
    fireEvent.keyDown(document, { key: 's', ctrlKey: true })
    expect(await screen.findByText('Đã lưu phiên bản 2.')).toBeInTheDocument()
    expect(backend.tenants[0].current_version).toBe(2)
    expect(screen.queryByText('Chưa lưu')).not.toBeInTheDocument()
  })

  it('filters processed claims by outcome from the URL', async () => {
    backend.claimsList = [
      { id: 'c1', tenant_id: 'id-safeguard', tenant_slug: 'safeguard', config_version: 1, claim_type: 'OUTPATIENT', amount: 1000, routing: 'auto_approved', created_at: '2026-10-17T08:00:00.000Z' },
      { id: 'c2', tenant_id: 'id-govhealth', tenant_slug: 'govhealth', config_version: 1, claim_type: 'OUTPATIENT', amount: 1000, routing: 'manual_review', approver_role: 'committee', created_at: '2026-10-17T08:00:01.000Z' },
    ]
    const { user } = renderApp('/claims?routing=manual_review', { as: 'viewer@papaya.com' })
    expect(await screen.findByRole('button', { name: 'Duyệt tay (1)' })).toHaveAttribute('aria-pressed', 'true')
    expect(screen.getByText('govhealth')).toBeInTheDocument()
    expect(screen.queryByText('safeguard')).not.toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Tất cả (2)' }))
    expect(screen.getByText('safeguard')).toBeInTheDocument()
  })
})
