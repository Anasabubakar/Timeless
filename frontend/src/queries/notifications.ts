import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";

export interface Notification {
  id: string;
  organization_id: string;
  user_id: string;
  type: string;
  title: string;
  body: string;
  action_url?: string;
  entity_type?: string;
  entity_id?: string;
  metadata: Record<string, unknown>;
  read: boolean;
  read_at?: string;
  created_at: string;
}

export interface NotificationPreference {
  id: string;
  type: string;
  in_app: boolean;
  email: boolean;
}

interface NotificationsResponse {
  data: Notification[];
  total: number;
}

export function useNotifications(unreadOnly = false, limit = 20, offset = 0) {
  const params = new URLSearchParams({
    limit: String(limit),
    offset: String(offset),
  });
  if (unreadOnly) params.set("unread", "true");

  return useQuery<NotificationsResponse>({
    queryKey: ["notifications", unreadOnly, limit, offset],
    queryFn: () => api.get(`/notifications?${params}`),
    refetchInterval: 30_000,
  });
}

export function useUnreadCount() {
  return useQuery<{ count: number }>({
    queryKey: ["notifications", "count"],
    queryFn: () => api.get("/notifications/count"),
    refetchInterval: 15_000,
  });
}

export function useMarkRead() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.patch(`/notifications/${id}/read`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["notifications"] });
    },
  });
}

export function useMarkAllRead() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => api.post("/notifications/read-all"),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["notifications"] });
    },
  });
}

export function useDeleteNotification() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.delete(`/notifications/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["notifications"] });
    },
  });
}

export function useNotificationPreferences() {
  return useQuery<{ data: NotificationPreference[] }>({
    queryKey: ["notifications", "preferences"],
    queryFn: () => api.get("/notifications/preferences"),
  });
}

export function useUpdateNotificationPreference() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: { type: string; in_app?: boolean; email?: boolean }) =>
      api.put("/notifications/preferences", data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["notifications", "preferences"] });
    },
  });
}
