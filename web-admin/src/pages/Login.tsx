import { useState } from 'react'

export default function Login({ onLogin }: { onLogin: (token: string, isAdmin: boolean, userId: string) => void }) {
  const [mode, setMode] = useState<'login' | 'register'>('login')
  const [form, setForm] = useState({
    username: '', email: '', password: '', admin_key: '',
    full_name: '', nationality: '', date_of_birth: '',
  })
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [step, setStep] = useState(0) // 0=account, 1=KYC (register only)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (mode === 'register' && step === 0) { setStep(1); return }
    setLoading(true); setError('')
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
          body: JSON.stringify({ username: form.username, email: form.email, password: form.password,
            admin_key: form.admin_key, full_name: form.full_name, nationality: form.nationality,
            date_of_birth: form.date_of_birth, default_currency: 'USD' }),
        })
        const d = await r.json()
        if (!r.ok) throw new Error(d.error || 'Registration failed')
        const lr = await fetch('/api/v2/auth/login', { method: 'POST', headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ username: form.username, password: form.password }) })
        const ld = await lr.json()
        if (!lr.ok) throw new Error(ld.error || 'Auto-login failed')
        localStorage.setItem('auth_token', ld.token)
        onLogin(ld.token, ld.is_admin, ld.user_id)
      }
    } catch (err: any) { setError(err.message) }
    finally { setLoading(false) }
  }

  return (
    <div className="min-h-screen bg-gray-950 flex items-center justify-center p-4">
      <div className="w-full max-w-md animate-fade-in">
        {/* Brand */}
        <div className="text-center mb-10">
          <div className="inline-flex items-center justify-center mb-4">
            <img src="/logo.png" alt="Aspira Pay" className="w-20 h-20 object-contain opacity-95" />
          </div>
          <h1 className="text-3xl font-bold text-white mb-1 tracking-tight">Aspira Pay</h1>
          <p className="text-gray-500 text-sm">Cross-Border Payment & Banking</p>
        </div>

        {/* Card */}
        <div className="bg-gray-900/80 border border-gray-800/60 rounded-2xl p-8 backdrop-blur-sm shadow-xl">
          {/* Tabs */}
          <div className="flex mb-8 bg-gray-800/50 rounded-xl p-1 gap-1">
            <button onClick={() => { setMode('login'); setStep(0); setError('') }}
              className={`flex-1 py-2.5 rounded-lg text-sm font-medium transition-all duration-200 ${mode === 'login' ? 'bg-[#C84B4B] text-white shadow-lg shadow-[#C84B4B]/20' : 'text-gray-400 hover:text-white'}`}>
              Sign In
            </button>
            <button onClick={() => { setMode('register'); setStep(0); setError('') }}
              className={`flex-1 py-2.5 rounded-lg text-sm font-medium transition-all duration-200 ${mode === 'register' ? 'bg-[#C84B4B] text-white shadow-lg shadow-[#C84B4B]/20' : 'text-gray-400 hover:text-white'}`}>
              Create Account
            </button>
          </div>

          {error && <div className="bg-red-900/30 border border-red-800/50 rounded-xl p-3.5 text-red-400 text-sm mb-5 flex items-start gap-2">
            <span className="shrink-0 mt-0.5">⚠</span>{error}</div>}

          <form onSubmit={handleSubmit} className="space-y-4">
            {mode === 'register' && step === 1 ? (
              <>
                <div className="text-center mb-2">
                  <p className="text-sm text-gray-300 font-medium">KYC Information</p>
                  <p className="text-xs text-gray-500 mt-1">Required for account verification</p>
                </div>
                <input type="text" value={form.full_name} onChange={e => setForm({...form, full_name: e.target.value})}
                  placeholder="Full Legal Name" required className="w-full bg-gray-800/80 border border-gray-700/60 rounded-xl px-4 py-3 text-white placeholder-gray-500 focus:border-[#C84B4B]/50 transition-colors" />
                <div className="grid grid-cols-2 gap-3">
                  <input type="text" value={form.nationality} onChange={e => setForm({...form, nationality: e.target.value})}
                    placeholder="Nationality" className="bg-gray-800/80 border border-gray-700/60 rounded-xl px-4 py-3 text-white placeholder-gray-500 focus:border-[#C84B4B]/50 transition-colors" />
                  <input type="text" value={form.date_of_birth} onChange={e => setForm({...form, date_of_birth: e.target.value})}
                    placeholder="DOB YYYY-MM-DD" className="bg-gray-800/80 border border-gray-700/60 rounded-xl px-4 py-3 text-white placeholder-gray-500 focus:border-[#C84B4B]/50 transition-colors" />
                </div>
                <div className="bg-gray-800/40 rounded-xl p-4">
                  <label className="text-sm text-gray-400">Admin Key <span className="text-gray-600 text-xs">(optional)</span></label>
                  <input type="text" value={form.admin_key} onChange={e => setForm({...form, admin_key: e.target.value})}
                    placeholder="aspira-xxx for admin access"
                    className="w-full bg-transparent mt-1 text-white font-mono text-sm placeholder-gray-600 focus:outline-none" />
                </div>
                <div className="flex gap-3 pt-2">
                  <button type="button" onClick={() => setStep(0)} className="flex-1 py-3 bg-gray-800 hover:bg-gray-700 rounded-xl text-sm font-medium transition-colors">← Back</button>
                  <button type="submit" disabled={loading} className="flex-1 py-3 bg-[#C84B4B] hover:bg-[#B04040] rounded-xl text-sm font-medium transition-colors disabled:opacity-50">
                    {loading ? 'Creating...' : 'Create Account'}</button>
                </div>
              </>
            ) : (
              <>
                <div>
                  <label className="block text-xs text-gray-500 mb-1.5 ml-1 uppercase tracking-wider">Username</label>
                  <input type="text" value={form.username} onChange={e => setForm({...form, username: e.target.value})}
                    required className="w-full bg-gray-800/80 border border-gray-700/60 rounded-xl px-4 py-3 text-white placeholder-gray-500 focus:border-[#C84B4B]/50 transition-colors" />
                </div>
                {mode === 'register' && (
                  <div>
                    <label className="block text-xs text-gray-500 mb-1.5 ml-1 uppercase tracking-wider">Email</label>
                    <input type="email" value={form.email} onChange={e => setForm({...form, email: e.target.value})}
                      required className="w-full bg-gray-800/80 border border-gray-700/60 rounded-xl px-4 py-3 text-white placeholder-gray-500 focus:border-[#C84B4B]/50 transition-colors" />
                  </div>
                )}
                <div>
                  <label className="block text-xs text-gray-500 mb-1.5 ml-1 uppercase tracking-wider">Password</label>
                  <input type="password" value={form.password} onChange={e => setForm({...form, password: e.target.value})}
                    required className="w-full bg-gray-800/80 border border-gray-700/60 rounded-xl px-4 py-3 text-white placeholder-gray-500 focus:border-[#C84B4B]/50 transition-colors" />
                </div>
                {mode === 'register' ? (
                  <button type="submit" disabled={loading} className="w-full py-3 bg-[#C84B4B] hover:bg-[#B04040] rounded-xl text-sm font-medium transition-colors disabled:opacity-50">
                    Continue →</button>
                ) : (
                  <button type="submit" disabled={loading} className="w-full py-3 bg-[#C84B4B] hover:bg-[#B04040] rounded-xl text-sm font-medium transition-colors disabled:opacity-50 mt-2">
                    {loading ? 'Signing in...' : 'Sign In'}</button>
                )}
              </>
            )}
          </form>
        </div>
        <p className="text-center text-xs text-gray-600 mt-6">Aspira Pay · Sandbox V4</p>
      </div>
    </div>
  )
}
