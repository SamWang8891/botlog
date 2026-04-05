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

export interface DisplayHit extends HitEvent {
  _id: string;
  _visible: boolean;
  _isNew: boolean;
}

const MAX_HITS = 200;
let idCounter = 0;

export function useSSE(url: string) {
  const [displayHits, setDisplayHits] = useState<DisplayHit[]>([]);
  const [connected, setConnected] = useState(false);
  const eventSourceRef = useRef<EventSource | null>(null);
  const retriesRef = useRef(0);
  const isFirstMessage = useRef(true);

  const connect = useCallback(() => {
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
    }

    isFirstMessage.current = true;

    const es = new EventSource(url);
    eventSourceRef.current = es;

    es.onopen = () => {
      setConnected(true);
      retriesRef.current = 0;
    };

    es.onmessage = (event) => {
      try {
        const newHits: HitEvent[] = JSON.parse(event.data);

        if (isFirstMessage.current && newHits.length > 1) {
          // Initial backfill: add all hidden, then reveal with staggered delay
          isFirstMessage.current = false;

          const tagged: DisplayHit[] = newHits.map(h => ({
            ...h,
            _id: `hit-${idCounter++}`,
            _visible: false,
            _isNew: false,
          }));

          setDisplayHits(tagged);

          // Stagger reveal: 2000ms / count ≈ 20ms each for 100 items
          const interval = Math.min(2000 / tagged.length, 30);
          tagged.forEach((_, idx) => {
            setTimeout(() => {
              setDisplayHits(prev => prev.map((h, i) =>
                i === idx ? { ...h, _visible: true } : h
              ));
            }, idx * interval);
          });
        } else {
          // Live hits: add at top, immediately visible, marked as new
          isFirstMessage.current = false;

          const tagged: DisplayHit[] = newHits.map(h => ({
            ...h,
            _id: `hit-${idCounter++}`,
            _visible: true,
            _isNew: true,
          }));

          setDisplayHits(prev => {
            const combined = [...tagged, ...prev];
            return combined.slice(0, MAX_HITS);
          });
        }
      } catch (e) {
        console.error('Failed to parse SSE data:', e);
      }
    };

    es.onerror = () => {
      setConnected(false);
      es.close();
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

  return { hits: displayHits, connected };
}
