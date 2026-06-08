import { useState, useEffect } from 'react'
import { Outlet, NavLink, useNavigate } from 'react-router-dom'

export default function Layout() {
  const nav = useNavigate()
  const [username, setUsername] = useState('')

  useEffect(() => {
    const tok = localStorage.getItem('auth_token')
    if (tok) {
      fetch('/api/v2/users/me', { headers: { Authorization: `Bearer ${tok}` } })
        .then(r => r.json())
        .then(d => { if (d.username) setUsername(d.username) })
        .catch(() => {})
    }
  }, [])

  const logout = () => {
    localStorage.removeItem('auth_token')
    nav('/')
    window.location.reload()
  }

  const navItems = [
    { to: '/', label: 'Dashboard', icon: '📊' },
    { to: '/transactions', label: 'Transactions', icon: '💱' },
    { to: '/cards', label: 'Cards', icon: '💳' },
    { to: '/users', label: 'Users', icon: '👥' },
    { to: '/ledger', label: 'Ledger', icon: '📒' },
    { to: '/audit', label: 'Audit Chain', icon: '⛓️' },
  ]

  return (
    <div className="min-h-screen bg-gray-950 flex">
      <aside className="w-64 bg-gray-900 border-r border-gray-800 flex flex-col">
        {/* Header */}
        <div className="p-6 border-b border-gray-800">
          <h1 className="text-xl font-bold bg-gradient-to-r from-blue-400 to-cyan-300 bg-clip-text text-transparent">
            Aspira Pay V2
          </h1>
          <p className="text-xs text-gray-500 mt-1">Cross-Border Payment System</p>
        </div>

        {/* Admin Info */}
        <div className="px-5 py-3 border-b border-gray-800">
          <div className="flex items-center gap-3">
            <div className="w-9 h-9 rounded-full bg-blue-700 flex items-center justify-center text-white text-sm font-bold">
              {username ? username[0].toUpperCase() : 'A'}
            </div>
            <div className="flex-1 min-w-0">
              <p className="text-sm text-white truncate">{username || 'Admin'}</p>
              <div className="flex items-center gap-1.5 mt-0.5">
                <span className="w-1.5 h-1.5 rounded-full bg-blue-400"></span>
                <span className="text-xs text-blue-400">Administrator</span>
              </div>
            </div>
          </div>
        </div>

        {/* Nav */}
        <nav className="flex-1 p-4 space-y-1">
          {navItems.map((item) => (
            <NavLink key={item.to} to={item.to} end={item.to === '/'}
              className={({ isActive }) =>
                `flex items-center gap-3 px-4 py-3 rounded-lg text-sm transition-colors ${
                  isActive ? 'bg-blue-600/20 text-blue-400 border border-blue-600/30'
                  : 'text-gray-400 hover:text-white hover:bg-gray-800'
                }`}>
              <span>{item.icon}</span>{item.label}
            </NavLink>
          ))}
        </nav>

        {/* Logout — prominent */}
        <div className="p-4 border-t border-gray-800">
          <button onClick={logout}
            className="w-full flex items-center justify-center gap-2 px-4 py-3 rounded-lg text-sm font-medium text-white bg-red-600 hover:bg-red-500 transition-colors">
            🚪 Sign Out
          </button>
          <p className="text-center text-xs text-gray-600 mt-2">Version 2.0.0-sandbox</p>
        </div>
      </aside>

      <main className="flex-1 overflow-auto">
        <div className="p-8"><Outlet /></div>
      </main>
    </div>
  )
}
