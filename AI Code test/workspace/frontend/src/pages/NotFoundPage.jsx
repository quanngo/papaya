import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import ui from '../components/ui.module.css'

export default function NotFoundPage() {
  const { t } = useTranslation()
  return (
    <div className={ui.page}>
      <h1 className={ui.title}>{t('notFound.title')}</h1>
      <p className={ui.subtitle}>
        <Link to="/tenants">{t('notFound.back')}</Link>
      </p>
    </div>
  )
}
