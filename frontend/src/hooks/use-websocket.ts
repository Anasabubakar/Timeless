import { useEffect, useRef, useCallback } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { useAuthStore } from '@/stores/auth';

type EventType =
  | 'sponsor.updated'
  | 'sponsor.created'
  | 'campaign.updated'
  | 'agent.completed'
  | 'notification'
  | 'pipeline.move';

interface WSEvent {
  type: EventType;
  payload: Record<string, unknown>;
  org_id: string;
  timestamp: string;
}

const QUERY_INVALIDATION_MAP: Record<EventType, string[]> = {
  'sponsor.updated': ['sponsors'],
  'sponsor.created': ['sponsors'],
  'campaign.updated': ['campaigns'],
  'agent.completed': ['ai-agents'],
  'notification': [],
  'pipeline.move': ['sponsors'],
};

export function useWebSocket() {
  const wsRef = useRef<WebSocket | null>(null);
  const queryClient = useQueryClient();
  const token = useAuthStore((s) => s.accessToken);

  const connect = useCallback(() => {
    if (!token) return;

    const wsUrl = `${process.env.NEXT_PUBLIC_WS_URL || 'ws://localhost:8080'}/ws?token=${token}`;
    const ws = new WebSocket(wsUrl);

    ws.onopen = () => {
      console.log('[WS] connected');
    };

    ws.onmessage = (event) => {
      try {
        const data: WSEvent = JSON.parse(event.data);
        const keys = QUERY_INVALIDATION_MAP[data.type];
        if (keys) {
          keys.forEach((key) => {
            queryClient.invalidateQueries({ queryKey: [key] });
          });
        }
      } catch {
        // ignore malformed messages
      }
    };

    ws.onclose = () => {
      console.log('[WS] disconnected, reconnecting in 3s...');
      setTimeout(connect, 3000);
    };

    ws.onerror = () => {
      ws.close();
    };

    wsRef.current = ws;
  }, [token, queryClient]);

  useEffect(() => {
    connect();
    return () => {
      wsRef.current?.close();
    };
  }, [connect]);

  return wsRef;
}
