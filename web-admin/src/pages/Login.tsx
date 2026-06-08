import { useState } from 'react'

export default function Login({ onLogin }: { onLogin: (token: string, isAdmin: boolean, userId: string) => void }) {
  const [mode, setMode] = useState<'login' | 'register'>('login')
  const [form, setForm] = useState({
    username: '', email: '', password: '', admin_key: '',
    full_name: '', nationality: '', date_of_birth: '',
  })
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setError('')
    try {
      if (mode === 'login') {
        const r = await fetch('/api/v2/auth/login', {
          method: 'POST', headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ username: form.username, password: form.password }),
        })
        const d = await r.json()
        if (!r.ok) throw new Error(d.error || 'Login failed')
        localStorage.setItem('auth_token', d.token)
        onLogin(d.token, d.is_admin, d.user_id)
      } else {
        const r = await fetch('/api/v2/auth/register', {
          method: 'POST', headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            username: form.username, email: form.email, password: form.password,
            admin_key: form.admin_key, full_name: form.full_name,
            nationality: form.nationality, date_of_birth: form.date_of_birth,
            default_currency: 'USD',
          }),
        })
        const d = await r.json()
        if (!r.ok) throw new Error(d.error || 'Registration failed')
        // Auto-login after registration
        const lr = await fetch('/api/v2/auth/login', {
          method: 'POST', headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ username: form.username, password: form.password }),
        })
        const ld = await lr.json()
        if (!lr.ok) throw new Error(ld.error || 'Auto-login failed')
        localStorage.setItem('auth_token', ld.token)
        onLogin(ld.token, ld.is_admin, ld.user_id)
      }
    } catch (err: any) {
      setError(err.message)
    } finally { setLoading(false) }
  }

  return (
    <div className="min-h-screen bg-gray-950 flex items-center justify-center">
      <div className="w-full max-w-md">
        <div className="text-center mb-8">
          <h1 className="text-3xl font-bold bg-gradient-to-r from-blue-400 to-cyan-300 bg-clip-text text-transparent">
            Aspira Pay
          </h1>
          <p className="text-gray-500 mt-2">Cross-Border Payment System</p>
        </div>

        <div className="bg-gray-900 border border-gray-800 rounded-2xl p-8">
          {/* Mode Tabs */}
          <div className="flex mb-6 bg-gray-800 rounded-lg p-1">
            <button onClick={() => setMode('login')}
              className={`flex-1 py-2 rounded-md text-sm font-medium transition-colors ${mode === 'login' ? 'bg-blue-600 text-white' : 'text-gray-400 hover:text-white'}`}>
              Sign In
            </button>
            <button onClick={() => setMode('register')}
              className={`flex-1 py-2 rounded-md text-sm font-medium transition-colors ${mode === 'register' ? 'bg-blue-600 text-white' : 'text-gray-400 hover:text-white'}`}>
              Register
            </button>
          </div>

          {error && <div className="bg-red-900/30 border border-red-800 rounded-lg p-3 text-red-400 text-sm mb-4">{error}</div>}

          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label className="block text-sm text-gray-400 mb-1">Username</label>
              <input type="text" value={form.username} onChange={e => setForm({ ...form, username: e.target.value })}
                className="w-full bg-gray-800 border border-gray-700 rounded-lg px-4 py-2.5 text-white" required />
            </div>

            {mode === 'register' && (
              <div>
                <label className="block text-sm text-gray-400 mb-1">Email</label>
                <input type="email" value={form.email} onChange={e => setForm({ ...form, email: e.target.value })}
                  className="w-full bg-gray-800 border border-gray-700 rounded-lg px-4 py-2.5 text-white" required />
              </div>
            )}

            <div>
              <label className="block text-sm text-gray-400 mb-1">Password</label>
              <input type="password" value={form.password} onChange={e => setForm({ ...form, password: e.target.value })}
                className="w-full bg-gray-800 border border-gray-700 rounded-lg px-4 py-2.5 text-white" required />
            </div>

            {mode === 'register' && (
              <>
                <div className="border-t border-gray-800 pt-4 mt-4">
                  <p className="text-xs text-gray-500 mb-3">KYC Information (required for card applications)</p>
                  <div className="space-y-3">
                    <input type="text" value={form.full_name} onChange={e => setForm({ ...form, full_name: e.target.value })}
                      placeholder="Full Legal Name" className="w-full bg-gray-800 border border-gray-700 rounded-lg px-4 py-2.5 text-white" />
                    <div className="grid grid-cols-2 gap-3">
                      <input type="text" value={form.nationality} onChange={e => setForm({ ...form, nationality: e.target.value })}
                        placeholder="Nationality (e.g. US)" className="bg-gray-800 border border-gray-700 rounded-lg px-4 py-2.5 text-white" />
                      <input type="text" value={form.date_of_birth} onChange={e => setForm({ ...form, date_of_birth: e.target.value })}
                        placeholder="DOB (YYYY-MM-DD)" className="bg-gray-800 border border-gray-700 rounded-lg px-4 py-2.5 text-white" />
                    </div>
                  </div>
                </div>

                <div className="border-t border-gray-800 pt-4">
                  <label className="block text-sm text-gray-400 mb-1">
                    Admin Key <span className="text-gray-600">(optional — "aspira-" prefix = admin account)</span>
                  </label>
                  <input type="text" value={form.admin_key} onChange={e => setForm({ ...form, admin_key: e.target.value })}
                    placeholder="aspira-xxx" className="w-full bg-gray-800 border border-gray-700 rounded-lg px-4 py-2.5 text-white font-mono text-sm" />
                </div>
              </>
            )}

            <button type="submit" disabled={loading}
              className="w-full py-2.5 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition-colors disabled:opacity-50">
              {loading ? 'Please wait...' : mode === 'login' ? 'Sign In' : 'Create Account'}
            </button>
          </form>
        </div>
      </div>
    </div>
  )
}
