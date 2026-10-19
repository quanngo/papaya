import '@testing-library/jest-dom/vitest'
import { afterAll, afterEach, beforeAll, beforeEach } from 'vitest'
import { cleanup } from '@testing-library/react'
import i18n from '../i18n'
import { resetBackend, server } from './server'

beforeAll(() => server.listen({ onUnhandledRequest: 'error' }))
beforeEach(async () => {
  resetBackend()
  window.localStorage.clear()
  await i18n.changeLanguage('vi') // tests run in Vietnamese; switch explicitly when a test is about English
})
afterEach(() => {
  cleanup()
  server.resetHandlers()
})
afterAll(() => server.close())
