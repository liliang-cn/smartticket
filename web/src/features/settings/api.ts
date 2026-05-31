import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api, unwrap } from "@/lib/api";
import type { Branding } from "@/lib/types";

export interface BrandingUpdate {
  app_name?: string;
  app_subtitle?: string;
  workspace_name?: string;
  primary_color?: string;
  login_tagline?: string;
  login_subtext?: string;
}

/** Patch the branding configuration (admin only). */
export function useUpdateBranding() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (body: BrandingUpdate) => {
      const res = await api.put("/settings/branding", body);
      return unwrap<Branding>(res.data);
    },
    onSuccess: (data) => {
      qc.setQueryData(["branding"], data);
      qc.invalidateQueries({ queryKey: ["branding"] });
    },
  });
}

/** Upload a logo image (admin only). */
export function useUploadLogo() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (file: File) => {
      const form = new FormData();
      form.append("file", file);
      const res = await api.post("/settings/branding/logo", form, {
        headers: { "Content-Type": "multipart/form-data" },
      });
      return unwrap<Branding>(res.data);
    },
    onSuccess: (data) => {
      qc.setQueryData(["branding"], data);
      qc.invalidateQueries({ queryKey: ["branding"] });
    },
  });
}

/** Remove the uploaded logo (admin only). */
export function useDeleteLogo() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      const res = await api.delete("/settings/branding/logo");
      return unwrap<Branding>(res.data);
    },
    onSuccess: (data) => {
      qc.setQueryData(["branding"], data);
      qc.invalidateQueries({ queryKey: ["branding"] });
    },
  });
}
