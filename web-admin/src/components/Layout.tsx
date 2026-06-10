import { useState, useEffect } from 'react'
import { Outlet, NavLink, useNavigate } from 'react-router-dom'

export default function Layout() {
  const nav = useNavigate()
  const [username, setUsername] = useState('')

  useEffect(() => {
    const tok = localStorage.getItem('auth_token')
    if (tok) {
      fetch('/api/v2/users/me', { headers: { Authorization: `Bearer ${tok}` } })
        .then(r => r.json()).then(d => { if (d.username) setUsername(d.username) }).catch(() => {})
    }
  }, [])

  const logout = () => { localStorage.removeItem('auth_token'); nav('/'); window.location.reload() }

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
      {/* Sidebar */}
      <aside className="w-64 bg-gray-900/90 border-r border-gray-800/50 flex flex-col backdrop-blur-sm">
        {/* Brand */}
        <div className="px-5 py-5 border-b border-gray-800/50">
          <div className="flex items-center gap-3">
            <img src="/logo.png" alt="Aspira Pay" className="w-9 h-9 object-contain opacity-90" />
            <div>
              <h1 className="text-base font-bold text-white leading-tight">Aspira Pay</h1>
              <p className="text-[10px] text-gray-500 uppercase tracking-wider">Admin Console</p>
            </div>
          </div>
        </div>

        {/* User */}
        <div className="mx-4 my-3 p-3 bg-gray-800/40 rounded-xl">
          <div className="flex items-center gap-3">
            <div className="w-8 h-8 rounded-full bg-[#8B2E2E] flex items-center justify-center text-white text-xs font-bold">
              {username ? username[0].toUpperCase() : 'A'}
            </div>
            <div className="flex-1 min-w-0">
              <p className="text-sm text-white truncate font-medium">{username || 'Admin'}</p>
              <div className="flex items-center gap-1.5 mt-0.5">
                <span className="w-1.5 h-1.5 rounded-full bg-blue-400 animate-pulse"></span>
                <span className="text-[11px] text-[#E07373]">Administrator</span>
              </div>
            </div>
          </div>
        </div>

        {/* Nav */}
        <nav className="flex-1 px-3 py-2 space-y-0.5">
          {navItems.map(item => (
            <NavLink key={item.to} to={item.to} end={item.to === '/'}
              className={({ isActive }) =>
                `flex items-center gap-3 px-3 py-2.5 rounded-xl text-sm transition-all duration-200 ${
                  isActive ? 'bg-[#C84B4B]/15 text-[#E07373] font-medium border border-[#C84B4B]/20' : 'text-gray-400 hover:text-white hover:bg-gray-800/50'
                }`}>
              <span className="text-base">{item.icon}</span>{item.label}
            </NavLink>
          ))}
        </nav>

        {/* Footer */}
        <div className="p-3 border-t border-gray-800/50 space-y-2">
          <button onClick={logout}
            className="w-full flex items-center justify-center gap-2 px-4 py-2.5 rounded-xl text-sm font-medium text-white bg-red-600/80 hover:bg-red-500 transition-all">
            <span>🚪</span>Sign Out
          </button>
          <p className="text-center text-[10px] text-gray-600">Aspira Pay · Sandbox V4</p>
        </div>
      </aside>

      {/* Main */}
      <main className="flex-1 overflow-auto bg-gradient-to-br from-gray-950 via-gray-950 to-gray-900">
        <div className="p-8 max-w-7xl"><Outlet /></div>
      </main>
    </div>
  )
}
