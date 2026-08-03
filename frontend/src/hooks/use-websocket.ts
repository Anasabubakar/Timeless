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

    // Derived from NEXT_PUBLIC_API_URL rather than a separate
    // NEXT_PUBLIC_WS_URL — a second URL env var that has to be kept in
    // sync by hand is exactly the kind of thing that gets set once and
    // forgotten on the next deploy, which is what happened here: the API
    // origin was configured correctly, but nothing derived the
    // WebSocket URL from it, so this fell back to the ws://localhost:8080
    // dev default in production, which the CSP (rightly) blocked outright.
    const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api/v1';
    const wsOrigin = apiUrl.replace(/^http/, 'ws').replace(/\/api\/v1\/?$/, '');
    const wsUrl = `${wsOrigin}/ws?token=${token}`;
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
