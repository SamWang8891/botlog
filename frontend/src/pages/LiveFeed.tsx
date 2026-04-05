import { useSSE, type HitEvent } from '../hooks/useSSE';
import MethodBadge from '../components/MethodBadge';

function formatTime(ts: string) {
  const d = new Date(ts);
  return d.toLocaleTimeString('en-US', { hour12: false, hour: '2-digit', minute: '2-digit', second: '2-digit' });
}

function truncateUA(ua: string, max = 50) {
  if (ua.length <= max) return ua;
  return ua.slice(0, max) + '...';
}

function HitRow({ hit }: { hit: HitEvent }) {
  return (
    <tr className="animate-fade-in border-b border-dark-700/50 hover:bg-dark-700/30 transition-colors">
      <td className="px-3 py-2 text-xs text-neon-green/80 whitespace-nowrap font-mono">
        {formatTime(hit.timestamp)}
      </td>
      <td className="px-3 py-2">
        <MethodBadge method={hit.method} />
      </td>
      <td className="px-3 py-2 text-xs text-neon-blue font-mono truncate max-w-[200px]" title={hit.path}>
        {hit.path}
      </td>
      <td className="px-3 py-2 text-xs text-gray-300 whitespace-nowrap">
        <span className="text-neon-purple/80">{hit.city}</span>
        <span className="text-dark-500 mx-1">/</span>
        <span>{hit.country}</span>
      </td>
      <td className="hidden md:table-cell px-3 py-2 text-xs text-gray-500 truncate max-w-[300px]" title={hit.user_agent}>
        {truncateUA(hit.user_agent)}
      </td>
      <td className="hidden lg:table-cell px-3 py-2 text-xs">
        {hit.body_size > 0 ? (
          <span className="text-neon-yellow" title={hit.body_preview}>
            {hit.body_size}B
          </span>
        ) : (
          <span className="text-dark-500">-</span>
        )}
      </td>
    </tr>
  );
}

export default function LiveFeed() {
  const { hits, connected } = useSSE('/api/hits/live');

  return (
    <div>
      {/* Status bar */}
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <div className={`w-2 h-2 rounded-full ${connected ? 'bg-neon-green animate-pulse-glow' : 'bg-neon-red'}`} />
          <span className="text-xs text-gray-400">
            {connected ? 'LIVE — streaming bot traffic' : 'RECONNECTING...'}
          </span>
        </div>
        <span className="text-xs text-dark-500">{hits.length} events buffered</span>
      </div>

      {/* Table */}
      <div className="border border-dark-600 rounded-lg overflow-hidden border-glow">
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="bg-dark-800 border-b border-dark-600">
                <th className="px-3 py-3 text-left text-xs font-medium text-dark-500 uppercase tracking-wider">Time</th>
                <th className="px-3 py-3 text-left text-xs font-medium text-dark-500 uppercase tracking-wider">Method</th>
                <th className="px-3 py-3 text-left text-xs font-medium text-dark-500 uppercase tracking-wider">Endpoint</th>
                <th className="px-3 py-3 text-left text-xs font-medium text-dark-500 uppercase tracking-wider">Region</th>
                <th className="hidden md:table-cell px-3 py-3 text-left text-xs font-medium text-dark-500 uppercase tracking-wider">User Agent</th>
                <th className="hidden lg:table-cell px-3 py-3 text-left text-xs font-medium text-dark-500 uppercase tracking-wider">Body</th>
              </tr>
            </thead>
            <tbody>
              {hits.length === 0 ? (
                <tr>
                  <td colSpan={6} className="px-3 py-12 text-center text-dark-500 text-sm">
                    <div className="animate-pulse-glow">Waiting for bot traffic...</div>
                    <div className="text-xs mt-2 text-dark-600">Hits will appear here in real-time</div>
                  </td>
                </tr>
              ) : (
                hits.map((hit, i) => <HitRow key={`${hit.timestamp}-${i}`} hit={hit} />)
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
