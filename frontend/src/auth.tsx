import { createContext, useContext, useState, useCallback, useEffect, ReactNode } from 'react'

const TOKEN_KEY = 'kk_alert_token'

export type UserRole = 'admin' | 'user'

export type User = {
  id: number
  username: string
  role: UserRole
}

type AuthContextType = {
  token: string | null
  user: User | null
  /** True while token is set but /auth/me has not yet returned (e.g. after refresh). */
  userLoading: boolean
  login: (username: string, password: string) => Promise<void>
  logout: () => void
  refreshUser: () => Promise<void>
}

const AuthContext = createContext<AuthContextType | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [token, setToken] = useState<string | null>(() => localStorage.getItem(TOKEN_KEY))
  const [user, setUser] = useState<User | null>(null)
  // Start true when token exists so first paint (before /auth/me) does not redirect admin routes to dashboard
  const [userLoading, setUserLoading] = useState<boolean>(() => !!localStorage.getItem(TOKEN_KEY))

  const refreshUser = useCallback(async () => {
    const t = localStorage.getItem(TOKEN_KEY)
    if (!t) {
      setUser(null)
      setUserLoading(false)
      return
    }
    setUserLoading(true)
    try {
      const res = await fetch('/api/v1/auth/me', { headers: { Authorization: `Bearer ${t}` } })
      if (!res.ok) {
        setUser(null)
        return
      }
      const data = await res.json()
      setUser({
        id: data.id,
        username: data.username,
        role: (data.role === 'admin' ? 'admin' : 'user') as UserRole,
      })
    } catch {
      setUser(null)
    } finally {
      setUserLoading(false)
    }
  }, [])

  useEffect(() => {
    if (token) refreshUser()
    else {
      setUser(null)
      setUserLoading(false)
    }
  }, [token, refreshUser])

  const login = useCallback(async (username: string, password: string) => {
    const res = await fetch('/api/v1/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username, password }),
    })
    if (!res.ok) {
      const e = await res.json().catch(() => ({}))
      throw new Error(e.error || 'Login failed')
    }
    const data = await res.json()
    setToken(data.token)
    localStorage.setItem(TOKEN_KEY, data.token)
    setUser({
      id: data.user?.id ?? 0,
      username: data.user?.username ?? username,
      role: (data.user?.role === 'admin' ? 'admin' : 'user') as UserRole,
    })
  }, [])
  const logout = useCallback(() => {
    setToken(null)
    setUser(null)
    localStorage.removeItem(TOKEN_KEY)
  }, [])
  return (
    <AuthContext.Provider value={{ token, user, userLoading, login, logout, refreshUser }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}

export function authHeaders(): Record<string, string> {
  const t = localStorage.getItem(TOKEN_KEY)
  if (!t) return { 'Content-Type': 'application/json' }
  return { 'Content-Type': 'application/json', Authorization: `Bearer ${t}` }
}
