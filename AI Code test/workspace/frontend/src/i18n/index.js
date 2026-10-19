import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import vi from '../locales/vi.json'
import en from '../locales/en.json'

export const LANGUAGES = [
  { code: 'vi', label: 'Tiếng Việt' },
  { code: 'en', label: 'English' },
]

/** The only value this app keeps in localStorage: the UI language code. */
export const LANGUAGE_STORAGE_KEY = 'claimconfig-admin-lang'

export function isSupportedLanguage(code) {
  return LANGUAGES.some((l) => l.code === code)
}

function storedLanguage() {
  try {
    const v = window.localStorage.getItem(LANGUAGE_STORAGE_KEY)
    return isSupportedLanguage(v) ? v : null
  } catch {
    return null
  }
}

export function rememberLanguage(code) {
  try {
    window.localStorage.setItem(LANGUAGE_STORAGE_KEY, code)
  } catch {
    // storage unavailable (private mode): the choice still applies to this page
  }
}

i18n.use(initReactI18next).init({
  resources: { vi: { translation: vi }, en: { translation: en } },
  lng: storedLanguage() || 'vi',
  fallbackLng: 'vi',
  interpolation: { escapeValue: false },
  returnNull: false,
})

i18n.on('languageChanged', (lng) => {
  if (typeof document !== 'undefined') document.documentElement.lang = lng
})

export default i18n
