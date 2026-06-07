import { useEffect, useState } from 'react'
import { api, ensureAuth } from '../api/client'
import ChainExplorer from '../components/ChainExplorer'

export default function Audit() {
  const [blocks, setBlocks] = useState<any[]>([])
  const [searchId, setSearchId] = useState('')
  const [auditTrail, setAuditTrail] = useState<any>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    ensureAuth().then(() => api.getBlocks().then(data => setBlocks(data.blocks || [])).catch(console.error)).catch(e => setError(e.message))
  }, [])

  const handleAuditSearch = async () => {
    if (!searchId.trim()) return
    setLoading(true)
    try { const data = await api.getAudit(searchId); setAuditTrail(data) } catch (err: any) { alert(err.message) }
    finally { setLoading(false) }
  }

  if (error) return <div className="bg-red-900/30 border border-red-800 rounded-lg p-4 text-red-400">{error}</div>

  return (
    <div>
      <h2 className="text-2xl font-bold mb-6">Blockchain Audit Explorer</h2>
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
        <div className="bg-gray-900 border border-gray-800 rounded-xl p-6"><h3 className="text-lg font-semibold mb-4">Chain Status</h3><dl className="space-y-2 text-sm"><div className="flex justify-between"><dt className="text-gray-500">Block Height</dt><dd className="font-mono">{blocks.length}</dd></div><div className="flex justify-between"><dt className="text-gray-500">Latest Hash</dt><dd className="font-mono text-xs text-gray-400">{blocks[0]?.block_hash?.substring(0, 32)}...</dd></div><div className="flex justify-between"><dt className="text-gray-500">Mode</dt><dd className="text-green-400">Hash Chain (Sandbox)</dd></div></dl></div>
        <div className="bg-gray-900 border border-gray-800 rounded-xl p-6"><h3 className="text-lg font-semibold mb-4">Payment Audit Lookup</h3><div className="flex gap-3"><input type="text" value={searchId} onChange={e => setSearchId(e.target.value)} placeholder="Payment ID" className="flex-1 bg-gray-800 border border-gray-700 rounded-lg px-4 py-2 text-white font-mono text-sm" /><button onClick={handleAuditSearch} disabled={loading} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium">{loading ? '...' : 'Audit'}</button></div></div>
      </div>
      {auditTrail && <ChainExplorer trail={auditTrail} />}
      <div className="bg-gray-900 border border-gray-800 rounded-xl overflow-hidden"><table className="w-full"><thead><tr className="border-b border-gray-800 text-left"><th className="p-4 text-sm text-gray-500">Height</th><th className="p-4 text-sm text-gray-500">Block Hash</th><th className="p-4 text-sm text-gray-500">Prev Hash</th><th className="p-4 text-sm text-gray-500">Events</th><th className="p-4 text-sm text-gray-500">Time</th></tr></thead>
      <tbody>{blocks.length === 0 ? <tr><td colSpan={5} className="p-8 text-center text-gray-600">No blocks</td></tr> : blocks.map((b: any) => (<tr key={b.block_height} className="border-b border-gray-800/50"><td className="p-4 text-sm font-mono">{b.block_height}</td><td className="p-4 text-xs font-mono text-gray-400">{b.block_hash?.substring(0, 24)}...</td><td className="p-4 text-xs font-mono text-gray-600">{b.prev_hash?.substring(0, 24)}...</td><td className="p-4 text-sm">{b.event_count}</td><td className="p-4 text-sm text-gray-500">{new Date(b.created_at).toLocaleString()}</td></tr>))}</tbody></table></div>
    </div>
  )
}
