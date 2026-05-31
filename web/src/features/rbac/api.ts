import {
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import { api, unwrap } from "@/lib/api";
import type { RbacPermission, RbacRole } from "@/lib/types";

// The backend exposes a couple of fields the shared lib/types omit (notably
// `is_system` on permissions, plus timestamps). Extend locally rather than
// touching lib/types so existing consumers stay untouched.
export interface RbacPermissionFull extends RbacPermission {
  is_system?: boolean;
  created_at?: string;
  updated_at?: string;
}

export interface RbacRoleFull extends RbacRole {
  created_at?: string;
  updated_at?: string;
}

export interface RoleInput {
  name: string;
  description?: string;
}

export interface PermissionInput {
  code: string;
  name: string;
  description?: string;
  category?: string;
}

// Roles and permissions are returned unpaginated as { success, data: [...] }.

export function useRoles() {
  return useQuery({
    queryKey: ["rbac-roles"],
    queryFn: async () => {
      const res = await api.get("/roles");
      return unwrap<RbacRoleFull[]>(res.data) ?? [];
    },
  });
}

export function usePermissions() {
  return useQuery({
    queryKey: ["rbac-permissions"],
    queryFn: async () => {
      const res = await api.get("/permissions");
      return unwrap<RbacPermissionFull[]>(res.data) ?? [];
    },
  });
}

// --- Role mutations ---------------------------------------------------------

export function useCreateRole() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: RoleInput) => {
      const res = await api.post("/roles", input);
      return unwrap<RbacRoleFull>(res.data);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["rbac-roles"] });
    },
  });
}

export function useUpdateRole(roleId: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: RoleInput) => {
      const res = await api.put(`/roles/${roleId}`, input);
      return unwrap<RbacRoleFull>(res.data);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["rbac-roles"] });
    },
  });
}

export function useDeleteRole() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (roleId: number) => {
      await api.delete(`/roles/${roleId}`);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["rbac-roles"] });
    },
  });
}

// --- Role <-> permission wiring --------------------------------------------

export function useRolePermissions(roleId: number | undefined) {
  return useQuery({
    queryKey: ["role-permissions", roleId],
    enabled: roleId != null,
    queryFn: async () => {
      const res = await api.get(`/roles/${roleId}/permissions`);
      return unwrap<RbacPermissionFull[]>(res.data) ?? [];
    },
  });
}

export function useAssignPermissionToRole(roleId: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (permissionId: number) => {
      // Backend expects { permission_id } on POST /roles/:id/permissions/assign.
      await api.post(`/roles/${roleId}/permissions/assign`, {
        permission_id: permissionId,
      });
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["role-permissions", roleId] });
      qc.invalidateQueries({ queryKey: ["rbac-roles"] });
    },
  });
}

export function useRemovePermissionFromRole(roleId: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (permissionId: number) => {
      // Backend route: DELETE /roles/:id/permissions/:permissionId.
      await api.delete(`/roles/${roleId}/permissions/${permissionId}`);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["role-permissions", roleId] });
      qc.invalidateQueries({ queryKey: ["rbac-roles"] });
    },
  });
}

// --- Permission mutations ---------------------------------------------------

export function useCreatePermission() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: PermissionInput) => {
      const res = await api.post("/permissions", input);
      return unwrap<RbacPermissionFull>(res.data);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["rbac-permissions"] });
    },
  });
}

export function useUpdatePermission(permissionId: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: PermissionInput) => {
      const res = await api.put(`/permissions/${permissionId}`, input);
      return unwrap<RbacPermissionFull>(res.data);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["rbac-permissions"] });
      // Role permission sets may embed the renamed permission.
      qc.invalidateQueries({ queryKey: ["role-permissions"] });
      qc.invalidateQueries({ queryKey: ["rbac-roles"] });
    },
  });
}

export function useDeletePermission() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (permissionId: number) => {
      await api.delete(`/permissions/${permissionId}`);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["rbac-permissions"] });
      qc.invalidateQueries({ queryKey: ["role-permissions"] });
      qc.invalidateQueries({ queryKey: ["rbac-roles"] });
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
