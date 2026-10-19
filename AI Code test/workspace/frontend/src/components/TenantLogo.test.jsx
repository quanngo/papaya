import { describe, expect, it } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import TenantLogo, { isImageUrl } from './TenantLogo'
import { renderApp } from '../test/renderApp'

describe('tenant logo', () => {
  it('renders the logo image from the URL', () => {
    render(<TenantLogo url="https://example.com/logo.png" name="CareLink Insurance" color="#6A1B9A" />)
    expect(screen.getByRole('img', { name: 'Logo CareLink Insurance' })).toHaveAttribute('src', 'https://example.com/logo.png')
  })

  it('falls back to initials when the URL is empty or the image fails', () => {
    const { rerender } = render(<TenantLogo url="" name="CareLink Insurance" color="#6A1B9A" />)
    expect(screen.getByRole('img', { name: 'Logo CareLink Insurance' })).toHaveTextContent('CI')
    rerender(<TenantLogo url="https://example.com/broken.png" name="CareLink Insurance" color="#6A1B9A" />)
    fireEvent.error(screen.getByRole('img'))
    expect(screen.getByRole('img', { name: 'Logo CareLink Insurance' })).toHaveTextContent('CI')
  })

  it('only accepts http(s) URLs', () => {
    expect(isImageUrl('https://a.b/c.png')).toBe(true)
    expect(isImageUrl('javascript:alert(1)')).toBe(false)
    expect(isImageUrl('data:image/svg+xml,<svg/>')).toBe(false)
  })

  it('previews the logo next to the URL field and reports a broken URL', async () => {
    renderApp('/tenants/govhealth', { as: 'admin@papaya.com' })
    const field = await screen.findByLabelText('URL logo')
    fireEvent.change(field, { target: { value: 'https://example.com/missing.png' } })
    // header logo + field preview both point at the URL; a browser fails both
    const broken = screen.getAllByRole('img', { name: 'Logo GovHealth' }).filter((el) => el.getAttribute('src') === 'https://example.com/missing.png')
    expect(broken).toHaveLength(2)
    expect(screen.getByText('Đang tải logo…')).toBeInTheDocument()
    broken.forEach((img) => fireEvent.error(img))
    expect(await screen.findByText(/Không tải được ảnh từ URL này/)).toBeInTheDocument()
    expect(field).toHaveAttribute('aria-invalid', 'true')
  })
})
