// ui.ts — Shadow DOM widget UI builder

import type { MessageResponse } from './api';

export interface UIOptions {
  accent: string;
  appName: string;
}

export interface UIElements {
  shadow: ShadowRoot;
  panel: HTMLElement;
  launcher: HTMLElement;
  messageList: HTMLElement;
  prechatForm: HTMLElement;
  chatArea: HTMLElement;
  textarea: HTMLTextAreaElement;
  sendBtn: HTMLButtonElement;
  emailInput: HTMLInputElement;
  nameInput: HTMLInputElement;
  firstMsgInput: HTMLTextAreaElement;
  prechatSubmit: HTMLButtonElement;
  errorBanner: HTMLElement;
}

const STYLES = (accent: string) => `
  *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }

  :host { all: initial; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; }

  #launcher {
    position: fixed;
    bottom: 24px;
    right: 24px;
    width: 56px;
    height: 56px;
    border-radius: 50%;
    background: ${accent};
    border: none;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    box-shadow: 0 4px 16px rgba(0,0,0,0.22);
    z-index: 2147483646;
    transition: transform 0.2s ease, box-shadow 0.2s ease;
  }
  #launcher:hover { transform: scale(1.07); box-shadow: 0 6px 20px rgba(0,0,0,0.28); }
  #launcher svg { width: 26px; height: 26px; fill: #fff; }

  #panel {
    position: fixed;
    bottom: 92px;
    right: 24px;
    width: 360px;
    max-width: calc(100vw - 32px);
    height: 520px;
    max-height: calc(100vh - 110px);
    border-radius: 16px;
    background: #fff;
    box-shadow: 0 8px 40px rgba(0,0,0,0.18);
    display: flex;
    flex-direction: column;
    overflow: hidden;
    z-index: 2147483645;
    opacity: 0;
    transform: translateY(12px) scale(0.97);
    pointer-events: none;
    transition: opacity 0.22s ease, transform 0.22s ease;
  }
  #panel.open {
    opacity: 1;
    transform: translateY(0) scale(1);
    pointer-events: auto;
  }

  #header {
    background: ${accent};
    color: #fff;
    padding: 14px 16px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    flex-shrink: 0;
  }
  #header-title { font-size: 15px; font-weight: 600; }
  #close-btn {
    background: transparent;
    border: none;
    color: rgba(255,255,255,0.85);
    cursor: pointer;
    font-size: 22px;
    line-height: 1;
    padding: 0 4px;
    transition: color 0.15s;
  }
  #close-btn:hover { color: #fff; }

  #prechat {
    padding: 20px;
    display: flex;
    flex-direction: column;
    gap: 12px;
    overflow-y: auto;
    flex: 1;
  }
  #prechat p {
    font-size: 14px;
    color: #555;
    line-height: 1.5;
  }
  .field-label {
    font-size: 12px;
    color: #777;
    margin-bottom: 4px;
    font-weight: 500;
  }
  .field-wrap { display: flex; flex-direction: column; }
  .field-input {
    border: 1px solid #ddd;
    border-radius: 8px;
    padding: 9px 12px;
    font-size: 14px;
    outline: none;
    transition: border-color 0.15s;
    font-family: inherit;
    resize: none;
  }
  .field-input:focus { border-color: ${accent}; }
  #prechat-submit {
    background: ${accent};
    color: #fff;
    border: none;
    border-radius: 10px;
    padding: 11px;
    font-size: 14px;
    font-weight: 600;
    cursor: pointer;
    transition: opacity 0.15s;
    margin-top: 4px;
  }
  #prechat-submit:hover { opacity: 0.9; }
  #prechat-submit:disabled { opacity: 0.6; cursor: default; }

  #chat-area {
    display: flex;
    flex-direction: column;
    flex: 1;
    overflow: hidden;
  }

  #message-list {
    flex: 1;
    overflow-y: auto;
    padding: 14px 14px 0 14px;
    display: flex;
    flex-direction: column;
    gap: 10px;
    scroll-behavior: smooth;
  }

  .msg-row { display: flex; align-items: flex-end; gap: 8px; }
  .msg-row.from-me { flex-direction: row-reverse; }

  .msg-avatar {
    width: 28px;
    height: 28px;
    border-radius: 50%;
    background: #e5e7eb;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    font-size: 11px;
    color: #6b7280;
    font-weight: 600;
  }
  .msg-row.from-me .msg-avatar { background: ${accent}; color: #fff; }

  .msg-bubble-wrap { display: flex; flex-direction: column; max-width: 75%; gap: 3px; }
  .msg-row.from-me .msg-bubble-wrap { align-items: flex-end; }

  .msg-meta {
    font-size: 10px;
    color: #9ca3af;
    padding: 0 6px;
  }
  .msg-meta .ai-badge {
    background: #f3f4f6;
    color: #6b7280;
    border-radius: 4px;
    padding: 1px 5px;
    font-size: 9px;
    font-weight: 600;
    text-transform: uppercase;
    margin-left: 4px;
    letter-spacing: 0.03em;
  }

  .msg-bubble {
    padding: 9px 13px;
    border-radius: 16px;
    font-size: 14px;
    line-height: 1.5;
    word-break: break-word;
    white-space: pre-wrap;
  }
  .msg-row.from-agent .msg-bubble {
    background: #f3f4f6;
    color: #111;
    border-bottom-left-radius: 4px;
  }
  .msg-row.from-me .msg-bubble {
    background: ${accent};
    color: #fff;
    border-bottom-right-radius: 4px;
  }
  .msg-row.optimistic .msg-bubble { opacity: 0.7; }

  #composer {
    display: flex;
    gap: 8px;
    padding: 12px 14px;
    border-top: 1px solid #f0f0f0;
    flex-shrink: 0;
    align-items: flex-end;
  }
  #msg-input {
    flex: 1;
    border: 1px solid #ddd;
    border-radius: 10px;
    padding: 9px 12px;
    font-size: 14px;
    outline: none;
    resize: none;
    font-family: inherit;
    line-height: 1.4;
    max-height: 100px;
    overflow-y: auto;
    transition: border-color 0.15s;
  }
  #msg-input:focus { border-color: ${accent}; }
  #send-btn {
    background: ${accent};
    border: none;
    border-radius: 10px;
    width: 38px;
    height: 38px;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    transition: opacity 0.15s;
  }
  #send-btn:hover { opacity: 0.88; }
  #send-btn:disabled { opacity: 0.5; cursor: default; }
  #send-btn svg { width: 18px; height: 18px; fill: #fff; }

  #error-banner {
    display: none;
    background: #fef2f2;
    color: #dc2626;
    font-size: 12px;
    padding: 7px 14px;
    text-align: center;
    border-top: 1px solid #fecaca;
  }
  #error-banner.visible { display: block; }
`;

const CHAT_ICON = `<svg viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg"><path d="M20 2H4C2.9 2 2 2.9 2 4v18l4-4h14c1.1 0 2-.9 2-2V4c0-1.1-.9-2-2-2zm0 12H6l-2 2V4h16v10z"/></svg>`;
const CLOSE_ICON = `<svg viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg"><path d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"/></svg>`;
const SEND_ICON = `<svg viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg"><path d="M2.01 21L23 12 2.01 3 2 10l15 2-15 2z"/></svg>`;

export function buildUI(opts: UIOptions): UIElements {
  const host = document.createElement('div');
  host.id = 'st-widget-host';
  document.body.appendChild(host);
  const shadow = host.attachShadow({ mode: 'open' });

  const style = document.createElement('style');
  style.textContent = STYLES(opts.accent);
  shadow.appendChild(style);

  // Launcher
  const launcher = document.createElement('button');
  launcher.id = 'launcher';
  launcher.setAttribute('aria-label', 'Open support chat');
  launcher.innerHTML = CHAT_ICON;
  shadow.appendChild(launcher);

  // Panel
  const panel = document.createElement('div');
  panel.id = 'panel';
  panel.setAttribute('role', 'dialog');
  panel.setAttribute('aria-label', opts.appName + ' chat');
  shadow.appendChild(panel);

  // Header
  const header = document.createElement('div');
  header.id = 'header';
  const headerTitle = document.createElement('span');
  headerTitle.id = 'header-title';
  headerTitle.textContent = opts.appName;
  const closeBtn = document.createElement('button');
  closeBtn.id = 'close-btn';
  closeBtn.setAttribute('aria-label', 'Close chat');
  closeBtn.innerHTML = CLOSE_ICON;
  header.appendChild(headerTitle);
  header.appendChild(closeBtn);
  panel.appendChild(header);

  // Pre-chat form
  const prechatForm = document.createElement('div');
  prechatForm.id = 'prechat';
  const prechatHint = document.createElement('p');
  prechatHint.textContent = 'Start a conversation. Email and name are optional.';
  const emailWrap = document.createElement('div');
  emailWrap.className = 'field-wrap';
  const emailLabel = document.createElement('span');
  emailLabel.className = 'field-label';
  emailLabel.textContent = 'Email (optional)';
  const emailInput = document.createElement('input') as HTMLInputElement;
  emailInput.className = 'field-input';
  emailInput.type = 'email';
  emailInput.placeholder = 'you@example.com';
  emailWrap.appendChild(emailLabel);
  emailWrap.appendChild(emailInput);

  const nameWrap = document.createElement('div');
  nameWrap.className = 'field-wrap';
  const nameLabel = document.createElement('span');
  nameLabel.className = 'field-label';
  nameLabel.textContent = 'Name (optional)';
  const nameInput = document.createElement('input') as HTMLInputElement;
  nameInput.className = 'field-input';
  nameInput.type = 'text';
  nameInput.placeholder = 'Your name';
  nameWrap.appendChild(nameLabel);
  nameWrap.appendChild(nameInput);

  const msgWrap = document.createElement('div');
  msgWrap.className = 'field-wrap';
  const msgLabel = document.createElement('span');
  msgLabel.className = 'field-label';
  msgLabel.textContent = 'Message';
  const firstMsgInput = document.createElement('textarea') as HTMLTextAreaElement;
  firstMsgInput.className = 'field-input';
  firstMsgInput.rows = 3;
  firstMsgInput.placeholder = 'How can we help?';
  msgWrap.appendChild(msgLabel);
  msgWrap.appendChild(firstMsgInput);

  const prechatSubmit = document.createElement('button') as HTMLButtonElement;
  prechatSubmit.id = 'prechat-submit';
  prechatSubmit.textContent = 'Start chat';

  prechatForm.appendChild(prechatHint);
  prechatForm.appendChild(emailWrap);
  prechatForm.appendChild(nameWrap);
  prechatForm.appendChild(msgWrap);
  prechatForm.appendChild(prechatSubmit);
  panel.appendChild(prechatForm);

  // Chat area
  const chatArea = document.createElement('div');
  chatArea.id = 'chat-area';
  chatArea.style.display = 'none';

  const messageList = document.createElement('div');
  messageList.id = 'message-list';
  messageList.setAttribute('aria-live', 'polite');
  chatArea.appendChild(messageList);

  const errorBanner = document.createElement('div');
  errorBanner.id = 'error-banner';
  chatArea.appendChild(errorBanner);

  const composer = document.createElement('div');
  composer.id = 'composer';
  const textarea = document.createElement('textarea') as HTMLTextAreaElement;
  textarea.id = 'msg-input';
  textarea.rows = 1;
  textarea.placeholder = 'Type a message…';
  textarea.setAttribute('aria-label', 'Message');
  const sendBtn = document.createElement('button') as HTMLButtonElement;
  sendBtn.id = 'send-btn';
  sendBtn.setAttribute('aria-label', 'Send');
  sendBtn.innerHTML = SEND_ICON;
  composer.appendChild(textarea);
  composer.appendChild(sendBtn);
  chatArea.appendChild(composer);

  panel.appendChild(chatArea);

  return {
    shadow,
    panel,
    launcher,
    messageList,
    prechatForm,
    chatArea,
    textarea,
    sendBtn,
    emailInput,
    nameInput,
    firstMsgInput,
    prechatSubmit,
    errorBanner,
  };
}

export function appendMessage(
  list: HTMLElement,
  msg: MessageResponse,
  isOwn: boolean,
  optimistic = false,
): HTMLElement {
  const row = document.createElement('div');
  row.className = 'msg-row ' + (isOwn ? 'from-me' : 'from-agent') + (optimistic ? ' optimistic' : '');
  row.dataset.msgId = String(msg.id);

  const avatar = document.createElement('div');
  avatar.className = 'msg-avatar';
  avatar.textContent = isOwn ? 'You' : (msg.author_name ? msg.author_name.charAt(0).toUpperCase() : 'A');

  const bubbleWrap = document.createElement('div');
  bubbleWrap.className = 'msg-bubble-wrap';

  const meta = document.createElement('div');
  meta.className = 'msg-meta';
  const time = new Date(msg.created_at);
  meta.textContent = !isNaN(time.getTime())
    ? time.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
    : '';
  if (msg.is_from_ai) {
    const badge = document.createElement('span');
    badge.className = 'ai-badge';
    badge.textContent = 'AI';
    meta.appendChild(badge);
  } else if (!isOwn && msg.author_name) {
    meta.textContent = msg.author_name + '  ' + meta.textContent;
  }

  const bubble = document.createElement('div');
  bubble.className = 'msg-bubble';
  bubble.textContent = msg.content;

  bubbleWrap.appendChild(meta);
  bubbleWrap.appendChild(bubble);
  row.appendChild(avatar);
  row.appendChild(bubbleWrap);
  list.appendChild(row);
  return row;
}

export function scrollToBottom(list: HTMLElement): void {
  list.scrollTop = list.scrollHeight;
}
