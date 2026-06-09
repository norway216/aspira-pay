import { useState, useEffect } from 'react'
import { Outlet, NavLink, useNavigate } from 'react-router-dom'

export default function UserLayout() {
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

  return (
    <div className="min-h-screen bg-gray-950 flex">
      <aside className="w-60 bg-gray-900 border-r border-gray-800 flex flex-col">
        {/* Header */}
        <div className="p-5 border-b border-gray-800">
          <h1 className="text-lg font-bold bg-gradient-to-r from-emerald-400 to-cyan-300 bg-clip-text text-transparent">Aspira Pay</h1>
          <p className="text-xs text-gray-500 mt-1">Personal Banking</p>
        </div>

        {/* User Info */}
        <div className="px-5 py-3 border-b border-gray-800">
          <div className="flex items-center gap-3">
            <div className="w-9 h-9 rounded-full bg-emerald-700 flex items-center justify-center text-white text-sm font-bold">
              {username ? username[0].toUpperCase() : '?'}
            </div>
            <div className="flex-1 min-w-0">
              <p className="text-sm text-white truncate">{username || 'Loading...'}</p>
              <div className="flex items-center gap-1.5 mt-0.5">
                <span className="w-1.5 h-1.5 rounded-full bg-emerald-400"></span>
                <span className="text-xs text-emerald-400">Online</span>
              </div>
            </div>
          </div>
        </div>

        {/* Nav */}
        <nav className="flex-1 p-3 space-y-1">
          {[
            { to: '/', label: 'Dashboard', icon: '📊' },
            { to: '/transfer', label: 'Transfer', icon: '💸' },
            { to: '/cards', label: 'My Cards', icon: '💳' },
            { to: '/payments', label: 'Payments', icon: '💱' },
          ].map(item => (
            <NavLink key={item.to} to={item.to} end={item.to === '/'}
              className={({ isActive }) =>
                `flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm transition-colors ${
                  isActive ? 'bg-emerald-600/20 text-emerald-400 border border-emerald-600/30'
                  : 'text-gray-400 hover:text-white hover:bg-gray-800'
                }`}>
              <span>{item.icon}</span>{item.label}
            </NavLink>
          ))}
        </nav>

        {/* Logout — prominent */}
        <div className="p-3 border-t border-gray-800">
          <button onClick={logout}
            className="w-full flex items-center justify-center gap-2 px-4 py-3 rounded-lg text-sm font-medium text-white bg-red-600 hover:bg-red-500 transition-colors">
            🚪 Sign Out
          </button>
          <p className="text-center text-xs text-gray-600 mt-2">Aspira Pay · Sandbox</p>
        </div>
      </aside>

      <main className="flex-1 overflow-auto"><div className="p-6"><Outlet /></div></main>
    </div>
  )
}
