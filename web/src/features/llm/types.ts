export type LLMTaskType = "chat" | "embedding";

/**
 * A provider as returned by the backend. Note `task_types` is a JSON-ENCODED
 * STRING (e.g. `"[\"chat\"]"`) on read; parse it before use. The raw API key
 * is never returned — `api_key_masked` is "********" when a key is stored.
 */
export interface LLMProvider {
  id: number;
  name: string;
  provider_type: string;
  api_endpoint: string;
  model: string;
  task_types: string;
  dimensions?: number;
  max_tokens?: number;
  temperature?: number;
  is_default: boolean;
  is_enabled: boolean;
  api_key_masked?: string;
}

/**
 * Body for POST/PUT. Unlike the response, `task_types` is a real array here.
 * Leave `api_key` blank on edit to keep the existing key.
 */
export interface ProviderInput {
  name: string;
  provider_type: string;
  api_endpoint: string;
  api_key?: string;
  model: string;
  task_types: LLMTaskType[];
  dimensions?: number;
  max_tokens?: number;
  temperature?: number;
  is_default: boolean;
  is_enabled: boolean;
}

export interface TestResult {
  chat_ok: boolean;
  embedding_ok: boolean;
  cortex_ok: boolean;
  latency_ms: number;
  error?: string;
}
