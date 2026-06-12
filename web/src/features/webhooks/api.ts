import {
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import { api } from "@/lib/api";

export interface Webhook {
  id: number;
  name: string;
  url: string;
  events: string[];
  active: boolean;
}

export type WebhookEventType =
  | "ticket.created"
  | "ticket.updated"
  | "ticket.resolved"
  | "message.created"
  | "ticket.sla_warning";

export const ALL_WEBHOOK_EVENTS: WebhookEventType[] = [
  "ticket.created",
  "ticket.updated",
  "ticket.resolved",
  "message.created",
  "ticket.sla_warning",
];

export interface CreateWebhookInput {
  name: string;
  url: string;
  events: string[];
}

export interface CreateWebhookResult {
  webhook: Webhook;
  secret: string;
}

export interface WebhookDelivery {
  id: number;
  event_type: string;
  status: string;
  status_code: number | null;
  attempts: number;
  last_attempt_at: number | null;
  error: string | null;
  created_at: number;
}

const QK = ["webhooks"] as const;

function deliveryQK(webhookId: number) {
  return ["webhooks", webhookId, "deliveries"] as const;
}

export function useWebhooks() {
  return useQuery({
    queryKey: QK,
    queryFn: async () => {
      const res = await api.get("/admin/webhooks");
      const payload = res.data as { webhooks?: Webhook[] };
      return payload.webhooks ?? [];
    },
  });
}

export function useCreateWebhook() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: CreateWebhookInput): Promise<CreateWebhookResult> => {
      const res = await api.post("/admin/webhooks", input);
      return res.data as CreateWebhookResult;
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: QK });
    },
  });
}

export function useDeleteWebhook() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (id: number) => {
      await api.delete(`/admin/webhooks/${id}`);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: QK });
    },
  });
}

export function useWebhookDeliveries(webhookId: number | null) {
  return useQuery({
    queryKey: webhookId != null ? deliveryQK(webhookId) : ["webhooks", "deliveries-disabled"],
    enabled: webhookId != null,
    queryFn: async () => {
      const res = await api.get(`/admin/webhooks/${webhookId}/deliveries`);
      const payload = res.data as { deliveries?: WebhookDelivery[] };
      return payload.deliveries ?? [];
    },
  });
}

export function useTestWebhook() {
  return useMutation({
    mutationFn: async (id: number) => {
      const res = await api.post(`/admin/webhooks/${id}/test`);
      return res.data as { queued: boolean };
    },
  });
}
