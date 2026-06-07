interface StatsCardProps {
  title: string
  value: string | number
  subtitle?: string
  icon?: string
  trend?: 'up' | 'down'
}

export default function StatsCard({ title, value, subtitle, icon, trend }: StatsCardProps) {
  return (
    <div className="bg-gray-900 border border-gray-800 rounded-xl p-6 hover:border-gray-700 transition-colors">
      <div className="flex items-start justify-between">
        <div>
          <p className="text-sm text-gray-500 mb-1">{title}</p>
          <p className="text-2xl font-bold text-white">{value}</p>
          {subtitle && <p className="text-xs text-gray-600 mt-1">{subtitle}</p>}
        </div>
        {icon && <span className="text-2xl">{icon}</span>}
      </div>
      {trend && (
        <div className={`text-xs mt-3 ${trend === 'up' ? 'text-green-400' : 'text-red-400'}`}>
          {trend === 'up' ? '↑' : '↓'} vs last period
        </div>
      )}
    </div>
  )
}
