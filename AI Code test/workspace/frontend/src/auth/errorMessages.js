import i18n from '../i18n'

/**
 * Turns an error kept in state into text at render time, so switching the
 * language re-translates it. Order: this app's wording for the context
 * (errors.<context>.<CODE>), then the common wording (errors.common.<CODE>),
 * then the backend's error_text, then a generic message.
 *
 * @param {(key: string, opts?: object) => string} t
 * @param {any} err  ApiError, an Error or a code string
 * @param {string} [context]  e.g. 'login', 'tenant'
 */
export function actionErrorMessage(t, err, context = 'common') {
  if (!err) return ''
  const code = typeof err === 'string' ? err : err.code
  if (code) {
    for (const key of [`errors.${context}.${code}`, `errors.common.${code}`]) {
      if (i18n.exists(key)) return t(key)
    }
  }
  return err.errorText || t('errors.common.UNKNOWN')
}
