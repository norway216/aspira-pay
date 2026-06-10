import { useEffect, useState, useCallback, useRef } from 'react'

const CURRENCIES = ['USD', 'EUR', 'GBP', 'JPY', 'CNY', 'HKD', 'SGD', 'CHF', 'AUD', 'CAD']
const FLAGS: Record<string, string> = {
  USD: '🇺🇸', EUR: '🇪🇺', GBP: '🇬🇧', JPY: '🇯🇵', CNY: '🇨🇳',
  HKD: '🇭🇰', SGD: '🇸🇬', CHF: '🇨🇭', AUD: '🇦🇺', CAD: '🇨🇦',
}

export default function FXRates() {
  const [rates, setRates] = useState<Record<string, string>>({})
  const [baseCurrency, setBaseCurrency] = useState('USD')
  const [lastUpdate, setLastUpdate] = useState('')
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const fetchRates = useCallback(async () => {
    try {
      const r = await fetch('/api/v2/fx/rates')
      const d = await r.json()
      if (d.rates) setRates(d.rates)
      setLastUpdate(new Date().toLocaleTimeString())
    } catch (e) { /* silent */ }
  }, [])

  useEffect(() => {
    fetchRates()
    timerRef.current = setInterval(fetchRates, 3000)
    return () => { if (timerRef.current) clearInterval(timerRef.current) }
  }, [fetchRates])

  const getRate = (from: string, to: string): string => {
    if (from === to) return '1.000000'
    const key1 = `${from}/${to}`
    const key2 = `${to}/${from}`
    if (rates[key1]) return parseFloat(rates[key1]).toFixed(6)
    if (rates[key2]) return (1 / parseFloat(rates[key2])).toFixed(6)
    return '-'
  }

  const getChangeColor = (from: string, to: string) => {
    const rate = parseFloat(getRate(from, to))
    if (isNaN(rate)) return 'text-gray-500'
    return rate > 1 ? 'text-green-400' : 'text-[#E07373]'
  }

  return (
    <div className="space-y-6 animate-fade-in">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold text-white">Live Exchange Rates</h2>
          <p className="text-gray-500 text-sm mt-1">Updated every 3s · Last: {lastUpdate || 'loading...'}</p>
        </div>
        <div className="flex items-center gap-2">
          <span className="text-xs text-gray-500">Base:</span>
          <select value={baseCurrency} onChange={e => setBaseCurrency(e.target.value)}
            className="bg-gray-800 border border-gray-700 rounded-lg px-3 py-1.5 text-white text-sm">
            {CURRENCIES.map(c => <option key={c}>{c}</option>)}
          </select>
        </div>
      </div>

      {/* Rate Cards */}
      <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-5 gap-3">
        {CURRENCIES.filter(c => c !== baseCurrency).map(currency => {
          const rate = getRate(baseCurrency, currency)
          return (
            <div key={currency} className="bg-gray-900/80 border border-gray-800/50 rounded-2xl p-4 hover:border-gray-700/50 transition-all">
              <div className="flex items-center gap-2 mb-2">
                <span className="text-xl">{FLAGS[currency] || '💱'}</span>
                <span className="text-sm font-medium text-white">{currency}</span>
              </div>
              <p className={`text-xl font-bold tabular-nums ${getChangeColor(baseCurrency, currency)}`}>
                {rate === '-' ? '—' : rate}
              </p>
              <p className="text-xs text-gray-600 mt-1">1 {baseCurrency} = {rate} {currency}</p>
            </div>
          )
        })}
      </div>

      {/* Full Rate Matrix */}
      <div className="bg-gray-900/80 border border-gray-800/50 rounded-2xl overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-800/50">
                <th className="px-4 py-3 text-left text-xs text-gray-500 uppercase tracking-wider font-medium sticky left-0 bg-gray-900/95">Currency</th>
                {CURRENCIES.map(c => (
                  <th key={c} className="px-4 py-3 text-right text-xs text-gray-500 uppercase tracking-wider font-medium">{c}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {CURRENCIES.map(from => (
                <tr key={from} className="border-b border-gray-800/30 hover:bg-gray-800/20 transition-colors">
                  <td className="px-4 py-3 font-medium text-white sticky left-0 bg-gray-900/95">
                    <span className="mr-2">{FLAGS[from]}</span>{from}
                  </td>
                  {CURRENCIES.map(to => {
                    const rate = getRate(from, to)
                    const isHighlight = rate !== '-' && parseFloat(rate) > 0 && from !== to
                    return (
                      <td key={to} className={`px-4 py-3 text-right tabular-nums font-mono text-xs ${
                        from === to ? 'text-gray-700' : isHighlight ? 'text-gray-300' : 'text-gray-600'
                      }`}>
                        {from === to ? '1.0000' : rate}
                      </td>
                    )
                  })}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}
