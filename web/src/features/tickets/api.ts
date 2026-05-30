import {
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import { api, unwrap } from "@/lib/api";
import type { Ticket, TicketMessage, TicketStats } from "@/lib/types";

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
    },
  });
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
    },
  });
}
