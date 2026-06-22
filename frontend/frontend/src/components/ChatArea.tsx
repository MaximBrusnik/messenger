import { useState, useEffect, useRef, useCallback } from "react";
import { apiRequest, deleteMessage } from "../api/client";
import type { Chat, Message, Reaction } from "../types";
import { useAuth } from "../context/AuthContext";
import MessageInput from "./MessageInput";
import { useWebSocket } from "../hooks/useWebSocket";

const emojis = [
  "👍", "❤️", "🔥", "😂", "😮", "😢", "🙏",
  "🎉", "👏", "💯", "🥰", "😍", "🤣", "😭",
  "😡", "🤔", "👀", "💪", "🤝", "✨", "⭐",
];

interface Props {
  chat: Chat;
  onBack?: () => void;
  onMessage?: () => void;
  onUserStatus?: (userId: number, status: string) => void;
  onOpenUserProfile?: (userId: number) => void;
  onDeleteChat?: (chatId: number) => void;
}

function formatTime(iso?: string): string {
  if (!iso) return "";
  const d = new Date(iso);
  const now = new Date();
  const isToday = d.toDateString() === now.toDateString();
  if (isToday) return d.toLocaleTimeString("ru-RU", { hour: "2-digit", minute: "2-digit" });
  const yesterday = new Date(now);
  yesterday.setDate(now.getDate() - 1);
  if (d.toDateString() === yesterday.toDateString()) return "вчера " + d.toLocaleTimeString("ru-RU", { hour: "2-digit", minute: "2-digit" });
  return d.toLocaleDateString("ru-RU", { day: "numeric", month: "short", year: "numeric" }) + " " + d.toLocaleTimeString("ru-RU", { hour: "2-digit", minute: "2-digit" });
}

function lastSeenLabel(iso?: string): string {
  if (!iso) return "";
  const d = new Date(iso);
  if (isNaN(d.getTime())) return "";
  const now = new Date();
  const diffMs = now.getTime() - d.getTime();
  const diffMin = Math.floor(diffMs / 60000);
  if (diffMin < 1) return "в сети";
  if (diffMin < 60) return `был(а) ${diffMin} мин. назад`;
  const diffH = Math.floor(diffMin / 60);
  if (diffH < 24) return `был(а) ${diffH} ч назад`;
  const diffD = Math.floor(diffH / 24);
  if (diffD === 1) return "был(а) вчера";
  if (diffD < 7) return `был(а) ${diffD} дн. назад`;
  return "был(а) " + d.toLocaleDateString("ru-RU", { day: "numeric", month: "short" });
}

function formatSize(bytes?: number): string {
  if (!bytes) return "";
  if (bytes < 1024) return bytes + " B";
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + " KB";
  return (bytes / (1024 * 1024)).toFixed(1) + " MB";
}

export default function ChatArea({ chat, onBack, onMessage, onUserStatus, onOpenUserProfile, onDeleteChat }: Props) {
  const { user } = useAuth();
  const [messages, setMessages] = useState<Message[]>([]);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [editText, setEditText] = useState("");
  const [reactionMsgId, setReactionMsgId] = useState<number | null>(null);
  const [pickerTop, setPickerTop] = useState(0);
  const [error, setError] = useState<string | null>(null);
  const [showMenu, setShowMenu] = useState(false);
  const [ctxMsgId, setCtxMsgId] = useState<number | null>(null);
  const [ctxPos, setCtxPos] = useState({ x: 0, y: 0 });
  const bottomRef = useRef<HTMLDivElement>(null);
  const editRef = useRef<HTMLInputElement>(null);
  const ctxRef = useRef<HTMLDivElement>(null);
  const menuRef = useRef<HTMLDivElement>(null);

  const loadMessages = useCallback(async () => {
    try {
      setError(null);
      const res = await apiRequest<{ data: Message[] }>(`/chats/${chat.id}/messages`);
      setMessages(res.data ?? []);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Ошибка загрузки");
    }
  }, [chat.id]);

  const onNewMessage = useCallback((msg: Message) => {
    setMessages((prev) => {
      if (prev.some((m) => m.id === msg.id)) return prev;
      return [...prev, msg];
    });
  }, []);

  const onMessageEdited = useCallback((msg: Message) => {
    setMessages((prev) =>
      prev.map((m) => (m.id === msg.id ? { ...m, text: msg.text, edited: true, edited_at: msg.edited_at } : m))
    );
  }, []);

  const onReactionChange = useCallback(() => {
    loadMessages();
  }, [loadMessages]);

  const onMessageDeleted = useCallback((msgId: number) => {
    setMessages((prev) => prev.filter((m) => m.id !== msgId));
  }, []);

  const onChatDeleted = useCallback((deletedChatId: number) => {
    if (deletedChatId === chat.id) {
      onDeleteChat?.(deletedChatId);
    }
  }, [chat.id, onDeleteChat]);

  const onMessagesRead = useCallback((chatId: number, messageIds: number[]) => {
    if (chatId !== chat.id) return;
    setMessages((prev) =>
      prev.map((m) =>
        messageIds.includes(m.id) ? { ...m, is_read: true, read_at: new Date().toISOString() } : m
      )
    );
  }, [chat.id]);

  useWebSocket(chat.id, onNewMessage, onMessageEdited, onReactionChange, onMessage, onUserStatus, onMessageDeleted, onChatDeleted, onMessagesRead);

  useEffect(() => {
    loadMessages();
  }, [loadMessages]);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  useEffect(() => {
    if (editingId && editRef.current) {
      editRef.current.focus();
    }
  }, [editingId]);

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (ctxRef.current && !ctxRef.current.contains(e.target as Node)) {
        setCtxMsgId(null);
      }
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setShowMenu(false);
      }
    }
    if (ctxMsgId !== null || showMenu) {
      document.addEventListener("mousedown", handleClickOutside);
    }
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, [ctxMsgId, showMenu]);

  async function handleSend(text: string, attachment?: { url: string; name: string; size: number; type: string }) {
    try {
      const body: Record<string, unknown> = { content: text };
      if (attachment) {
        body.attachment_url = attachment.url;
        body.attachment_name = attachment.name;
        body.attachment_size = attachment.size;
        body.attachment_type = attachment.type;
      }
      const res = await apiRequest<{ data: Message }>(`/chats/${chat.id}/messages`, "POST", body);
      setMessages((prev) => prev.some((m) => m.id === res.data.id) ? prev : [...prev, res.data]);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Ошибка отправки");
    }
  }

  async function handleEdit(msgId: number) {
    try {
      if (!editText.trim()) return;
      const res = await apiRequest<{ data: Message }>(
        `/chats/${chat.id}/messages/${msgId}`,
        "PUT",
        { content: editText }
      );
      setMessages((prev) =>
        prev.map((m) => (m.id === msgId ? { ...m, text: res.data.text, edited: true, edited_at: res.data.edited_at } : m))
      );
      setEditingId(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Ошибка редактирования");
    }
  }

  async function handleReaction(msgId: number, emoji: string) {
    try {
      await apiRequest(`/chats/${chat.id}/messages/${msgId}/reactions`, "POST", { reaction: emoji });
      loadMessages();
    } catch { /* ignore */ }
  }

  async function removeReaction(msgId: number, emoji: string) {
    try {
      await apiRequest(
        `/chats/${chat.id}/messages/${msgId}/reactions?reaction=${encodeURIComponent(emoji)}`,
        "DELETE"
      );
      loadMessages();
    } catch { /* ignore */ }
  }

  async function handleDeleteMessage(msgId: number) {
    try {
      await deleteMessage(chat.id, msgId);
      setMessages((prev) => prev.filter((m) => m.id !== msgId));
      setCtxMsgId(null);
    } catch { /* ignore */ }
  }

  function handleCopyText(text: string) {
    navigator.clipboard.writeText(text);
    setCtxMsgId(null);
  }

  function handleOpenContextMenu(msgId: number, e: React.MouseEvent) {
    const rect = e.currentTarget.getBoundingClientRect();
    const menuWidth = 220;
    const menuHeight = 280;
    let left = rect.left;
    let top = rect.bottom + 4;

    if (left + menuWidth > window.innerWidth) {
      left = window.innerWidth - menuWidth - 8;
    }
    if (top + menuHeight > window.innerHeight) {
      top = rect.top - menuHeight - 4;
    }
    if (left < 8) left = 8;
    if (top < 8) top = 8;

    setCtxPos({ x: left, y: top });
    setCtxMsgId(msgId);
  }

  const partner = chat.participants?.find((p) => p.id !== user?.id);
  const ctxMessage = ctxMsgId !== null ? messages.find((m) => m.id === ctxMsgId) : null;

  return (
    <>
      <div className="chat-header">
        {onBack && <button className="back-btn" onClick={onBack}>←</button>}
        <div className="chat-header-avatar" style={{ cursor: partner ? "pointer" : "default" }} onClick={() => partner && onOpenUserProfile?.(partner.id)}>
          {partner?.avatar ? (
            <img src={partner.avatar} alt="" style={{ width: "100%", height: "100%", borderRadius: "50%", objectFit: "cover" }} />
          ) : partner ? partner.username.charAt(0).toUpperCase() : "#"}
        </div>
        <div className="chat-header-info" style={{ cursor: partner ? "pointer" : "default" }} onClick={() => partner && onOpenUserProfile?.(partner.id)}>
          <div className="chat-header-name">{partner?.username ?? chat.name}</div>
          <div className="chat-header-status">{lastSeenLabel(partner?.last_login)}</div>
        </div>
        <div className="chat-menu-container" ref={menuRef}>
          <button className="chat-menu-btn" onClick={() => setShowMenu(!showMenu)}>⋮</button>
          {showMenu && (
            <div className="chat-menu-dropdown">
              <button onClick={() => { onDeleteChat?.(chat.id); setShowMenu(false); }}>Удалить чат</button>
            </div>
          )}
        </div>
      </div>

      {error && (
        <div style={{ padding: "8px 16px", background: "#ffebee", color: "#c62828", fontSize: 13 }}>
          {error}
        </div>
      )}

      <div className="messages">
        {messages.length === 0 && !error && (
          <div className="empty-state">
            <div className="empty-icon">💬</div>
            <div>Нет сообщений</div>
          </div>
        )}
        {messages.map((m) => {
          const isMine = m.sender_id === user?.id;
          return (
            <div
              key={m.id}
              className={`msg ${isMine ? "mine" : ""}`}
              onClick={(e) => handleOpenContextMenu(m.id, e)}
              onMouseEnter={(e) => {
                if (ctxMsgId === null) {
                  const rect = e.currentTarget.getBoundingClientRect();
                  setPickerTop(rect.top - 8);
                  setReactionMsgId(m.id);
                }
              }}
              onMouseLeave={() => {
                setReactionMsgId(null);
              }}
            >
              {!isMine && <div className="msg-sender">{m.sender?.username}</div>}

              {editingId === m.id ? (
                <div className="edit-inline">
                  <input
                    ref={editRef}
                    value={editText}
                    onChange={(e) => setEditText(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === "Enter") handleEdit(m.id);
                      if (e.key === "Escape") setEditingId(null);
                    }}
                  />
                  <button className="edit-save" onClick={() => handleEdit(m.id)}>✓</button>
                  <button className="edit-cancel" onClick={() => setEditingId(null)}>✕</button>
                </div>
              ) : (
                <>
                  <div className="msg-text">
                    {m.text}
                    {m.edited && <span className="edited-mark"> edited</span>}
                  </div>

                  {m.attachment_url && (
                    <div className="msg-attachment">
                      {m.attachment_type === "image" ? (
                        <img src={m.attachment_url} alt={m.attachment_name} />
                      ) : m.attachment_name?.toLowerCase().endsWith(".pdf") ? (
                        <a className="file-attachment" href={m.attachment_url} target="_blank" rel="noreferrer">
                          <span className="file-icon">📄</span>
                          <div>
                            <div className="file-name">{m.attachment_name}</div>
                            <div className="file-size">{formatSize(m.attachment_size)} · PDF</div>
                          </div>
                        </a>
                      ) : (
                        <a className="file-attachment" href={m.attachment_url} target="_blank" rel="noreferrer">
                          <span className="file-icon">📎</span>
                          <div>
                            <div className="file-name">{m.attachment_name}</div>
                            <div className="file-size">{formatSize(m.attachment_size)}</div>
                          </div>
                        </a>
                      )}
                    </div>
                  )}

                  <div className="msg-footer">
                    <span className="msg-time">{formatTime(m.created_at)}</span>
                    {isMine && (
                      <span className={`msg-check ${m.is_read ? "read" : ""}`}>
                        {m.is_read ? "✓✓" : "✓"}
                      </span>
                    )}
                  </div>
                </>
              )}

              {m.reactions && m.reactions.length > 0 && (
                <div className="msg-reactions">
                  {m.reactions.map((r: Reaction) => (
                    <span
                      key={`${r.reaction}-${r.user_id}`}
                      className={`reaction${r.user_id === user?.id ? " mine" : ""}`}
                      onClick={() => {
                        if (r.user_id === user?.id) removeReaction(m.id, r.reaction);
                      }}
                      title={r.username}
                    >
                      {r.reaction}
                    </span>
                  ))}
                </div>
              )}

              {reactionMsgId === m.id && editingId !== m.id && (
                <div className="reaction-picker" style={{ position: "fixed", top: pickerTop, left: "50%", transform: "translateX(-50%)" }} onMouseLeave={() => setReactionMsgId(null)}>
                  {emojis.map((e) => (
                    <span key={e} onClick={() => { handleReaction(m.id, e); setReactionMsgId(null); }}>
                      {e}
                    </span>
                  ))}
                </div>
              )}
            </div>
          );
        })}
        <div ref={bottomRef} />
      </div>

      {ctxMsgId !== null && ctxMessage && (
        <div
          ref={ctxRef}
          className="msg-context-menu"
          style={{ position: "fixed", top: ctxPos.y, left: ctxPos.x }}
        >
          {ctxMessage.sender_id === user?.id && (
            <button onClick={() => { setEditingId(ctxMsgId); setEditText(ctxMessage.text); setCtxMsgId(null); }}>
              ✏️ Редактировать
            </button>
          )}
          <button onClick={() => handleCopyText(ctxMessage.text)}>
            📋 Копировать
          </button>
          <button onClick={() => { alert("Пересылка будет позже"); setCtxMsgId(null); }}>
            📤 Переслать
          </button>
          <button onClick={() => { alert("Закрепление будет позже"); setCtxMsgId(null); }}>
            📌 Закрепить
          </button>
          {ctxMessage.sender_id === user?.id && (
            <button className="danger" onClick={() => handleDeleteMessage(ctxMsgId)}>
              🗑️ Удалить
            </button>
          )}
          <div className="ctx-divider" />
          <div className="ctx-read-info">
            {ctxMessage.is_read && ctxMessage.read_at
              ? `Прочитано: ${formatTime(ctxMessage.read_at)}`
              : "Не прочитано"}
          </div>
        </div>
      )}

      <MessageInput onSend={handleSend} />
    </>
  );
}