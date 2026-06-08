import { Outlet, NavLink, useNavigate } from 'react-router-dom'

export default function UserLayout() {
  const nav = useNavigate()
  const logout = () => { localStorage.removeItem('auth_token'); window.location.reload() }

  return (
    <div className="min-h-screen bg-gray-950 flex">
      <aside className="w-60 bg-gray-900 border-r border-gray-800 flex flex-col">
        <div className="p-5 border-b border-gray-800">
          <h1 className="text-lg font-bold bg-gradient-to-r from-emerald-400 to-cyan-300 bg-clip-text text-transparent">Aspira Pay</h1>
          <p className="text-xs text-gray-500 mt-1">Personal Banking</p>
        </div>
        <nav className="flex-1 p-3 space-y-1">
          {[
            { to: '/', label: 'Dashboard', icon: '📊' },
            { to: '/cards', label: 'My Cards', icon: '💳' },
            { to: '/payments', label: 'Payments', icon: '💱' },
          ].map(item => (
            <NavLink key={item.to} to={item.to} end={item.to === '/'}
              className={({ isActive }) =>
                `flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm transition-colors ${
                  isActive ? 'bg-emerald-600/20 text-emerald-400 border border-emerald-600/30' : 'text-gray-400 hover:text-white hover:bg-gray-800'
                }`}>
              <span>{item.icon}</span>{item.label}
            </NavLink>
          ))}
        </nav>
        <div className="p-3 border-t border-gray-800">
          <button onClick={logout} className="w-full text-left px-3 py-2 rounded-lg text-sm text-gray-500 hover:text-red-400 hover:bg-gray-800 transition-colors">
            🚪 Sign Out
          </button>
        </div>
      </aside>
      <main className="flex-1 overflow-auto"><div className="p-6"><Outlet /></div></main>
    </div>
  )
}
