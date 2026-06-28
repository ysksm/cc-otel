import { createContext, useContext, useEffect, type ReactNode } from 'react'
import { api, setOnUnauthorized, type Account } from '../lib/api'
import { useAsync } from './useAsync'

interface AuthContextValue {
  loading: boolean
  enabled: boolean
  authenticated: boolean
  account: Account | undefined
  refresh: () => void
  logout: () => Promise<void>
}

const AuthContext = createContext<AuthContextValue | undefined>(undefined)

export function AuthProvider({ children }: { children: ReactNode }) {
  const { data, loading, reload } = useAsync(() => api.getMe(), [])

  // When any API call fails with 401 mid-session, re-fetch /me so the gate can
  // re-render into the login screen.
  useEffect(() => {
    setOnUnauthorized(() => reload())
    return () => setOnUnauthorized(undefined)
  }, [reload])

  const value: AuthContextValue = {
    loading,
    enabled: data?.enabled ?? false,
    authenticated: data?.authenticated ?? false,
    account: data?.account,
    refresh: reload,
    logout: async () => {
      try {
        await api.logout()
      } finally {
        reload()
      }
    },
  }
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}
