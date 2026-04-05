import { useEffect, useRef, useState, useCallback } from 'react';

export interface HitEvent {
  timestamp: string;
  method: string;
  path: string;
  user_agent: string;
  country: string;
  city: string;
  content_type: string;
  body_preview: string;
  body_size: number;
}

const MAX_HITS = 200;

export function useSSE(url: string) {
  const [hits, setHits] = useState<HitEvent[]>([]);
  const [connected, setConnected] = useState(false);
  const eventSourceRef = useRef<EventSource | null>(null);
  const retriesRef = useRef(0);

  const connect = useCallback(() => {
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
    }

    const es = new EventSource(url);
    eventSourceRef.current = es;

    es.onopen = () => {
      setConnected(true);
      retriesRef.current = 0;
    };

    es.onmessage = (event) => {
      try {
        const newHits: HitEvent[] = JSON.parse(event.data);
        setHits(prev => {
          const combined = [...newHits, ...prev];
          return combined.slice(0, MAX_HITS);
        });
      } catch (e) {
        console.error('Failed to parse SSE data:', e);
      }
    };

    es.onerror = () => {
      setConnected(false);
      es.close();
      // Backoff: 1s, 2s, 4s, 8s, max 15s
      const delay = Math.min(1000 * Math.pow(2, retriesRef.current), 15000);
      retriesRef.current++;
      setTimeout(connect, delay);
    };
  }, [url]);

  useEffect(() => {
    connect();
    return () => {
      eventSourceRef.current?.close();
    };
  }, [connect]);

  return { hits, connected };
}
