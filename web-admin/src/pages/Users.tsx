import { useEffect, useState } from 'react'
import { api, ensureAuth } from '../api/client'

export default function Users() {
  const [users, setUsers] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [showRegister, setShowRegister] = useState(false)
  const [regForm, setRegForm] = useState({ username: '', email: '', password: '' })

  useEffect(() => {
    ensureAuth().then(() => loadUsers()).catch(e => setError(e.message))
  }, [])

  const loadUsers = async () => {
    try {
      const data = await api.getUsers()
      setUsers(data.users || [])
    } catch (e: any) { setError(e.message) } finally { setLoading(false) }
  }

  const handleRegister = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      await api.register(regForm.username, regForm.email, regForm.password)
      setShowRegister(false)
      loadUsers()
    } catch (err: any) { alert(err.message) }
  }

  if (error) return <div className="bg-red-900/30 border border-red-800 rounded-lg p-4 text-red-400">{error}</div>

  return (
    <div>
      <div className="flex items-center justify-between mb-6"><h2 className="text-2xl font-bold">Users</h2><button onClick={() => setShowRegister(!showRegister)} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium">+ Register</button></div>
      {showRegister && (<div className="bg-gray-900 border border-gray-800 rounded-xl p-6 mb-6"><h3 className="text-lg font-semibold mb-4">Register New User</h3><form onSubmit={handleRegister} className="space-y-4 max-w-md"><div><label className="block text-sm text-gray-400 mb-1">Username</label><input type="text" value={regForm.username} onChange={e => setRegForm({...regForm, username: e.target.value})} className="w-full bg-gray-800 border border-gray-700 rounded-lg px-4 py-2 text-white" required /></div><div><label className="block text-sm text-gray-400 mb-1">Email</label><input type="email" value={regForm.email} onChange={e => setRegForm({...regForm, email: e.target.value})} className="w-full bg-gray-800 border border-gray-700 rounded-lg px-4 py-2 text-white" required /></div><div><label className="block text-sm text-gray-400 mb-1">Password</label><input type="password" value={regForm.password} onChange={e => setRegForm({...regForm, password: e.target.value})} className="w-full bg-gray-800 border border-gray-700 rounded-lg px-4 py-2 text-white" required /></div><div className="flex gap-3"><button type="submit" className="px-6 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm">Register</button><button type="button" onClick={() => setShowRegister(false)} className="px-6 py-2 bg-gray-700 rounded-lg text-sm">Cancel</button></div></form></div>)}
      {loading ? <div className="text-gray-500">Loading...</div> : (
        <div className="bg-gray-900 border border-gray-800 rounded-xl overflow-hidden"><table className="w-full"><thead><tr className="border-b border-gray-800 text-left"><th className="p-4 text-sm text-gray-500">User ID</th><th className="p-4 text-sm text-gray-500">Username</th><th className="p-4 text-sm text-gray-500">Email</th><th className="p-4 text-sm text-gray-500">Status</th><th className="p-4 text-sm text-gray-500">Risk Level</th></tr></thead>
        <tbody>{users.length === 0 ? <tr><td colSpan={5} className="p-8 text-center text-gray-600">No users</td></tr> : users.map((u: any) => (<tr key={u.user_id} className="border-b border-gray-800/50 hover:bg-gray-800/30"><td className="p-4 text-sm font-mono text-gray-300">{u.user_id}</td><td className="p-4 text-sm">{u.username}</td><td className="p-4 text-sm text-gray-400">{u.email}</td><td className="p-4 text-sm">{u.status}</td><td className="p-4 text-sm">{u.risk_level}</td></tr>))}</tbody></table></div>
      )}
    </div>
  )
}
