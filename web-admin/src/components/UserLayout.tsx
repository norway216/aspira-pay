import { useState, useEffect } from 'react'
import { Outlet, NavLink, useNavigate } from 'react-router-dom'

export default function UserLayout() {
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

  return (
    <div className="min-h-screen bg-gray-950 flex">
      {/* Sidebar */}
      <aside className="w-60 bg-gray-900/90 border-r border-gray-800/50 flex flex-col backdrop-blur-sm">
        {/* Brand */}
        <div className="px-4 py-4 border-b border-gray-800/50">
          <div className="flex items-center gap-2.5">
            <img src="/logo.png" alt="Aspira Pay" className="w-8 h-8 object-contain opacity-90" />
            <div>
              <h1 className="text-sm font-bold text-white leading-tight">Aspira Pay</h1>
              <p className="text-[10px] text-gray-500 uppercase tracking-wider">Personal</p>
            </div>
          </div>
        </div>

        {/* User */}
        <div className="mx-3 my-2 p-2.5 bg-gray-800/40 rounded-xl">
          <div className="flex items-center gap-2.5">
            <div className="w-7 h-7 rounded-full bg-[#3D1A1A] flex items-center justify-center text-white text-xs font-bold">
              {username ? username[0].toUpperCase() : '?'}
            </div>
            <div className="flex-1 min-w-0">
              <p className="text-sm text-white truncate font-medium">{username || 'Loading...'}</p>
              <div className="flex items-center gap-1 mt-0.5">
                <span className="w-1.5 h-1.5 rounded-full bg-emerald-400"></span>
                <span className="text-[11px] text-[#C84B4B]">Online</span>
              </div>
            </div>
          </div>
        </div>

        {/* Nav */}
        <nav className="flex-1 px-2.5 py-2 space-y-0.5">
          {[
            { to: '/', label: 'Dashboard', icon: '📊' },
            { to: '/transfer', label: 'Transfer', icon: '💸' },
            { to: '/cards', label: 'My Cards', icon: '💳' },
            { to: '/payments', label: 'Payments', icon: '💱' },
            { to: '/fx', label: 'FX Rates', icon: '💹' },
          ].map(item => (
            <NavLink key={item.to} to={item.to} end={item.to === '/'}
              className={({ isActive }) =>
                `flex items-center gap-2.5 px-3 py-2.5 rounded-xl text-sm transition-all duration-200 ${
                  isActive ? 'bg-[#C84B4B]/15 text-[#C84B4B] font-medium border border-[#C84B4B]/20' : 'text-gray-400 hover:text-white hover:bg-gray-800/50'
                }`}>
              <span className="text-base">{item.icon}</span>{item.label}
            </NavLink>
          ))}
        </nav>

        {/* Footer */}
        <div className="p-2.5 border-t border-gray-800/50 space-y-2">
          <button onClick={logout}
            className="w-full flex items-center justify-center gap-2 px-4 py-2.5 rounded-xl text-sm font-medium text-white bg-red-600/80 hover:bg-red-500 transition-all">
            <span>🚪</span>Sign Out
          </button>
          <p className="text-center text-[10px] text-gray-600">Aspira Pay · Sandbox V4</p>
        </div>
      </aside>

      {/* Main */}
      <main className="flex-1 overflow-auto bg-gradient-to-br from-gray-950 via-gray-950 to-gray-900">
        <div className="p-6 max-w-6xl"><Outlet /></div>
      </main>
    </div>
  )
}
