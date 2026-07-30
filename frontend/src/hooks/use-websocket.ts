import { useEffect, useRef, useCallback } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { useAuthStore } from '@/stores/auth';
import { useRealtimeStore } from '@/stores/realtime';

type EventType =
  | 'sponsor.updated'
  | 'sponsor.created'
  | 'campaign.updated'
  | 'agent.completed'
  | 'notification'
  | 'pipeline.move'
  | 'activity';

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
  // 'activity' drives the notification banner (see RealtimeActivityBanner)
  // rather than an automatic silent refetch — the whole point is to ask
  // the user before replacing what's on their screen, not do it for them.
  'activity': [],
};

export function useWebSocket() {
  const wsRef = useRef<WebSocket | null>(null);
  const queryClient = useQueryClient();
  const token = useAuthStore((s) => s.tokens?.access_token);
  const currentUserEmail = useAuthStore((s) => s.user?.email);
  const pushActivity = useRealtimeStore((s) => s.push);

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

        if (data.type === 'activity') {
          const actorEmail = String(data.payload.actor_email ?? '');
          // Don't notify someone about their own edit — they already
          // know, and it'd otherwise flash a banner after every save.
          if (actorEmail && actorEmail !== currentUserEmail) {
            pushActivity({
              actorEmail,
              action: String(data.payload.action ?? 'updated'),
              entityType: String(data.payload.entity_type ?? ''),
              subject: String(data.payload.subject ?? ''),
            });
          }
          return;
        }

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
  }, [token, queryClient, currentUserEmail, pushActivity]);

  useEffect(() => {
    connect();
    return () => {
      wsRef.current?.close();
    };
  }, [connect]);

  return wsRef;
}
