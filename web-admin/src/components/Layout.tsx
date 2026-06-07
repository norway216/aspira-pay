import { Outlet, NavLink } from 'react-router-dom'

export default function Layout() {
  const navItems = [
    { to: '/', label: 'Dashboard', icon: '📊' },
    { to: '/transactions', label: 'Transactions', icon: '💱' },
    { to: '/users', label: 'Users', icon: '👥' },
    { to: '/ledger', label: 'Ledger', icon: '📒' },
    { to: '/audit', label: 'Audit Chain', icon: '⛓️' },
  ]

  return (
    <div className="min-h-screen bg-gray-950 flex">
      {/* Sidebar */}
      <aside className="w-64 bg-gray-900 border-r border-gray-800 flex flex-col">
        <div className="p-6 border-b border-gray-800">
          <h1 className="text-xl font-bold bg-gradient-to-r from-blue-400 to-cyan-300 bg-clip-text text-transparent">
            Aspira Pay V2
          </h1>
          <p className="text-xs text-gray-500 mt-1">Cross-Border Payment System</p>
        </div>

        <nav className="flex-1 p-4 space-y-1">
          {navItems.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.to === '/'}
              className={({ isActive }) =>
                `flex items-center gap-3 px-4 py-3 rounded-lg text-sm transition-colors ${
                  isActive
                    ? 'bg-blue-600/20 text-blue-400 border border-blue-600/30'
                    : 'text-gray-400 hover:text-white hover:bg-gray-800'
                }`
              }
            >
              <span>{item.icon}</span>
              {item.label}
            </NavLink>
          ))}
        </nav>

        <div className="p-4 border-t border-gray-800">
          <div className="text-xs text-gray-600">
            Version 2.0.0-sandbox
          </div>
        </div>
      </aside>

      {/* Main content */}
      <main className="flex-1 overflow-auto">
        <div className="p-8">
          <Outlet />
        </div>
      </main>
    </div>
  )
}
