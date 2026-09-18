import { useState, useEffect, useRef, useCallback, useLayoutEffect } from "react";
import type { CSSProperties } from "react";
import {
  Archive,
  ArchiveRestore,
  Bot,
  Bookmark,
  Check,
  CheckCheck,
  ChevronLeft,
  Copy,
  EllipsisVertical,
  FileText,
  Forward,
  MessageSquare,
  Paperclip,
  Pencil,
  Phone,
  PhoneCall,
  Pin,
  Reply,
  Trash2,
  Video,
  X,
} from "lucide-react";
import { apiRequest, deleteMessage, pinMessage, unpinMessage } from "../api/client";
import type { Chat, Message, Reaction } from "../types";
import type { ChatAppearance } from "../utils/chatTheme";
import { CHAT_COLORS, CHAT_WALLPAPERS } from "../utils/chatTheme";
import { useAuth } from "../context/AuthContext";
import { useSettings } from "../context/SettingsContext";
import ForwardPicker from "./ForwardPicker";
import ImageModal from "./ImageModal";
import MessageInput from "./MessageInput";
import type { ChatLiveHandlers } from "../hooks/useGlobalWebSocket";
import { playNotificationSound } from "../utils/sound";
import { useCall } from "../context/CallContext";

const emojis = [
  "👍", "❤️", "🔥", "😂", "😮", "😢", "🙏",
  "🎉", "👏", "💯", "🥰", "😍", "🤣", "😭",
  "😡", "🤔", "👀", "💪", "🤝", "✨", "⭐",
];

interface Props {
  chat: Chat;
  onBack?: () => void;
  onMessage?: () => void;
  onOpenUserProfile?: (userId: number) => void;
  onDeleteChat?: (chatId: number) => void;
  onArchiveChat?: (chatId: number) => void;
  onUnarchiveChat?: (chatId: number) => void;
  appearance?: ChatAppearance;
  onAppearanceChange?: (patch: ChatAppearance) => void;
  userStatuses?: Record<number, string>;
  registerLiveHandlers?: (chatId: number, handlers: ChatLiveHandlers) => void;
  unregisterLiveHandlers?: (chatId: number) => void;
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

export default function ChatArea({ chat, onBack, onMessage, onOpenUserProfile, onDeleteChat, onArchiveChat, onUnarchiveChat, appearance, onAppearanceChange, userStatuses, registerLiveHandlers, unregisterLiveHandlers }: Props) {
  const { user } = useAuth();
  const { settings } = useSettings();
  const { startCall } = useCall();
  const [messages, setMessages] = useState<Message[]>([]);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [editText, setEditText] = useState("");
  const [pinnedMessage, setPinnedMessage] = useState<Message | undefined>(chat.pinned_message);

  const [error, setError] = useState<string | null>(null);
  const [showMenu, setShowMenu] = useState(false);
  const [ctxMsgId, setCtxMsgId] = useState<number | null>(null);
  const [ctxPos, setCtxPos] = useState({ x: 0, y: 0 });
  const [forwardMsg, setForwardMsg] = useState<Message | null>(null);
  const [replyTo, setReplyTo] = useState<Message | null>(null);
  const [highlightId, setHighlightId] = useState<number | null>(null);
  const [selectedImage, setSelectedImage] = useState<{ url: string; name?: string } | null>(null);
  const editRef = useRef<HTMLInputElement>(null);
  const ctxRef = useRef<HTMLDivElement>(null);
  const menuRef = useRef<HTMLDivElement>(null);
  const messagesRef = useRef<HTMLDivElement>(null);
  const offsetRef = useRef(0);
  const scrollPosRef = useRef<{ scrollTop: number; scrollHeight: number } | null>(null);
  const loadMorePendingRef = useRef(false);
  const pendingJumpRef = useRef<number | null>(null);
  const jumpInProgressRef = useRef(false);
  const loadingMoreRef = useRef(false);
  const hasMoreRef = useRef(true);

  const [loadingMore, setLoadingMore] = useState(false);
  const [hasMore, setHasMore] = useState(true);

  const loadMessages = useCallback(async () => {
    try {
      setError(null);
      offsetRef.current = 0;
      setHasMore(true);
      hasMoreRef.current = true;
      const res = await apiRequest<{ data: Message[] }>(`/chats/${chat.id}/messages?limit=50&offset=0`);
      const data = res.data ?? [];
      setMessages(data);
      setHasMore(data.length >= 50);
      hasMoreRef.current = data.length >= 50;
      offsetRef.current = data.length;
    } catch (e) {
      setError(e instanceof Error ? e.message : "Ошибка загрузки");
    }
  }, [chat.id]);

  const markReadRef = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const markRead = useCallback(() => {
    if (document.hidden) return;
    if (markReadRef.current) return;
    markReadRef.current = setTimeout(() => {
      markReadRef.current = undefined;
    }, 2000);
    apiRequest(`/chats/${chat.id}/read`, "POST");
  }, [chat.id]);

  const onNewMessage = useCallback((msg: Message) => {
    setMessages((prev) => {
      if (prev.some((m) => m.id === msg.id)) return prev;
      return [...prev, msg];
    });
    if (msg.sender_id !== user?.id) {
      if (settings?.sound_enabled !== false) {
        playNotificationSound();
      }
      if (document.hidden && "Notification" in window && Notification.permission === "granted") {
        new Notification("MessangerMax", {
          body: `${msg.sender?.username ?? "Пользователь"}: ${msg.text.slice(0, 80)}`,
          icon: msg.sender?.avatar || undefined,
        });
      }
      markRead();
    }
  }, [user?.id, markRead, settings?.sound_enabled]);

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

  const onMessagesRead = useCallback((chatId: number, messageIds: number[]) => {
    if (chatId !== chat.id) return;
    setMessages((prev) =>
      prev.map((m) =>
        messageIds.includes(m.id) ? { ...m, is_read: true, read_at: new Date().toISOString() } : m
      )
    );
  }, [chat.id]);

  const onMessagePinned = useCallback((chatId: number, msg: Message) => {
    if (chatId !== chat.id) return;
    setPinnedMessage(msg);
  }, [chat.id]);

  const onMessageUnpinned = useCallback((chatId: number) => {
    if (chatId !== chat.id) return;
    setPinnedMessage(undefined);
  }, [chat.id]);

  useEffect(() => {
    if (!registerLiveHandlers || !unregisterLiveHandlers) return;
    registerLiveHandlers(chat.id, {
      onNewMessage,
      onMessageEdited,
      onReactionChange,
      onMessageDeleted,
      onMessagesRead: (messageIds) => onMessagesRead(chat.id, messageIds),
      onMessagePinned: (msg) => onMessagePinned(chat.id, msg),
      onMessageUnpinned: () => onMessageUnpinned(chat.id),
    });
    return () => unregisterLiveHandlers(chat.id);
  }, [chat.id, onNewMessage, onMessageEdited, onReactionChange, onMessageDeleted, onMessagesRead, onMessagePinned, onMessageUnpinned, registerLiveHandlers, unregisterLiveHandlers]);

  useEffect(() => {
    setPinnedMessage(chat.pinned_message);
    loadMessages();
  }, [loadMessages, chat.id]);

  useLayoutEffect(() => {
    const el = messagesRef.current;
    if (!el) return;
    if (scrollPosRef.current) {
      el.scrollTop = scrollPosRef.current.scrollTop + (el.scrollHeight - scrollPosRef.current.scrollHeight);
      scrollPosRef.current = null;
      return;
    }
    if (loadMorePendingRef.current) {
      loadMorePendingRef.current = false;
      return;
    }
    if (messages.length > 0) {
      el.scrollTop = el.scrollHeight;
    }
  }, [messages]);

  const pinIfAtBottom = useCallback(() => {
    const el = messagesRef.current;
    if (!el) return;
    if (el.scrollTop + el.clientHeight >= el.scrollHeight - 80) {
      el.scrollTop = el.scrollHeight;
    }
  }, []);

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

  async function handleSend(text: string, attachment?: { url: string; name: string; size: number; type: string }, replyToId?: number) {
    try {
      const body: Record<string, unknown> = { content: text };
      if (attachment) {
        body.attachment_url = attachment.url;
        body.attachment_name = attachment.name;
        body.attachment_size = attachment.size;
        body.attachment_type = attachment.type;
      }
      if (replyToId) {
        body.reply_to_message_id = replyToId;
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

  async function loadMore() {
    if (loadingMore || !hasMore) return;
    setLoadingMore(true);
    loadingMoreRef.current = true;
    loadMorePendingRef.current = true;

    const el = messagesRef.current;
    if (el) {
      scrollPosRef.current = { scrollTop: el.scrollTop, scrollHeight: el.scrollHeight };
    }

    try {
      const res = await apiRequest<{ data: Message[] }>(`/chats/${chat.id}/messages?limit=50&offset=${offsetRef.current}`);
      const newMsgs = res.data ?? [];
      offsetRef.current += newMsgs.length;
      setHasMore(newMsgs.length >= 50);
      hasMoreRef.current = newMsgs.length >= 50;
      setMessages((prev) => [...newMsgs, ...prev]);
    } catch { /* ignore */ }

    setLoadingMore(false);
    loadingMoreRef.current = false;
  }

  async function fetchMoreForJump(): Promise<boolean> {
    if (loadingMoreRef.current || !hasMoreRef.current) return false;
    loadingMoreRef.current = true;
    setLoadingMore(true);
    try {
      const res = await apiRequest<{ data: Message[] }>(`/chats/${chat.id}/messages?limit=50&offset=${offsetRef.current}`);
      const newMsgs = res.data ?? [];
      offsetRef.current += newMsgs.length;
      const more = newMsgs.length >= 50;
      setHasMore(more);
      hasMoreRef.current = more;
      setMessages((prev) => [...newMsgs, ...prev]);
      return newMsgs.length > 0;
    } catch {
      return false;
    } finally {
      loadingMoreRef.current = false;
      setLoadingMore(false);
    }
  }

  function revealMessage(targetId: number, el: Element) {
    el.scrollIntoView({ behavior: "smooth", block: "center" });
    setHighlightId(targetId);
    setTimeout(() => setHighlightId(null), 1600);
    setCtxMsgId(null);
  }

  async function jumpToMessage(targetId: number) {
    if (pendingJumpRef.current === targetId) return;
    const el = document.querySelector(`[data-msg-id="${targetId}"]`);
    if (el) {
      revealMessage(targetId, el);
      return;
    }
    pendingJumpRef.current = targetId;
    if (jumpInProgressRef.current) return;
    jumpInProgressRef.current = true;
    try {
      while (pendingJumpRef.current !== null && hasMoreRef.current) {
        const got = await fetchMoreForJump();
        await new Promise((r) => requestAnimationFrame(() => requestAnimationFrame(r)));
        const found = document.querySelector(`[data-msg-id="${pendingJumpRef.current}"]`);
        if (found) {
          revealMessage(pendingJumpRef.current, found);
          pendingJumpRef.current = null;
          break;
        }
        if (!got) break;
      }
    } finally {
      pendingJumpRef.current = null;
      jumpInProgressRef.current = false;
    }
  }

  function handleScroll() {
    if (loadingMore || !hasMore) return;
    const el = messagesRef.current;
    if (el && el.scrollTop < 80) {
      loadMore();
    }
  }

  function handleCopyText(text: string) {
    navigator.clipboard.writeText(text);
    setCtxMsgId(null);
  }

  function linkifyText(text: string) {
    const parts = text.split(/(https?:\/\/[^\s]+)/g);
    return parts.map((part, i) =>
      part.match(/^https?:\/\//)
        ? <a key={i} href={part} target="_blank" rel="noopener noreferrer">{part}</a>
        : part
    );
  }

  function handleOpenContextMenu(msgId: number, e: React.MouseEvent) {
    const rect = e.currentTarget.getBoundingClientRect();
    const menuWidth = 240;
    const menuHeight = 340;
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
  const partnerStatus = partner && userStatuses ? userStatuses[partner.id] : undefined;
  const ctxMessage = ctxMsgId !== null ? messages.find((m) => m.id === ctxMsgId) : null;
  const activeBg = appearance?.bg ?? "default";

  const accentStyle = {
    display: "contents",
    ...(appearance?.color ? { ["--chat-accent" as string]: appearance.color } : {}),
  } as CSSProperties;

  const selectColor = (color: string | undefined) => {
    onAppearanceChange?.({ color });
  };

  return (
    <div style={accentStyle}>
      <div className="chat-header">
        {onBack && <button className="back-btn" onClick={onBack}><ChevronLeft size={22} /></button>}
        <div className="chat-header-avatar" style={{ cursor: partner && !partner.is_bot ? "pointer" : "default", background: partner?.is_bot ? "#7c4dff" : chat.is_favorites ? "#ff9800" : undefined }} onClick={() => partner && !partner.is_bot && onOpenUserProfile?.(partner.id)}>
          {chat.is_favorites ? (
            <Bookmark size={22} />
          ) : partner?.avatar ? (
            <img src={partner.avatar} alt="" style={{ width: "100%", height: "100%", borderRadius: "50%", objectFit: "cover" }} />
          ) : partner?.is_bot ? <Bot size={22} /> : partner ? partner.username.charAt(0).toUpperCase() : "#"}
        </div>
        <div className="chat-header-info" style={{ cursor: partner && !partner.is_bot ? "pointer" : "default" }} onClick={() => partner && !partner.is_bot && onOpenUserProfile?.(partner.id)}>
          <div className="chat-header-name">{chat.is_favorites ? "Избранное" : partner?.username ?? chat.name}</div>
          <div className="chat-header-status">{chat.is_favorites ? "Сохранённые сообщения" : partner?.is_bot ? "AI-ассистент" : partnerStatus === "online" ? <span className="chat-header-status-online">в сети</span> : partnerStatus === "offline" ? "не в сети" : lastSeenLabel(partner?.last_login)}</div>
        </div>
        <div className="chat-header-actions">
          {partner && !partner.is_bot && (
            <>
              <button
                className="chat-call-btn"
                title="Аудиозвонок"
                onClick={() => startCall({ userId: partner.id, username: partner.username }, "audio", chat.id)}
              >
                <Phone size={20} />
              </button>
              <button
                className="chat-call-btn"
                title="Видеозвонок"
                onClick={() => startCall({ userId: partner.id, username: partner.username }, "video", chat.id)}
              >
                <Video size={21} />
              </button>
            </>
          )}
          <div className="chat-menu-container" ref={menuRef}>
            <button className="chat-menu-btn" onClick={() => setShowMenu(!showMenu)}>
              <EllipsisVertical size={20} />
            </button>
            {showMenu && (
              <div className="chat-menu-dropdown">
                <div className="chat-menu-section">
                  <div className="chat-menu-heading">Цвет чата</div>
                  <div className="color-row">
                    <div
                      className={`color-dot-baseline${!appearance?.color ? " selected" : ""}`}
                      onClick={() => selectColor(undefined)}
                      title="Стандартный"
                    >
                      <Check size={14} />
                    </div>
                    {CHAT_COLORS.map((c) => (
                      <div
                        key={c}
                        className={`color-dot${appearance?.color === c ? " selected" : ""}`}
                        style={{ background: c }}
                        onClick={() => selectColor(c)}
                        title={c}
                      />
                    ))}
                  </div>
                </div>
                <div className="chat-menu-section">
                  <div className="chat-menu-heading">Обои чата</div>
                  <div className="bg-row">
                    {CHAT_WALLPAPERS.map((w) => (
                      <div
                        key={w.id}
                        className={`bg-swatch bg-preview-${w.id}${activeBg === w.id ? " selected" : ""}`}
                        onClick={() => onAppearanceChange?.({ bg: w.id })}
                        title={w.name}
                      />
                    ))}
                  </div>
                </div>
                {!chat.is_favorites && (
                  <div className="chat-menu-section">
                    {chat.is_archived
                      ? onUnarchiveChat && (
                          <button className="plain" onClick={() => { onUnarchiveChat?.(chat.id); setShowMenu(false); }}>
                            <ArchiveRestore size={16} /> Разархивировать
                          </button>
                        )
                      : onArchiveChat && (
                          <button className="plain" onClick={() => { onArchiveChat?.(chat.id); setShowMenu(false); }}>
                            <Archive size={16} /> Архивировать
                          </button>
                        )}
                  </div>
                )}
                {!chat.is_favorites && (
                  <div className="chat-menu-section">
                    <button className="plain danger" onClick={() => { onDeleteChat?.(chat.id); setShowMenu(false); }}>
                      <Trash2 size={16} /> Удалить чат
                    </button>
                  </div>
                )}
              </div>
            )}
          </div>
        </div>
      </div>

      {error && (
        <div style={{ padding: "8px 16px", background: "#ffebee", color: "#c62828", fontSize: 13 }}>
          {error}
        </div>
      )}

      {pinnedMessage && (
        <div className="pinned-banner">
          <Pin size={15} className="pinned-icon" />
          <span className="pinned-text">{pinnedMessage.text.slice(0, 100)}</span>
          <button className="pinned-unpin" onClick={() => unpinMessage(chat.id)}><X size={15} /></button>
        </div>
      )}

      <div className={`messages chat-bg-${activeBg}`} ref={messagesRef} onScroll={handleScroll}>
        {loadingMore && (
          <div style={{ textAlign: "center", padding: "12px", color: "#888", fontSize: 13 }}>
            Загрузка...
          </div>
        )}
        {!loadingMore && !hasMore && messages.length > 0 && (
          <div style={{ textAlign: "center", padding: "12px", color: "#aaa", fontSize: 12 }}>
            Все сообщения загружены
          </div>
        )}
        {messages.length === 0 && !error && (
          <div className="empty-state">
            <div className="empty-icon"><MessageSquare size={30} /></div>
            <div>Нет сообщений</div>
          </div>
        )}
        {messages.map((m) => {
          if (m.system_type) {
            const isCall = m.system_type === "call";
            const isVideoCall = isCall && /видео|без звука/i.test(m.text);
            return (
              <div key={m.id} className={`msg-system${isCall ? " call-ended" : ""}`}>
                {isCall && (
                  <span className="sys-icon">
                    {isVideoCall ? <Video size={14} /> : <PhoneCall size={14} />}
                  </span>
                )}
                {m.text}
              </div>
            );
          }
          const isMine = m.sender_id === user?.id;
          return (
            <div
              key={m.id}
              data-msg-id={m.id}
              className={`msg ${isMine ? "mine" : ""}${m.id === highlightId ? " msg-highlight" : ""}`}
              onClick={(e) => handleOpenContextMenu(m.id, e)}
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
                  <button className="edit-save" onClick={() => handleEdit(m.id)}><Check size={16} /></button>
                  <button className="edit-cancel" onClick={() => setEditingId(null)}><X size={16} /></button>
                </div>
              ) : (
                <>
                  {m.reply_to && (
                    <div
                      className="reply-quote"
                      title="Перейти к сообщению"
                      onClick={(e) => {
                        e.stopPropagation();
                        jumpToMessage(m.reply_to!.id);
                      }}
                    >
                      <div className="reply-quote-name">{m.reply_to.username ?? "Пользователь"}</div>
                      <div className="reply-quote-text">
                        {m.reply_to.deleted
                          ? "Сообщение удалено"
                          : m.reply_to.system_type
                            ? m.reply_to.system_type
                            : m.reply_to.text || (m.reply_to.attachment_type === "image" ? "Фото" : m.reply_to.attachment_name || "Сообщение")}
                      </div>
                    </div>
                  )}
                  {m.is_forwarded && m.forwarded_from && (
                    <div className="forward-header">
                      <Forward size={13} />
                      <span>Переслано от {m.forwarded_from.username ?? "Пользователь"}</span>
                    </div>
                  )}
                  <div className="msg-text">
                    {linkifyText(m.text)}
                    {m.edited && <span className="edited-mark"> edited</span>}
                  </div>

                  {m.attachment_url && (
                    <div className="msg-attachment">
                      {m.attachment_type === "image" ? (
                        <img
                          src={m.attachment_url}
                          alt={m.attachment_name}
                          style={{ cursor: "pointer" }}
                          onLoad={pinIfAtBottom}
                          onClick={() => setSelectedImage({ url: m.attachment_url!, name: m.attachment_name })}
                        />
                      ) : m.attachment_name?.toLowerCase().endsWith(".pdf") ? (
                        <a className="file-attachment" href={m.attachment_url} target="_blank" rel="noreferrer">
                          <span className="file-icon"><FileText size={20} /></span>
                          <div>
                            <div className="file-name">{m.attachment_name}</div>
                            <div className="file-size">{formatSize(m.attachment_size)} · PDF</div>
                          </div>
                        </a>
                      ) : (
                        <a className="file-attachment" href={m.attachment_url} target="_blank" rel="noreferrer">
                          <span className="file-icon"><Paperclip size={20} /></span>
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
                        {m.is_read ? <CheckCheck size={15} /> : <Check size={15} />}
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
                      onClick={(e) => {
                        e.stopPropagation();
                        if (r.user_id === user?.id) removeReaction(m.id, r.reaction);
                      }}
                      title={r.username}
                    >
                      {r.reaction}
                    </span>
                  ))}
                </div>
              )}


            </div>
          );
        })}
      </div>

      {ctxMsgId !== null && ctxMessage && (
        <div
          ref={ctxRef}
          className="msg-context-menu"
          style={{ position: "fixed", top: ctxPos.y, left: ctxPos.x }}
        >
          <div className="ctx-reactions">
            {emojis.slice(0, 7).map((e) => (
              <span key={e} onClick={() => { handleReaction(ctxMsgId, e); setCtxMsgId(null); }}>
                {e}
              </span>
            ))}
          </div>
          {ctxMessage.sender_id === user?.id && (
            <button onClick={() => { setEditingId(ctxMsgId); setEditText(ctxMessage.text); setCtxMsgId(null); }}>
              <Pencil size={16} /> Редактировать
            </button>
          )}
          <button onClick={() => handleCopyText(ctxMessage.text)}>
            <Copy size={16} /> Копировать
          </button>
          <button onClick={() => { setReplyTo(ctxMessage); setCtxMsgId(null); }}>
            <Reply size={16} /> Ответить
          </button>
          <button onClick={() => { setForwardMsg(ctxMessage); setCtxMsgId(null); }}>
            <Forward size={16} /> Переслать
          </button>
          <button onClick={() => { pinMessage(chat.id, ctxMsgId); setCtxMsgId(null); }}>
            <Pin size={16} /> Закрепить
          </button>
          {ctxMessage.sender_id === user?.id && (
            <button className="danger" onClick={() => handleDeleteMessage(ctxMsgId)}>
              <Trash2 size={16} /> Удалить
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

      <MessageInput onSend={handleSend} replyTo={replyTo} onCancelReply={() => setReplyTo(null)} />

      {forwardMsg && (
        <ForwardPicker
          message={forwardMsg}
          excludeChatId={chat.id}
          onClose={() => setForwardMsg(null)}
          onForwarded={onMessage}
        />
      )}

      {selectedImage && (
        <ImageModal
          url={selectedImage.url}
          name={selectedImage.name}
          onClose={() => setSelectedImage(null)}
        />
      )}
    </div>
  );
}