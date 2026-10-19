import { useTranslation } from 'react-i18next'
import { useAuth } from '../auth/AuthContext'
import { LANGUAGES } from '../i18n'

/** Language picker; signed in, the choice is saved on the account. */
export default function LanguageSwitcher({ className }) {
  const { t, i18n } = useTranslation()
  const { setLanguage } = useAuth()
  return (
    <select className={className} aria-label={t('layout.language')} value={i18n.language} onChange={(e) => setLanguage(e.target.value)}>
      {LANGUAGES.map((l) => (
        <option key={l.code} value={l.code}>
          {l.label}
        </option>
      ))}
    </select>
  )
}
