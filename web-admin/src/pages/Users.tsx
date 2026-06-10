import { useCallback, useEffect, useState } from 'react'
import { api, ensureAuth } from '../api/client'
import { usePolling } from '../hooks/usePolling'

export default function Users() {
  const [users, setUsers] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [showRegister, setShowRegister] = useState(false)
  const [regForm, setRegForm] = useState({ username: '', email: '', password: '' })

  useEffect(() => { ensureAuth().catch(e => setError(e.message)) }, [])

  const loadUsers = useCallback(async () => {
    try {
      const data = await api.getUsers()
      setUsers(data.users || [])
      setError('')
    } catch (e: any) { if (!users.length) setError(e.message) }
    finally { setLoading(false) }
  }, [users.length])

  const { refresh } = usePolling(loadUsers, 5000)

  const handleRegister = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      await api.register(regForm.username, regForm.email, regForm.password)
      setShowRegister(false); setRegForm({ username: '', email: '', password: '' })
      await refresh()
    } catch (err: any) { alert(err.message) }
  }

  const toggleFreeze = async (userId: string, currentStatus: string) => {
    const newStatus = currentStatus === 'FROZEN' ? 'ACTIVE' : 'FROZEN'
    const action = newStatus === 'FROZEN' ? 'freeze' : 'unfreeze'
    if (!confirm(`${action.toUpperCase()} user ${userId}?`)) return
    try {
      await api.request(`/users/${userId}/status`, {
        method: 'PUT',
        body: JSON.stringify({ status: newStatus }),
      })
      refresh()
    } catch (err: any) { alert(err.message) }
  }

  if (error && !users.length) return (
    <div className="bg-red-900/30 border border-red-800 rounded-lg p-4 text-red-400">
      {error}
      <button onClick={refresh} className="ml-3 px-3 py-1 bg-red-800 rounded text-sm">Retry</button>
    </div>
  )

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-3">
          <h2 className="text-2xl font-bold text-white">Users</h2>
          <span className="text-xs text-gray-600 bg-gray-800 px-2 py-0.5 rounded">{users.length} total</span>
        </div>
        <div className="flex gap-2">
          <button onClick={refresh} className="px-3 py-2 bg-gray-800 hover:bg-gray-700 rounded-lg text-sm text-gray-400 hover:text-white transition-colors">⟳</button>
          <button onClick={() => setShowRegister(!showRegister)} className="px-4 py-2 bg-[#C84B4B] hover:bg-[#8B2E2E] rounded-lg text-sm font-medium transition-colors">+ Register</button>
        </div>
      </div>

      {showRegister && (
        <div className="bg-gray-900 border border-gray-800 rounded-xl p-6 mb-6 animate-fade-in">
          <h3 className="text-lg font-semibold mb-4">Register New User</h3>
          <form onSubmit={handleRegister} className="space-y-4 max-w-md">
            <div><label className="block text-sm text-gray-400 mb-1">Username</label><input type="text" value={regForm.username} onChange={e => setRegForm({...regForm, username: e.target.value})} className="w-full bg-gray-800 border border-gray-700 rounded-lg px-4 py-2 text-white" required /></div>
            <div><label className="block text-sm text-gray-400 mb-1">Email</label><input type="email" value={regForm.email} onChange={e => setRegForm({...regForm, email: e.target.value})} className="w-full bg-gray-800 border border-gray-700 rounded-lg px-4 py-2 text-white" required /></div>
            <div><label className="block text-sm text-gray-400 mb-1">Password</label><input type="password" value={regForm.password} onChange={e => setRegForm({...regForm, password: e.target.value})} className="w-full bg-gray-800 border border-gray-700 rounded-lg px-4 py-2 text-white" required /></div>
            <div className="flex gap-3"><button type="submit" className="px-6 py-2 bg-[#C84B4B] hover:bg-[#8B2E2E] rounded-lg text-sm">Register</button><button type="button" onClick={() => setShowRegister(false)} className="px-6 py-2 bg-gray-700 rounded-lg text-sm">Cancel</button></div>
          </form>
        </div>
      )}

      {loading ? <div className="text-gray-500">Loading...</div> : (
        <div className="bg-gray-900/80 border border-gray-800/50 rounded-2xl overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-800/50 text-left">
                <th className="px-4 py-3 text-xs text-gray-500 uppercase tracking-wider font-medium">User</th>
                <th className="px-4 py-3 text-xs text-gray-500 uppercase tracking-wider font-medium">Email</th>
                <th className="px-4 py-3 text-xs text-gray-500 uppercase tracking-wider font-medium">KYC</th>
                <th className="px-4 py-3 text-xs text-gray-500 uppercase tracking-wider font-medium">Status</th>
                <th className="px-4 py-3 text-xs text-gray-500 uppercase tracking-wider font-medium">Risk</th>
                <th className="px-4 py-3 text-xs text-gray-500 uppercase tracking-wider font-medium">Role</th>
                <th className="px-4 py-3 text-xs text-gray-500 uppercase tracking-wider font-medium">Actions</th>
              </tr>
            </thead>
            <tbody>
              {users.length === 0 && (
                <tr><td colSpan={7} className="px-5 py-16 text-center text-gray-600">
                  <span className="text-4xl block mb-3">👥</span>No users registered yet</td></tr>
              )}
              {users.map((u: any) => (
                <tr key={u.user_id} className="border-b border-gray-800/30 hover:bg-gray-800/20 transition-colors">
                  <td className="px-4 py-3">
                    <p className="text-white text-sm font-medium">{u.username}</p>
                    <p className="text-xs text-gray-600 font-mono">{u.user_id?.substring(0, 16)}...</p>
                  </td>
                  <td className="px-4 py-3 text-gray-400 text-xs">{u.email || '—'}</td>
                  <td className="px-4 py-3">
                    <span className={`inline-flex px-2 py-0.5 rounded-full text-xs font-medium ${
                      u.status === 'ACTIVE' ? 'bg-green-900/30 text-green-400' :
                      u.status === 'PENDING_KYC' ? 'bg-yellow-900/30 text-yellow-400' :
                      'bg-gray-800 text-gray-400'}`}>
                      {u.status === 'ACTIVE' ? '✓ Verified' : u.status === 'PENDING_KYC' ? 'Pending' : u.status}
                    </span>
                  </td>
                  <td className="px-4 py-3">
                    <span className={`inline-flex px-2 py-0.5 rounded-full text-xs font-medium ${
                      u.status === 'ACTIVE' ? 'bg-green-900/30 text-green-400' :
                      u.status === 'FROZEN' ? 'bg-red-900/30 text-red-400' :
                      'bg-yellow-900/30 text-yellow-400'}`}>{u.status}</span>
                  </td>
                  <td className="px-4 py-3">
                    <span className={`text-xs font-medium ${u.risk_level === 'HIGH' ? 'text-red-400' : u.risk_level === 'MEDIUM' ? 'text-yellow-400' : 'text-green-400'}`}>
                      {u.risk_level || 'LOW'}
                    </span>
                  </td>
                  <td className="px-4 py-3">
                    <span className="text-xs text-gray-500">{u.admin_key?.startsWith('aspira-') ? '🔑 Admin' : '👤 User'}</span>
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex gap-2">
                      <button
                        onClick={() => toggleFreeze(u.user_id, u.status)}
                        className={`px-2.5 py-1 rounded-lg text-xs font-medium transition-colors ${
                          u.status === 'FROZEN'
                            ? 'bg-green-900/30 hover:bg-green-800/30 text-green-400'
                            : 'bg-red-900/30 hover:bg-red-800/30 text-red-400'
                        }`}>
                        {u.status === 'FROZEN' ? 'Unfreeze' : 'Freeze'}
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Privacy Notice */}
      <div className="mt-4 p-4 bg-gray-900/40 border border-gray-800/30 rounded-xl text-xs text-gray-600 flex items-center gap-2">
        <span>🔒</span> Account balances are private. The system automatically checks balances during transactions.
        Admins manage identity, KYC status, and account freezes — not individual balances.
      </div>
    </div>
  )
}
