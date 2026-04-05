import { useState, useMemo } from 'react';
import {
  BarChart, Bar, LineChart, Line, AreaChart, Area,
  PieChart, Pie, Cell,
  XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend,
} from 'recharts';
import { useStats, useFilterOptions, downloadCSV, type Filters, type NameValue } from '../hooks/useStats';

const COLORS = ['#00ff88', '#00d4ff', '#b44dff', '#ff3366', '#ffdd00', '#ff8844', '#44ffcc', '#ff44aa', '#88ff44', '#4488ff'];

type ChartType = 'bar' | 'line' | 'area' | 'pie';

function ChartTypeSelector({ value, onChange }: { value: ChartType; onChange: (t: ChartType) => void }) {
  const types: { key: ChartType; label: string }[] = [
    { key: 'bar', label: 'BAR' },
    { key: 'line', label: 'LINE' },
    { key: 'area', label: 'AREA' },
    { key: 'pie', label: 'PIE' },
  ];
  return (
    <div className="flex gap-1">
      {types.map(t => (
        <button
          key={t.key}
          onClick={() => onChange(t.key)}
          className={`px-3 py-1 text-xs font-mono rounded border transition-all ${
            value === t.key
              ? 'bg-neon-blue/20 text-neon-blue border-neon-blue/50'
              : 'bg-dark-800 text-gray-500 border-dark-600 hover:text-gray-300'
          }`}
        >
          {t.label}
        </button>
      ))}
    </div>
  );
}

const customTooltipStyle = {
  backgroundColor: '#0f1520',
  border: '1px solid #1c2840',
  borderRadius: '8px',
  fontSize: '12px',
  fontFamily: 'monospace',
};

function TimelineChart({ data, chartType }: { data: { time: string; hits: number }[]; chartType: ChartType }) {
  const formatted = data.map(d => ({
    ...d,
    label: new Date(d.time).toLocaleString('en-US', { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit', hour12: false }),
  }));

  if (chartType === 'pie') {
    return (
      <ResponsiveContainer width="100%" height={300}>
        <PieChart>
          <Pie data={formatted.slice(-20)} dataKey="hits" nameKey="label" cx="50%" cy="50%" outerRadius={100} label>
            {formatted.slice(-20).map((_, i) => <Cell key={i} fill={COLORS[i % COLORS.length]} />)}
          </Pie>
          <Tooltip contentStyle={customTooltipStyle} />
        </PieChart>
      </ResponsiveContainer>
    );
  }

  const ChartComponent = chartType === 'line' ? LineChart : chartType === 'area' ? AreaChart : BarChart;

  return (
    <ResponsiveContainer width="100%" height={300}>
      <ChartComponent data={formatted}>
        <CartesianGrid strokeDasharray="3 3" stroke="#1c2840" />
        <XAxis dataKey="label" tick={{ fill: '#4a5568', fontSize: 10 }} angle={-30} textAnchor="end" height={60} />
        <YAxis tick={{ fill: '#4a5568', fontSize: 11 }} />
        <Tooltip contentStyle={customTooltipStyle} />
        {chartType === 'bar' && <Bar dataKey="hits" fill="#00d4ff" radius={[2, 2, 0, 0]} />}
        {chartType === 'line' && <Line type="monotone" dataKey="hits" stroke="#00ff88" strokeWidth={2} dot={false} />}
        {chartType === 'area' && <Area type="monotone" dataKey="hits" stroke="#b44dff" fill="#b44dff22" />}
      </ChartComponent>
    </ResponsiveContainer>
  );
}

function DistributionChart({ data, chartType }: { data: NameValue[]; chartType: ChartType }) {
  if (!data || data.length === 0) {
    return <div className="h-[250px] flex items-center justify-center text-dark-500 text-sm">No data</div>;
  }

  if (chartType === 'pie' || chartType === 'area') {
    return (
      <ResponsiveContainer width="100%" height={250}>
        <PieChart>
          <Pie data={data} dataKey="value" nameKey="name" cx="50%" cy="50%" outerRadius={80} label={({ name, percent }: { name?: string; percent?: number }) => `${name ?? ''} ${((percent ?? 0) * 100).toFixed(0)}%`}>
            {data.map((_, i) => <Cell key={i} fill={COLORS[i % COLORS.length]} />)}
          </Pie>
          <Tooltip contentStyle={customTooltipStyle} />
          <Legend wrapperStyle={{ fontSize: '11px' }} />
        </PieChart>
      </ResponsiveContainer>
    );
  }

  return (
    <ResponsiveContainer width="100%" height={250}>
      <BarChart data={data} layout="vertical">
        <CartesianGrid strokeDasharray="3 3" stroke="#1c2840" />
        <XAxis type="number" tick={{ fill: '#4a5568', fontSize: 11 }} />
        <YAxis dataKey="name" type="category" width={120} tick={{ fill: '#4a5568', fontSize: 10 }} />
        <Tooltip contentStyle={customTooltipStyle} />
        <Bar dataKey="value" fill="#00d4ff" radius={[0, 2, 2, 0]} />
      </BarChart>
    </ResponsiveContainer>
  );
}

function StatCard({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="bg-dark-800 border border-dark-600 rounded-lg p-4 border-glow">
      <div className="text-xs text-dark-500 uppercase tracking-wider mb-1">{label}</div>
      <div className="text-2xl font-bold text-neon-green glow-green">
        {typeof value === 'number' ? value.toLocaleString() : value}
      </div>
    </div>
  );
}

export default function Stats() {
  const now = useMemo(() => new Date(), []);
  const [filters, setFilters] = useState<Filters>({
    from: new Date(now.getTime() - 24 * 60 * 60 * 1000).toISOString(),
    to: now.toISOString(),
    country: '',
    method: '',
    path: '',
    agent: '',
  });
  const [timelineChartType, setTimelineChartType] = useState<ChartType>('bar');
  const [distChartType, setDistChartType] = useState<ChartType>('pie');

  const { timeline, countries, methods, endpoints, agents, overview, loading, refresh } = useStats(filters);
  const filterOptions = useFilterOptions();

  const updateFilter = (key: keyof Filters, value: string) => {
    setFilters(prev => ({ ...prev, [key]: value }));
  };

  return (
    <div className="space-y-6">
      {/* Filters */}
      <div className="bg-dark-800 border border-dark-600 rounded-lg p-4 border-glow">
        <div className="flex flex-wrap gap-3 items-end">
          <div className="flex flex-col gap-1">
            <label className="text-xs text-dark-500 uppercase">From</label>
            <input
              type="datetime-local"
              className="bg-dark-700 border border-dark-600 rounded px-3 py-1.5 text-xs text-gray-300 focus:border-neon-blue focus:outline-none"
              value={filters.from.slice(0, 16)}
              onChange={e => updateFilter('from', new Date(e.target.value).toISOString())}
            />
          </div>
          <div className="flex flex-col gap-1">
            <label className="text-xs text-dark-500 uppercase">To</label>
            <input
              type="datetime-local"
              className="bg-dark-700 border border-dark-600 rounded px-3 py-1.5 text-xs text-gray-300 focus:border-neon-blue focus:outline-none"
              value={filters.to.slice(0, 16)}
              onChange={e => updateFilter('to', new Date(e.target.value).toISOString())}
            />
          </div>
          <div className="flex flex-col gap-1">
            <label className="text-xs text-dark-500 uppercase">Country</label>
            <select
              className="bg-dark-700 border border-dark-600 rounded px-3 py-1.5 text-xs text-gray-300 focus:border-neon-blue focus:outline-none"
              value={filters.country}
              onChange={e => updateFilter('country', e.target.value)}
            >
              <option value="">All</option>
              {filterOptions.countries?.map(c => <option key={c} value={c}>{c}</option>)}
            </select>
          </div>
          <div className="flex flex-col gap-1">
            <label className="text-xs text-dark-500 uppercase">Method</label>
            <select
              className="bg-dark-700 border border-dark-600 rounded px-3 py-1.5 text-xs text-gray-300 focus:border-neon-blue focus:outline-none"
              value={filters.method}
              onChange={e => updateFilter('method', e.target.value)}
            >
              <option value="">All</option>
              {filterOptions.methods?.map(m => <option key={m} value={m}>{m}</option>)}
            </select>
          </div>
          <div className="flex flex-col gap-1">
            <label className="text-xs text-dark-500 uppercase">Path contains</label>
            <input
              type="text"
              className="bg-dark-700 border border-dark-600 rounded px-3 py-1.5 text-xs text-gray-300 focus:border-neon-blue focus:outline-none w-32"
              placeholder="/admin..."
              value={filters.path}
              onChange={e => updateFilter('path', e.target.value)}
            />
          </div>
          <div className="flex flex-col gap-1">
            <label className="text-xs text-dark-500 uppercase">Agent contains</label>
            <input
              type="text"
              className="bg-dark-700 border border-dark-600 rounded px-3 py-1.5 text-xs text-gray-300 focus:border-neon-blue focus:outline-none w-32"
              placeholder="curl..."
              value={filters.agent}
              onChange={e => updateFilter('agent', e.target.value)}
            />
          </div>
          <button
            onClick={refresh}
            className="px-4 py-1.5 bg-neon-blue/20 text-neon-blue border border-neon-blue/50 rounded text-xs font-bold hover:bg-neon-blue/30 transition-all"
          >
            REFRESH
          </button>
          <button
            onClick={() => downloadCSV(filters)}
            className="px-4 py-1.5 bg-neon-green/20 text-neon-green border border-neon-green/50 rounded text-xs font-bold hover:bg-neon-green/30 transition-all"
          >
            EXPORT CSV
          </button>
        </div>
      </div>

      {loading ? (
        <div className="text-center py-12 text-dark-500 animate-pulse-glow">Loading statistics...</div>
      ) : (
        <>
          {/* Overview cards */}
          {overview && (
            <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-3">
              <StatCard label="Total Hits" value={overview.total_hits} />
              <StatCard label="Countries" value={overview.unique_countries} />
              <StatCard label="Unique Paths" value={overview.unique_paths} />
              <StatCard label="Unique Agents" value={overview.unique_agents} />
              <StatCard label="With Payload" value={overview.with_body} />
            </div>
          )}

          {/* Timeline */}
          <div className="bg-dark-800 border border-dark-600 rounded-lg p-4 border-glow">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-sm font-bold text-neon-blue glow-blue uppercase tracking-wider">Hits Over Time</h2>
              <ChartTypeSelector value={timelineChartType} onChange={setTimelineChartType} />
            </div>
            <TimelineChart data={timeline} chartType={timelineChartType} />
          </div>

          {/* Distribution charts */}
          <div className="flex items-center justify-between">
            <h2 className="text-sm font-bold text-neon-purple uppercase tracking-wider">Distributions</h2>
            <ChartTypeSelector value={distChartType} onChange={setDistChartType} />
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="bg-dark-800 border border-dark-600 rounded-lg p-4 border-glow">
              <h3 className="text-xs font-bold text-dark-500 uppercase tracking-wider mb-3">By Country</h3>
              <DistributionChart data={countries} chartType={distChartType} />
            </div>
            <div className="bg-dark-800 border border-dark-600 rounded-lg p-4 border-glow">
              <h3 className="text-xs font-bold text-dark-500 uppercase tracking-wider mb-3">By Method</h3>
              <DistributionChart data={methods} chartType={distChartType} />
            </div>
            <div className="bg-dark-800 border border-dark-600 rounded-lg p-4 border-glow">
              <h3 className="text-xs font-bold text-dark-500 uppercase tracking-wider mb-3">Top Endpoints</h3>
              <DistributionChart data={endpoints} chartType={distChartType} />
            </div>
            <div className="bg-dark-800 border border-dark-600 rounded-lg p-4 border-glow">
              <h3 className="text-xs font-bold text-dark-500 uppercase tracking-wider mb-3">Top User Agents</h3>
              <DistributionChart data={agents} chartType={distChartType} />
            </div>
          </div>
        </>
      )}
    </div>
  );
}
