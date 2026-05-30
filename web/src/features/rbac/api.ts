import {
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import { api, unwrap } from "@/lib/api";
import type { RbacPermission, RbacRole } from "@/lib/types";

// Roles and permissions are returned unpaginated as { success, data: [...] }.

export function useRoles() {
  return useQuery({
    queryKey: ["rbac-roles"],
    queryFn: async () => {
      const res = await api.get("/roles");
      return unwrap<RbacRole[]>(res.data) ?? [];
    },
  });
}

export function usePermissions() {
  return useQuery({
    queryKey: ["rbac-permissions"],
    queryFn: async () => {
      const res = await api.get("/permissions");
      return unwrap<RbacPermission[]>(res.data) ?? [];
    },
  });
}

export function useUserRoles(userId: number | undefined) {
  return useQuery({
    queryKey: ["user-roles", userId],
    enabled: userId != null,
    queryFn: async () => {
      const res = await api.get(`/users/${userId}/roles`);
      return unwrap<RbacRole[]>(res.data) ?? [];
    },
  });
}

export function useAssignRole(userId: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (roleId: number) => {
      // Backend expects { role_id } on POST /users/:id/roles/assign.
      await api.post(`/users/${userId}/roles/assign`, { role_id: roleId });
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["user-roles", userId] });
    },
  });
}

export function useRemoveRole(userId: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (roleId: number) => {
      // Backend route: DELETE /users/:id/roles/:roleId.
      await api.delete(`/users/${userId}/roles/${roleId}`);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["user-roles", userId] });
    },
  });
}
