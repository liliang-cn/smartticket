// API types mirroring the SmartTicket backend.

export type Role = "admin" | "engineer" | "customer";

export interface UserInfo {
  id: number;
  email: string;
  username: string;
  first_name: string;
  last_name: string;
  role: Role;
  is_active: boolean;
  last_login_at?: string | null;
  created_at?: string | null;
  customer_id?: number | null;
}

export interface TokenPair {
  access_token: string;
  refresh_token: string;
  expires_at: string;
  token_type: string;
}

export interface LoginResponse {
  success: boolean;
  user: UserInfo;
  tokens: TokenPair;
  expires_in: number;
  refresh_in: number;
}

export type TicketStatus =
  | "open"
  | "in_progress"
  | "resolved"
  | "closed"
  | "cancelled";
export type TicketPriority = "low" | "medium" | "high" | "critical";
export type TicketSeverity = "trivial" | "minor" | "major" | "critical";

export interface Ticket {
  id: number;
  ticket_number: string;
  title: string;
  description: string;
  status: TicketStatus;
  priority: TicketPriority;
  severity: TicketSeverity;
  category: string;
  type: string;
  product_id?: number | null;
  service_id?: number | null;
  customer_id?: number | null;
  assigned_to?: number | null;
  assigned_user?: UserInfo | null;
  requester_name: string;
  requester_email: string;
  tags?: string[];
  custom_fields?: Record<string, unknown>;
  is_deleted: boolean;
  created_at: string;
  updated_at: string;
  resolved_at?: string | null;
  due_date?: string | null;
  sla_status?: string;
}

export interface TicketListResponse {
  data: Ticket[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

export interface TicketMessage {
  id: number;
  ticket_id: number;
  user_id: number;
  content: string;
  content_type: string;
  is_internal: boolean;
  is_from_ai: boolean;
  created_at: string;
}

export interface Customer {
  id: number;
  name: string;
  code: string;
  domain: string;
  is_active: boolean;
  description: string;
  created_at: string;
  updated_at: string;
}

export interface CustomerUser {
  id: number;
  email: string;
  username: string;
  first_name: string;
  last_name: string;
  role: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface TicketStats {
  total_tickets: number;
  open_tickets: number;
  in_progress_tickets: number;
  resolved_tickets: number;
  closed_tickets: number;
  overdue_tickets: number;
  priority_breakdown?: Record<string, number>;
  status_breakdown?: Record<string, number>;
}
