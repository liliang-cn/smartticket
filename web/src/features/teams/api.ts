import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api, unwrap } from "@/lib/api";

export interface Team {
  id: number;
  name: string;
  description: string;
}

export interface TeamMember {
  id: number;
  email: string;
  username: string;
  first_name: string;
  last_name: string;
  role: string;
}

export interface CreateTeamInput {
  name: string;
  description: string;
}

export interface UpdateTeamInput {
  name?: string;
  description?: string;
}

export function useTeams() {
  return useQuery({
    queryKey: ["teams"],
    queryFn: async () => {
      const res = await api.get("/teams");
      return unwrap<Team[]>(res.data) ?? [];
    },
  });
}

export function useTeamMembers(teamId: number | null) {
  return useQuery({
    queryKey: ["teams", teamId, "members"],
    enabled: teamId != null,
    queryFn: async () => {
      const res = await api.get(`/teams/${teamId}/members`);
      return unwrap<TeamMember[]>(res.data) ?? [];
    },
  });
}

export function useCreateTeam() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: CreateTeamInput) => {
      const res = await api.post("/teams", input);
      return unwrap<Team>(res.data);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["teams"] });
    },
  });
}

export function useUpdateTeam(id: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: UpdateTeamInput) => {
      const res = await api.put(`/teams/${id}`, input);
      return unwrap<Team>(res.data);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["teams"] });
    },
  });
}

export function useDeleteTeam() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (id: number) => {
      await api.delete(`/teams/${id}`);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["teams"] });
    },
  });
}

export function useAddTeamMember(teamId: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (userId: number) => {
      await api.post(`/teams/${teamId}/members`, { user_id: userId });
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["teams", teamId, "members"] });
    },
  });
}

export function useRemoveTeamMember(teamId: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (userId: number) => {
      await api.delete(`/teams/${teamId}/members/${userId}`);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["teams", teamId, "members"] });
    },
  });
}
