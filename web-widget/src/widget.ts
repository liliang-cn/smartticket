// widget.ts — SmartTicket embeddable chat widget entry point
//
// Usage:
//   <script src="https://BACKEND/widget.js" data-key="..." data-accent="#2563eb" async></script>

import {
  startSession,
  postMessage,
  getHistory,
  getBranding,
  openWebSocket,
  type MessageResponse,
} from './api';
import { buildUI, appendMessage, scrollToBottom } from './ui';

const TOKEN_KEY = 'st_widget_token';
const DEFAULT_ACCENT = '#2563eb';

function getBase(): string {
  // Resolve the backend origin from the script element that loaded this file.
  const scriptEl =
    (document.currentScript as HTMLScriptElement | null) ||
    [...document.scripts].find((s) => s.src.includes('widget.js'));
  if (scriptEl) {
    try {
      return new URL(scriptEl.src).origin;
    } catch {
      // fall through
    }
  }
  return window.location.origin;
}

function getAccentFromScript(): string {
  const scriptEl =
    (document.currentScript as HTMLScriptElement | null) ||
    [...document.scripts].find((s) => s.src.includes('widget.js'));
  return scriptEl?.dataset.accent || DEFAULT_ACCENT;
}

function showError(banner: HTMLElement, msg: string): void {
  banner.textContent = msg;
  banner.classList.add('visible');
  setTimeout(() => banner.classList.remove('visible'), 5000);
}

function dedupeId(list: HTMLElement, msg: MessageResponse): boolean {
  return !!list.querySelector(`[data-msg-id="${msg.id}"]`);
}

function removeOptimistic(list: HTMLElement): void {
  list.querySelectorAll('.optimistic').forEach((el) => el.remove());
}

function resolveMyUserId(msgs: MessageResponse[]): number | null {
  // The most recently received message where we are the author.
  // We can't know definitively from history alone, so we tag by checking
  // if author_role is "customer". The first message in history was posted
  // by the visitor, so we use that to fingerprint "own" messages.
  const first = msgs[0];
  if (first) return first.user_id;
  return null;
}

async function init(): Promise<void> {
  const base = getBase();
  let accent = getAccentFromScript();
  let appName = 'Support';

  // Attempt to fetch branding to pick up deployment-specific accent/name.
  try {
    const branding = await getBranding(base);
    if (branding) {
      if (branding.primary_color) accent = branding.primary_color;
      if (branding.app_name) appName = branding.app_name;
    }
  } catch {
    // Gracefully degrade if CORS or network fails.
  }

  const ui = buildUI({ accent, appName });

  let panelOpen = false;
  let sessionStarted = false;
  let token: string | null = localStorage.getItem(TOKEN_KEY);
  let myUserId: number | null = null;
  let ws: WebSocket | null = null;

  function togglePanel(): void {
    panelOpen = !panelOpen;
    if (panelOpen) {
      ui.panel.classList.add('open');
      if (token && !sessionStarted) {
        void loadHistory();
      }
    } else {
      ui.panel.classList.remove('open');
    }
  }

  ui.launcher.addEventListener('click', togglePanel);
  ui.panel.querySelector('#close-btn')?.addEventListener('click', togglePanel);

  function switchToChat(): void {
    ui.prechatForm.style.display = 'none';
    ui.chatArea.style.display = 'flex';
    ui.textarea.focus();
  }

  async function loadHistory(): Promise<void> {
    if (!token) return;
    try {
      const msgs = await getHistory(base, token);
      myUserId = resolveMyUserId(msgs);
      for (const m of msgs) {
        if (!dedupeId(ui.messageList, m)) {
          const isOwn = myUserId !== null && m.user_id === myUserId;
          appendMessage(ui.messageList, m, isOwn);
        }
      }
      scrollToBottom(ui.messageList);
      sessionStarted = true;
      switchToChat();
      openWS();
    } catch {
      // token may be expired — clear it and show prechat
      localStorage.removeItem(TOKEN_KEY);
      token = null;
    }
  }

  function openWS(): void {
    if (!token || ws) return;
    try {
      ws = openWebSocket(base, token);
      ws.onmessage = (ev: MessageEvent<string>) => {
        try {
          const msg: MessageResponse = JSON.parse(ev.data);
          if (msg.is_internal) return;
          if (dedupeId(ui.messageList, msg)) return;
          // If this message came from us (just confirmed by server), remove optimistic placeholder.
          if (myUserId !== null && msg.user_id === myUserId) {
            removeOptimistic(ui.messageList);
          }
          const isOwn = myUserId !== null && msg.user_id === myUserId;
          appendMessage(ui.messageList, msg, isOwn);
          scrollToBottom(ui.messageList);
        } catch {
          // ignore malformed frames
        }
      };
      ws.onerror = () => {
        ws = null;
      };
      ws.onclose = () => {
        ws = null;
        // Reconnect after 3s if we still have a token.
        if (token) setTimeout(openWS, 3000);
      };
    } catch {
      // WebSocket not available or connection refused — silent degradation.
    }
  }

  // Pre-chat form submission
  ui.prechatSubmit.addEventListener('click', async () => {
    const msg = ui.firstMsgInput.value.trim();
    if (!msg) {
      ui.firstMsgInput.focus();
      return;
    }
    ui.prechatSubmit.disabled = true;
    ui.prechatSubmit.textContent = 'Starting…';
    try {
      const session = await startSession(base, {
        email: ui.emailInput.value.trim() || undefined,
        name: ui.nameInput.value.trim() || undefined,
        message: msg,
      });
      token = session.token;
      myUserId = null; // will be resolved from history
      localStorage.setItem(TOKEN_KEY, token);
      sessionStarted = true;

      // Load history to get back the created message with proper id.
      const msgs = await getHistory(base, token);
      myUserId = resolveMyUserId(msgs);
      switchToChat();
      for (const m of msgs) {
        appendMessage(ui.messageList, m, myUserId !== null && m.user_id === myUserId);
      }
      scrollToBottom(ui.messageList);
      openWS();
    } catch (err) {
      ui.prechatSubmit.disabled = false;
      ui.prechatSubmit.textContent = 'Start chat';
      const msg = err instanceof Error ? err.message : 'Could not start session';
      showError(ui.errorBanner, msg);
      ui.prechatForm.appendChild(ui.errorBanner); // move banner into prechat
    }
  });

  // Allow Enter (without shift) to submit prechat form
  ui.firstMsgInput.addEventListener('keydown', (e: KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      ui.prechatSubmit.click();
    }
  });

  // Composer send
  async function sendMessage(): Promise<void> {
    const text = ui.textarea.value.trim();
    if (!text || !token) return;
    ui.textarea.value = '';
    ui.sendBtn.disabled = true;

    // Optimistic render
    const optimisticMsg: MessageResponse = {
      id: -Date.now(), // temp negative id — will never clash with real ids
      ticket_id: 0,
      user_id: myUserId ?? 0,
      author_name: '',
      author_role: 'customer',
      content: text,
      content_type: 'text',
      is_internal: false,
      is_from_ai: false,
      created_at: new Date().toISOString(),
    };
    appendMessage(ui.messageList, optimisticMsg, true, true);
    scrollToBottom(ui.messageList);

    try {
      const confirmed = await postMessage(base, token, text);
      // Remove the optimistic placeholder; the WS echo (or manual append below)
      // will add the confirmed message. If WS is down, add it manually.
      removeOptimistic(ui.messageList);
      if (!dedupeId(ui.messageList, confirmed)) {
        appendMessage(ui.messageList, confirmed, true);
        scrollToBottom(ui.messageList);
      }
    } catch (err) {
      // Remove optimistic and show error.
      removeOptimistic(ui.messageList);
      const msg = err instanceof Error ? err.message : "Couldn't send";
      showError(ui.errorBanner, `${msg} — tap send to retry`);
      // Restore text so user can retry.
      ui.textarea.value = text;
    } finally {
      ui.sendBtn.disabled = false;
      ui.textarea.focus();
    }
  }

  ui.sendBtn.addEventListener('click', () => void sendMessage());
  ui.textarea.addEventListener('keydown', (e: KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      void sendMessage();
    }
  });

  // Auto-resize textarea
  ui.textarea.addEventListener('input', () => {
    ui.textarea.style.height = 'auto';
    ui.textarea.style.height = Math.min(ui.textarea.scrollHeight, 100) + 'px';
  });

  // If we already have a token, show the chat area immediately on first open
  // (history loads when panel opens).
  if (token) {
    // Pre-load in background so first open is fast.
    void loadHistory();
  }
}

// Boot when DOM is ready.
if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', () => void init());
} else {
  void init();
}
