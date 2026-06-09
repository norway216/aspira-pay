import { useState, useEffect } from 'react'
import { Routes, Route, Navigate } from 'react-router-dom'
import Layout from './components/Layout'
import UserLayout from './components/UserLayout'
import Dashboard from './pages/Dashboard'
import Transactions from './pages/Transactions'
import Users from './pages/Users'
import Ledger from './pages/Ledger'
import Audit from './pages/Audit'
import Cards from './pages/Cards'
import Login from './pages/Login'
import UserDashboard from './pages/UserDashboard'
import UserCards from './pages/UserCards'
import UserPayments from './pages/UserPayments'
import UserTransfer from './pages/UserTransfer'

export default function App() {
  const [auth, setAuth] = useState<{ token: string; isAdmin: boolean; userId: string; username: string } | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const tok = localStorage.getItem('auth_token')
    if (tok) {
      fetch('/api/v2/users/me', { headers: { Authorization: `Bearer ${tok}` } })
        .then(r => r.json())
        .then(d => {
          if (d.user_id) {
            const isAdmin = !!(d.admin_key && d.admin_key.startsWith('aspira-'))
            setAuth({ token: tok, isAdmin, userId: d.user_id, username: d.username })
          } else {
            localStorage.removeItem('auth_token')
          }
        })
        .catch(() => localStorage.removeItem('auth_token'))
        .finally(() => setLoading(false))
    } else {
      setLoading(false)
    }
  }, [])

  const handleLogin = (token: string, isAdmin: boolean, userId: string) => {
    setAuth({ token, isAdmin, userId, username: '' })
  }

  if (loading) {
    return <div className="min-h-screen bg-gray-950 flex items-center justify-center">
      <div className="text-gray-500 text-lg">⟳ Loading...</div>
    </div>
  }

  if (!auth) {
    return <Login onLogin={handleLogin} />
  }

  return (
    <Routes>
      {auth.isAdmin ? (
        <Route element={<Layout />}>
          <Route path="/" element={<Dashboard />} />
          <Route path="/transactions" element={<Transactions />} />
          <Route path="/cards" element={<Cards />} />
          <Route path="/users" element={<Users />} />
          <Route path="/ledger" element={<Ledger />} />
          <Route path="/audit" element={<Audit />} />
          <Route path="*" element={<Navigate to="/" />} />
        </Route>
      ) : (
        <Route element={<UserLayout />}>
          <Route path="/" element={<UserDashboard userId={auth.userId} token={auth.token} />} />
          <Route path="/transfer" element={<UserTransfer userId={auth.userId} token={auth.token} />} />
          <Route path="/cards" element={<UserCards userId={auth.userId} token={auth.token} />} />
          <Route path="/payments" element={<UserPayments userId={auth.userId} token={auth.token} />} />
          <Route path="*" element={<Navigate to="/" />} />
        </Route>
      )}
    </Routes>
  )
}
