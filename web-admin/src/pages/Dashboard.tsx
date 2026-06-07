import { useEffect, useState } from 'react'
import { api, ensureAuth } from '../api/client'
import StatsCard from '../components/StatsCard'

export default function Dashboard() {
  const [stats, setStats] = useState<any>(null)
  const [error, setError] = useState('')
  const [authChecked, setAuthChecked] = useState(false)

  useEffect(() => {
    async function init() {
      try {
        // Auto-login for Sandbox
        await ensureAuth()
        setAuthChecked(true)
        // Now fetch dashboard data
        const data = await api.getDashboard()
        setStats(data)
      } catch (err: any) {
        setError(err.message)
      }
    }
    init()
  }, [])

  if (error) {
    return (
      <div className="bg-red-900/30 border border-red-800 rounded-lg p-4 text-red-400">
        Cannot connect to API: {error}
        <p className="text-sm mt-2">Make sure the API server is running on port 8080.</p>
      </div>
    )
  }

  if (!authChecked || !stats) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-gray-500 text-lg">
          <span className="inline-block animate-spin mr-3">⟳</span>
          Connecting to Aspira Pay V2...
        </div>
      </div>
    )
  }

  return (
    <div>
      <h2 className="text-2xl font-bold mb-6">Dashboard</h2>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
        <StatsCard title="Total Payments" value={stats?.total_payments || 0} icon="💱" />
        <StatsCard title="Total Users" value={stats?.total_users || 0} icon="👥" />
        <StatsCard title="Settlement Batches" value={stats?.total_settlement_batches || 0} icon="📒" />
        <StatsCard title="System Status" value={stats?.system_status || 'Unknown'} icon="🟢" />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="bg-gray-900 border border-gray-800 rounded-xl p-6">
          <h3 className="text-lg font-semibold mb-4">Quick Actions</h3>
          <div className="space-y-3">
            <a href="/transactions" className="block p-3 bg-gray-800 rounded-lg hover:bg-gray-700 transition-colors">
              💱 View Transactions
            </a>
            <a href="/users" className="block p-3 bg-gray-800 rounded-lg hover:bg-gray-700 transition-colors">
              👥 Manage Users
            </a>
            <a href="/audit" className="block p-3 bg-gray-800 rounded-lg hover:bg-gray-700 transition-colors">
              ⛓️ Blockchain Audit Explorer
            </a>
          </div>
        </div>

        <div className="bg-gray-900 border border-gray-800 rounded-xl p-6">
          <h3 className="text-lg font-semibold mb-4">System Info</h3>
          <dl className="space-y-2 text-sm">
            <div className="flex justify-between">
              <dt className="text-gray-500">Version</dt>
              <dd>2.0.0-sandbox</dd>
            </div>
            <div className="flex justify-between">
              <dt className="text-gray-500">Engine Status</dt>
              <dd className="text-green-400">{stats?.engine_status || 'connected'}</dd>
            </div>
            <div className="flex justify-between">
              <dt className="text-gray-500">API Endpoint</dt>
              <dd className="text-gray-400">/api/v2</dd>
            </div>
          </dl>
        </div>
      </div>
    </div>
  )
}
