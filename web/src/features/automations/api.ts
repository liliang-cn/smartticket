import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api, unwrap } from "@/lib/api";

export interface AutomationRule {
  id: number;
  name: string;
  description: string;
  enabled: boolean;
  event: string;
  match: "all" | "any";
  /** JSON string — array of Condition objects */
  conditions: string;
  /** JSON string — array of Action objects */
  actions: string;
  position: number;
  created_at: string;
  updated_at: string;
}

export interface Condition {
  field: string;
  op: string;
  value: string;
}

export interface ActionParam {
  [key: string]: string;
}

export interface RuleAction {
  type: string;
  params?: ActionParam;
}

export interface CreateRuleInput {
  name: string;
  description: string;
  enabled: boolean;
  event: string;
  match: "all" | "any";
  conditions: string;
  actions: string;
  position?: number;
}

export type UpdateRuleInput = Partial<CreateRuleInput>;

export function useAutomationRules() {
  return useQuery({
    queryKey: ["automations"],
    queryFn: async () => {
      const res = await api.get("/automations");
      return unwrap<AutomationRule[]>(res.data) ?? [];
    },
  });
}

export function useCreateAutomationRule() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: CreateRuleInput) => {
      const res = await api.post("/automations", input);
      return unwrap<AutomationRule>(res.data);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["automations"] });
    },
  });
}

export function useUpdateAutomationRule(id: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: UpdateRuleInput) => {
      const res = await api.put(`/automations/${id}`, input);
      return unwrap<AutomationRule>(res.data);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["automations"] });
    },
  });
}

export function useDeleteAutomationRule() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (id: number) => {
      await api.delete(`/automations/${id}`);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["automations"] });
    },
  });
}

export function useReorderAutomationRules() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (ids: number[]) => {
      await api.post("/automations/reorder", { ids });
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["automations"] });
    },
  });
}
