// api.ts — thin HTTP client for the SmartTicket widget endpoints.
//
// Response envelope: { success: boolean, data: T, error?: string }
// All requests go to the same origin as the widget script.

export interface MessageResponse {
  id: number;
  ticket_id: number;
  user_id: number;
  author_name: string;
  author_role: string;
  content: string;
  content_type: string;
  is_internal: boolean;
  is_from_ai: boolean;
  created_at: string; // ISO-8601 string
}

export interface SessionResponse {
  token: string;
  ticket_id: number;
}

export interface BrandingResponse {
  app_name: string;
  primary_color: string;
  logo_url?: string;
}

type ApiEnvelope<T> = { success: true; data: T } | { success: false; error: string };

async function apiGet<T>(base: string, path: string, token?: string): Promise<T> {
  const url = new URL(path, base);
  if (token) url.searchParams.set('token', token);
  const res = await fetch(url.toString(), {
    headers: { Accept: 'application/json' },
  });
  const json: ApiEnvelope<T> = await res.json();
  if (!json.success) throw new Error((json as { error: string }).error || 'request failed');
  return (json as { success: true; data: T }).data;
}

async function apiPost<T>(base: string, path: string, body: unknown, token?: string): Promise<T> {
  const url = new URL(path, base);
  const headers: Record<string, string> = { 'Content-Type': 'application/json', Accept: 'application/json' };
  if (token) headers['Authorization'] = `Bearer ${token}`;
  const res = await fetch(url.toString(), {
    method: 'POST',
    headers,
    body: JSON.stringify(body),
  });
  const json: ApiEnvelope<T> = await res.json();
  if (!json.success) throw new Error((json as { error: string }).error || 'request failed');
  return (json as { success: true; data: T }).data;
}

export function startSession(
  base: string,
  opts: { email?: string; name?: string; message?: string },
): Promise<SessionResponse> {
  return apiPost<SessionResponse>(base, '/widget/session', opts);
}

export function postMessage(base: string, token: string, message: string): Promise<MessageResponse> {
  return apiPost<MessageResponse>(base, '/widget/messages', { message }, token);
}

export function getHistory(base: string, token: string): Promise<MessageResponse[]> {
  return apiGet<MessageResponse[]>(base, '/widget/messages', token);
}

export async function getBranding(base: string): Promise<BrandingResponse | null> {
  try {
    return await apiGet<BrandingResponse>(base, '/api/v1/settings/branding');
  } catch {
    return null;
  }
}

export function openWebSocket(base: string, token: string): WebSocket {
  const wsBase = base.replace(/^http/, 'ws');
  return new WebSocket(`${wsBase}/widget/ws?conversation_token=${encodeURIComponent(token)}`);
}
