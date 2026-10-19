import { describe, expect, it } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import i18n from '../i18n'
import { renderApp } from '../test/renderApp'
import { backend } from '../test/server'

describe('authentication', () => {
  it('sends an anonymous visitor to the login page, keeping the target', async () => {
    renderApp('/tenants')
    await waitFor(() => expect(screen.getByTestId('location')).toHaveTextContent('/login?next=%2Ftenants'))
  })

  it('shows a translated error for wrong credentials', async () => {
    const { user } = renderApp('/login')
    await user.type(await screen.findByLabelText('Email'), 'admin@papaya.com')
    await user.type(screen.getByLabelText('Mật khẩu'), 'wrong')
    await user.click(screen.getByRole('button', { name: 'Đăng nhập' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('Email hoặc mật khẩu không đúng.')
  })

  it('logs in and lands on the requested page', async () => {
    const { user } = renderApp('/login?next=%2Fdiff')
    await user.type(await screen.findByLabelText('Email'), 'operator@papaya.com')
    await user.type(screen.getByLabelText('Mật khẩu'), 'Operator@2026')
    await user.click(screen.getByRole('button', { name: 'Đăng nhập' }))
    await waitFor(() => expect(screen.getByTestId('location')).toHaveTextContent('/diff'))
    expect(await screen.findByRole('heading', { name: 'So sánh cấu hình' })).toBeInTheDocument()
  })

  it('never follows an external next= after login', async () => {
    const { user } = renderApp('/login?next=%2F%2Fevil.example')
    await user.type(await screen.findByLabelText('Email'), 'admin@papaya.com')
    await user.type(screen.getByLabelText('Mật khẩu'), '123456')
    await user.click(screen.getByRole('button', { name: 'Đăng nhập' }))
    await waitFor(() => expect(screen.getByTestId('location')).toHaveTextContent('/dashboard'))
  })

  it('hides manage actions and pages from a viewer', async () => {
    renderApp('/tenants', { as: 'viewer@papaya.com' })
    expect(await screen.findByRole('link', { name: 'SafeGuard Insurance' })).toBeInTheDocument()
    expect(screen.queryByRole('link', { name: /Thêm công ty/ })).not.toBeInTheDocument()
    expect(screen.getAllByRole('link', { name: 'Xem' }).length).toBeGreaterThan(0)
  })

  it('blocks a page without the permission', async () => {
    renderApp('/tenants/new', { as: 'viewer@papaya.com' })
    expect(await screen.findByRole('alert')).toHaveTextContent('Bạn không có quyền truy cập chức năng này.')
  })

  it('switches the language and saves it on the account', async () => {
    const { user } = renderApp('/tenants', { as: 'admin@papaya.com' })
    await screen.findByRole('heading', { name: 'Công ty bảo hiểm' })
    await user.selectOptions(screen.getByLabelText('Ngôn ngữ'), 'en')
    expect(await screen.findByRole('heading', { name: 'Insurers' })).toBeInTheDocument()
    await waitFor(() => expect(backend.savedLocale).toBe('en'))
    expect(i18n.language).toBe('en')
  })

  it('confirms before logging out', async () => {
    const { user } = renderApp('/tenants', { as: 'admin@papaya.com' })
    await user.click(await screen.findByRole('button', { name: 'Đăng xuất' }))
    const dialog = await screen.findByRole('dialog')
    expect(dialog).toHaveTextContent('Đăng xuất?')
    expect(document.activeElement).toHaveTextContent('Hủy')
    await user.click(screen.getAllByRole('button', { name: 'Đăng xuất' }).at(-1))
    await waitFor(() => expect(screen.getByTestId('location')).toHaveTextContent('/login'))
    expect(backend.user).toBeNull()
  })
})
