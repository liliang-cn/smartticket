import {
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import { api, unwrap } from "@/lib/api";
import type {
  Attachment,
  Ticket,
  TicketMessage,
  TicketStats,
} from "@/lib/types";

export interface TicketFilters {
  page: number;
  page_size: number;
  status?: string;
  priority?: string;
  search?: string;
}

export interface TicketPage {
  items: Ticket[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

export function useTickets(filters: TicketFilters) {
  return useQuery({
    queryKey: ["tickets", filters],
    queryFn: async (): Promise<TicketPage> => {
      const res = await api.get("/tickets/", {
        params: {
          page: filters.page,
          page_size: filters.page_size,
          status: filters.status || undefined,
          priority: filters.priority || undefined,
          search: filters.search || undefined,
        },
      });
      // The list endpoint returns { success, data: Ticket[], meta: {...} }.
      const body = res.data as {
        data?: Ticket[];
        meta?: {
          total?: number;
          page?: number;
          page_size?: number;
          total_pages?: number;
        };
      };
      const meta = body.meta ?? {};
      return {
        items: body.data ?? [],
        total: meta.total ?? 0,
        page: meta.page ?? filters.page,
        page_size: meta.page_size ?? filters.page_size,
        total_pages: meta.total_pages ?? 1,
      };
    },
    placeholderData: (prev) => prev,
  });
}

export function useTicket(id: number | undefined) {
  return useQuery({
    queryKey: ["ticket", id],
    enabled: id != null,
    queryFn: async () => {
      const res = await api.get(`/tickets/${id}`);
      return unwrap<Ticket>(res.data);
    },
  });
}

export interface TicketSLA {
  ticket_id: number;
  priority: string;
  severity: string;
  source: "rule" | "default";
  policy_name: string;
  response_minutes: number;
  resolution_minutes: number;
  business_only: boolean;
  due_date?: string | null;
  sla_status: string;
}

/** The SLA policy governing a ticket (matched rule + template, targets, status). */
export function useTicketSLA(id: number | undefined) {
  return useQuery({
    queryKey: ["ticket-sla", id],
    enabled: id != null,
    queryFn: async (): Promise<TicketSLA> => {
      const res = await api.get(`/tickets/${id}/sla`);
      return unwrap<TicketSLA>(res.data);
    },
  });
}

export interface TicketEvent {
  id: number;
  action: string;
  summary: string;
  actor_name?: string;
  actor_role?: string;
  created_at: string;
}

/** A ticket's activity log (creation, status/priority changes, assignment, replies). */
export function useTicketEvents(id: number | undefined) {
  return useQuery({
    queryKey: ["ticket-events", id],
    enabled: id != null,
    queryFn: async (): Promise<TicketEvent[]> => {
      const res = await api.get(`/tickets/${id}/events`);
      return unwrap<TicketEvent[]>(res.data) ?? [];
    },
  });
}

export function useTicketStats() {
  return useQuery({
    queryKey: ["ticket-stats"],
    queryFn: async () => {
      const res = await api.get("/tickets/stats");
      return unwrap<TicketStats>(res.data);
    },
  });
}

export function useTicketMessages(id: number | undefined) {
  return useQuery({
    queryKey: ["ticket-messages", id],
    enabled: id != null,
    queryFn: async () => {
      const res = await api.get(`/tickets/${id}/messages`);
      return unwrap<TicketMessage[]>(res.data) ?? [];
    },
  });
}

export interface CreateTicketInput {
  title: string;
  description: string;
  priority: string;
  severity: string;
  requester_name: string;
  requester_email: string;
  category?: string;
}

export function useCreateTicket() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: CreateTicketInput) => {
      const res = await api.post("/tickets/", input);
      return unwrap<Ticket>(res.data);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["tickets"] });
      qc.invalidateQueries({ queryKey: ["ticket-stats"] });
    },
  });
}

export function useUpdateTicket(id: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (patch: Partial<Ticket>) => {
      const res = await api.put(`/tickets/${id}`, patch);
      return unwrap<Ticket>(res.data);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["ticket", id] });
      qc.invalidateQueries({ queryKey: ["tickets"] });
      qc.invalidateQueries({ queryKey: ["ticket-stats"] });
      qc.invalidateQueries({ queryKey: ["ticket-events", id] });
      qc.invalidateQueries({ queryKey: ["ticket-sla", id] });
    },
  });
}

export function useAssignTicket(id: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (assignedTo: number) => {
      const res = await api.post(`/tickets/${id}/assign`, { assigned_to: assignedTo });
      return unwrap<Ticket>(res.data);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["ticket", id] });
      qc.invalidateQueries({ queryKey: ["tickets"] });
      qc.invalidateQueries({ queryKey: ["ticket-events", id] });
    },
  });
}

export function useTicketAttachments(id: number | undefined) {
  return useQuery({
    queryKey: ["ticket-attachments", id],
    enabled: (id ?? 0) > 0,
    queryFn: async () => {
      const res = await api.get(`/tickets/${id}/attachments`);
      return unwrap<Attachment[]>(res.data) ?? [];
    },
  });
}

export function useUploadAttachment(id: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (file: File) => {
      const formData = new FormData();
      formData.append("file", file);
      const res = await api.post(`/tickets/${id}/attachments`, formData, {
        headers: { "Content-Type": "multipart/form-data" },
      });
      return unwrap<Attachment>(res.data);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["ticket-attachments", id] });
      qc.invalidateQueries({ queryKey: ["ticket", id] });
    },
  });
}

/** Download an attachment via the axios instance so the Bearer token is attached. */
export async function downloadAttachment(att: Attachment) {
  const res = await api.get(`/attachments/${att.id}/download`, {
    responseType: "blob",
  });
  const url = URL.createObjectURL(res.data as Blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = att.original_name;
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(url);
}

export function useAddMessage(id: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: { content: string; is_internal: boolean }) => {
      const res = await api.post(`/tickets/${id}/messages`, input);
      return unwrap<TicketMessage>(res.data);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["ticket-messages", id] });
      qc.invalidateQueries({ queryKey: ["ticket-events", id] });
    },
  });
}

// Ticket merge

export function useMergeTicket(id: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (into: number) => {
      await api.post(`/tickets/${id}/merge`, { into });
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["ticket", id] });
      qc.invalidateQueries({ queryKey: ["tickets"] });
      qc.invalidateQueries({ queryKey: ["ticket-events", id] });
    },
  });
}

// Ticket links

export interface TicketLink {
  id: number;
  source_id: number;
  target_id: number;
  type: string;
  other_ticket: {
    id: number;
    title: string;
    status: string;
  };
}

export function useTicketLinks(id: number | undefined) {
  return useQuery({
    queryKey: ["ticket-links", id],
    enabled: (id ?? 0) > 0,
    queryFn: async (): Promise<TicketLink[]> => {
      const res = await api.get(`/tickets/${id}/links`);
      return unwrap<TicketLink[]>(res.data) ?? [];
    },
  });
}

export function useCreateTicketLink(id: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: { target_id: number; type: string }) => {
      const res = await api.post(`/tickets/${id}/links`, input);
      return unwrap<TicketLink>(res.data);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["ticket-links", id] });
      qc.invalidateQueries({ queryKey: ["ticket-events", id] });
    },
  });
}

export function useDeleteTicketLink(ticketId: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (linkId: number) => {
      await api.delete(`/tickets/${ticketId}/links/${linkId}`);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["ticket-links", ticketId] });
      qc.invalidateQueries({ queryKey: ["ticket-events", ticketId] });
    },
  });
}
