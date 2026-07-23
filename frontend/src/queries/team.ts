import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";

export interface TeamMember {
  id: string;
  email: string;
  first_name: string;
  last_name: string;
  avatar_url?: string;
  job_title?: string;
  status: string;
  roles: string[];
  created_at: string;
}

export interface OrgRole {
  id: string;
  name: string;
  permissions: string[];
  is_system: boolean;
}

export function useTeamMembers() {
  return useQuery({
    queryKey: ["team", "members"],
    queryFn: () => api.get<{ data: TeamMember[] }>("/team/members"),
  });
}

export function useOrgRoles() {
  return useQuery({
    queryKey: ["team", "roles"],
    queryFn: () => api.get<{ data: OrgRole[] }>("/team/roles"),
  });
}

export function useInviteMember() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: { email: string; first_name: string; last_name: string; role: string }) =>
      api.post("/team/members", data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["team", "members"] });
    },
  });
}

export function useUpdateMemberRoles() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, roles }: { id: string; roles: string[] }) =>
      api.patch(`/team/members/${id}/roles`, { roles }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["team", "members"] });
    },
  });
}

export function useRemoveMember() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.delete(`/team/members/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["team", "members"] });
    },
  });
}
