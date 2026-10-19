import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react'
import * as authApi from '../api/auth'
import i18n, { isSupportedLanguage, rememberLanguage } from '../i18n'

const AuthContext = createContext(null)

/** Holds the signed-in user (from the session cookie). Nothing auth-related is stored in the browser. */
export function AuthProvider({ children }) {
  const [user, setUser] = useState(null)
  const [loading, setLoading] = useState(true)

  const applyUserLocale = (u) => {
    if (u && isSupportedLanguage(u.locale) && u.locale !== i18n.language) {
      i18n.changeLanguage(u.locale)
      rememberLanguage(u.locale)
    }
  }

  useEffect(() => {
    let alive = true
    authApi
      .fetchMe()
      .then((u) => {
        if (!alive) return
        applyUserLocale(u)
        setUser(u)
      })
      .catch(() => alive && setUser(null))
      .finally(() => alive && setLoading(false))
    return () => {
      alive = false
    }
  }, [])

  const login = useCallback(async (email, password, rememberMe) => {
    const u = await authApi.login(email, password, rememberMe)
    applyUserLocale(u)
    setUser(u)
    return u
  }, [])

  const logout = useCallback(async () => {
    try {
      await authApi.logout()
    } finally {
      setUser(null) // always sign out locally, even if the request failed
    }
  }, [])

  const can = useCallback((permission) => !!user?.permissions?.includes(permission), [user])

  /** Switches the UI now and saves the choice on the account when signed in. */
  const setLanguage = useCallback(
    async (code) => {
      if (!isSupportedLanguage(code)) return
      await i18n.changeLanguage(code)
      rememberLanguage(code)
      if (user) {
        try {
          setUser(await authApi.saveLocale(code))
        } catch {
          // the UI already switched; the account keeps its previous language
        }
      }
    },
    [user],
  )

  const value = useMemo(() => ({ user, loading, login, logout, can, setLanguage }), [user, loading, login, logout, can, setLanguage])
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used inside <AuthProvider>')
  return ctx
}
