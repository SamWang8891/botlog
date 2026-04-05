import { useState, useEffect, useCallback } from 'react';

export interface Filters {
  from: string;
  to: string;
  country: string;
  method: string;
  path: string;
  agent: string;
}

export interface NameValue {
  name: string;
  value: number;
}

export interface TimePoint {
  time: string;
  hits: number;
}

export interface Overview {
  total_hits: number;
  unique_countries: number;
  unique_paths: number;
  unique_agents: number;
  with_body: number;
}

export interface FilterOptions {
  countries: string[];
  methods: string[];
}

function buildQuery(filters: Filters): string {
  const params = new URLSearchParams();
  if (filters.from) params.set('from', filters.from);
  if (filters.to) params.set('to', filters.to);
  if (filters.country) params.set('country', filters.country);
  if (filters.method) params.set('method', filters.method);
  if (filters.path) params.set('path', filters.path);
  if (filters.agent) params.set('agent', filters.agent);
  return params.toString();
}

async function fetchJSON<T>(url: string): Promise<T | null> {
  try {
    const res = await fetch(url);
    if (!res.ok) return null;
    return await res.json();
  } catch {
    return null;
  }
}

export function useStats(filters: Filters) {
  const [timeline, setTimeline] = useState<TimePoint[]>([]);
  const [countries, setCountries] = useState<NameValue[]>([]);
  const [methods, setMethods] = useState<NameValue[]>([]);
  const [endpoints, setEndpoints] = useState<NameValue[]>([]);
  const [agents, setAgents] = useState<NameValue[]>([]);
  const [overview, setOverview] = useState<Overview | null>(null);
  const [loading, setLoading] = useState(true);

  const refresh = useCallback(async () => {
    setLoading(true);
    const q = buildQuery(filters);
    const [tl, co, me, ep, ag, ov] = await Promise.all([
      fetchJSON<TimePoint[]>(`/api/stats/timeline?${q}`),
      fetchJSON<NameValue[]>(`/api/stats/countries?${q}`),
      fetchJSON<NameValue[]>(`/api/stats/methods?${q}`),
      fetchJSON<NameValue[]>(`/api/stats/endpoints?${q}`),
      fetchJSON<NameValue[]>(`/api/stats/agents?${q}`),
      fetchJSON<Overview>(`/api/stats/overview?${q}`),
    ]);
    setTimeline(tl || []);
    setCountries(co || []);
    setMethods(me || []);
    setEndpoints(ep || []);
    setAgents(ag || []);
    setOverview(ov || null);
    setLoading(false);
  }, [filters]);

  useEffect(() => {
    refresh();
  }, [refresh]);

  return { timeline, countries, methods, endpoints, agents, overview, loading, refresh };
}

export function useFilterOptions() {
  const [options, setOptions] = useState<FilterOptions>({ countries: [], methods: [] });

  useEffect(() => {
    fetchJSON<FilterOptions>('/api/filters').then(data => {
      if (data) setOptions(data);
    });
  }, []);

  return options;
}

export function downloadCSV(filters: Filters) {
  const q = buildQuery(filters);
  window.open(`/api/export/csv?${q}`, '_blank');
}
