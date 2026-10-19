// Main user flow against the REAL backend (Go + SQLite), through the UI:
// identical claims -> 3 tenants behave differently; onboard a 4th tenant with
// no code change and process a claim; save a version, roll it back; diff.
//
// Needs a fresh backend database (the seeded tenants and accounts from
// workspace/backend/migrations/005_seed_users.sql and seed/*.json), e.g.:
//   cd workspace/backend && DATABASE_URL="file:./data/e2e.db?_pragma=foreign_keys(1)" go run .
// and E2E_BASE_URL pointing at it (the backend must serve ../frontend/dist).
// Optional: E2E_EMAIL / E2E_PASSWORD (default: the seeded admin).
import { expect, test } from '@playwright/test'

test.skip(!process.env.E2E_BASE_URL, 'E2E_BASE_URL is not set: start the real stack first (see header)')

const email = process.env.E2E_EMAIL || 'admin@papaya.com'
const password = process.env.E2E_PASSWORD || '123456'
// Unique per run, so the spec can run again against the same database.
const runId = Date.now().toString(36)
const slug = `carelink-${runId}`
const companyName = `CareLink ${runId}`

test.beforeEach(async ({ page }) => {
  await page.goto('/login')
  await page.getByLabel('Email').fill(email)
  await page.getByLabel('Mật khẩu').fill(password)
  await page.getByRole('button', { name: 'Đăng nhập' }).click()
  await expect(page.getByRole('heading', { name: 'Tổng quan' })).toBeVisible()
})

test('same claim, three tenants, three different outcomes', async ({ page }) => {
  await page.getByRole('link', { name: 'Xem trước xử lý' }).click()
  await page.getByLabel(/Employee ID/).fill('EMP-12345')
  await page.getByLabel(/Department/).fill('Finance')
  await page.getByLabel(/Budget Code/).fill('FIN-2026')
  await page.getByRole('button', { name: /^Xem trước \(/ }).click()

  await expect(page.getByRole('article', { name: 'SafeGuard Insurance' }).getByText('Tự động duyệt', { exact: true })).toBeVisible()
  await expect(page.getByRole('article', { name: 'HealthFirst' }).getByText('Chuyên viên thẩm định', { exact: true })).toBeVisible()
  await expect(page.getByRole('article', { name: 'GovHealth' }).getByText('Hội đồng xét duyệt', { exact: true })).toBeVisible()
})

test('onboard a 4th tenant through the UI and process a claim', async ({ page }) => {
  await page.getByRole('link', { name: /Thêm công ty/ }).click()
  await page.getByLabel('Mã định danh (slug)').fill(slug)
  await page.getByLabel('Tên công ty').fill(companyName)
  await page.getByRole('checkbox', { name: 'Nhãn khoa' }).check()
  await page.getByLabel('Nhãn khoa', { exact: true }).last().fill('3') // SLA for the newly enabled type
  await expect(page.getByText('Cấu hình hợp lệ.')).toBeVisible()
  await page.getByRole('button', { name: 'Tạo công ty' }).click()
  await expect(page.getByText('Đã tạo công ty (phiên bản 1).')).toBeVisible()

  await page.getByRole('link', { name: 'Xem trước xử lý' }).click()
  // Only the new tenant: uncheck every insurer, then check ours.
  const insurers = page.getByRole('group', { name: 'Công ty' }).getByRole('checkbox')
  await expect(insurers.first()).toBeVisible()
  for (const box of await insurers.all()) await box.uncheck()
  await page.getByRole('checkbox', { name: companyName }).check()
  await page.getByRole('button', { name: 'Gửi xử lý thật (1 công ty)' }).click()
  const card = page.getByRole('article', { name: companyName })
  await expect(card.getByText('Chuyên viên thẩm định', { exact: true })).toBeVisible()
  await expect(page.getByRole('heading', { name: 'Kết quả xử lý (đã lưu)' })).toBeVisible()
})

test('save a version, roll it back, and diff versions', async ({ page }) => {
  await page.goto('/tenants/healthfirst')
  const threshold = page.getByLabel('Ngưỡng tự động duyệt')
  await threshold.fill('8000')
  await page.getByRole('button', { name: 'Lưu phiên bản mới' }).click()
  await expect(page.getByText(/Đã lưu phiên bản \d+\./)).toBeVisible()

  await page.getByRole('link', { name: 'Lịch sử' }).click()
  const restore = page.getByRole('button', { name: /^Khôi phục v\d+$/ }).first()
  await restore.click()
  await page.getByRole('dialog').getByRole('button', { name: /^Khôi phục v/ }).click()
  await expect(page.getByText(/Đã khôi phục v\d+ thành phiên bản mới v\d+\./)).toBeVisible()

  await page.goto('/diff?a=safeguard&b=govhealth')
  await expect(page.getByRole('heading', { name: /điểm khác nhau/ })).toBeVisible()
  await expect(page.getByText('Ngưỡng tự động duyệt').first()).toBeVisible()
})
