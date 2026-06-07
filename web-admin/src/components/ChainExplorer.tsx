interface ChainExplorerProps {
  trail: {
    payment_id: string
    verified: boolean
    blocks: any[]
    events: any[]
  }
}

export default function ChainExplorer({ trail }: ChainExplorerProps) {
  return (
    <div className="bg-gray-900 border border-gray-800 rounded-xl p-6 mb-6">
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-lg font-semibold">
          Audit Trail: <span className="font-mono text-sm text-gray-400">{trail.payment_id}</span>
        </h3>
        <span className={`px-3 py-1 rounded-full text-xs font-medium ${
          trail.verified ? 'bg-green-900/50 text-green-400' : 'bg-red-900/50 text-red-400'
        }`}>
          {trail.verified ? '✓ Chain Verified' : '✗ Chain Broken'}
        </span>
      </div>

      <div className="space-y-3">
        <div className="text-sm text-gray-400 mb-2">
          Showing {trail.events.length} events across {trail.blocks.length} blocks
        </div>

        <div className="space-y-2">
          {trail.events.map((event: any, idx: number) => (
            <div key={idx} className="flex items-center gap-4 p-3 bg-gray-800/50 rounded-lg">
              <div className="w-2 h-2 rounded-full bg-blue-400 flex-shrink-0" />
              <div className="flex-1">
                <div className="text-sm font-medium">{event.event_type}</div>
                <div className="text-xs text-gray-500">Block #{event.block_height}</div>
              </div>
              <div className="text-xs text-gray-500">
                {new Date(event.created_at).toLocaleTimeString()}
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
