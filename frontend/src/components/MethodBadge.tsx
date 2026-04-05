const METHOD_COLORS: Record<string, string> = {
  GET: 'bg-neon-green/20 text-neon-green border-neon-green/30',
  POST: 'bg-neon-blue/20 text-neon-blue border-neon-blue/30',
  PUT: 'bg-neon-yellow/20 text-neon-yellow border-neon-yellow/30',
  DELETE: 'bg-neon-red/20 text-neon-red border-neon-red/30',
  PATCH: 'bg-neon-purple/20 text-neon-purple border-neon-purple/30',
  HEAD: 'bg-gray-500/20 text-gray-400 border-gray-500/30',
  OPTIONS: 'bg-gray-500/20 text-gray-400 border-gray-500/30',
};

export default function MethodBadge({ method }: { method: string }) {
  const colors = METHOD_COLORS[method] || METHOD_COLORS.GET;
  return (
    <span className={`inline-block px-2 py-0.5 text-xs font-bold rounded border ${colors}`}>
      {method}
    </span>
  );
}
