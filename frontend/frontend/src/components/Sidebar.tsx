import {
  useEffect,
  useRef,
  useState,
  type PointerEvent,
  type ReactNode,
  type TouchEvent,
} from "react";
import UserSearch from "./UserSearch";
import MusicTab from "./MusicTab";
import {
  Archive,
  ArchiveRestore,
  Bookmark,
  Bot,
  MessageSquare,
  Music,
  Trash2,
} from "lucide-react";
import type { Chat, User } from "../types";

type TabId = "chats" | "archived" | "music";
const TAB_ORDER: TabId[] = ["chats", "archived", "music"];

const TAB_SWIPE_THRESHOLD = 80;
const TAB_FLIGHT_MS = 190;
const REVEAL_WIDTH = 76;
const REVEAL_TRIGGER = 44;
const ARCHIVE_COMMIT = 120;

interface Props {
  user: User;
  chats: Chat[];
  archivedChats: Chat[];
  activeChat: Chat | null;
  activeTab: TabId;
  activeTrackId: number | null;
  onSelectChat: (chat: Chat) => void;
  onOpenUserProfile: (userId: number) => void;
  onOpenProfile: () => void;
  onDeleteChat: (chatId: number) => void;
  onArchiveChat: (chatId: number) => void;
  onUnarchiveChat: (chatId: number) => void;
  onStartAIChat?: () => void;
  onStartFavoritesChat?: () => void;
  onTabChange: (tab: TabId) => void;
  onPlayTrack: (id: number) => void;
}

function formatTime(iso?: string): string {
  if (!iso) return "";
  const d = new Date(iso);
  if (isNaN(d.getTime())) return "";
  const now = new Date();
  const diffMs = now.getTime() - d.getTime();
  const diffMin = Math.floor(diffMs / 60000);
  if (diffMin < 1) return "только что";
  if (diffMin < 60) return `${diffMin} мин. назад`;
  const diffH = Math.floor(diffMin / 60);
  if (diffH < 24) return `${diffH} ч назад`;
  const diffD = Math.floor(diffH / 24);
  if (diffD === 1) return "вчера";
  if (diffD < 7) return `${diffD} дн. назад`;
  return d.toLocaleDateString("ru-RU", { day: "numeric", month: "short" });
}

function chatTime(iso?: string): string {
  if (!iso) return "";
  const d = new Date(iso);
  const now = new Date();
  const isToday = d.toDateString() === now.toDateString();
  if (isToday) return d.toLocaleTimeString("ru-RU", { hour: "2-digit", minute: "2-digit" });
  const yesterday = new Date(now);
  yesterday.setDate(now.getDate() - 1);
  if (d.toDateString() === yesterday.toDateString()) return "вчера";
  return d.toLocaleDateString("ru-RU", { day: "numeric", month: "short" });
}

const initial = (name: string) => name.charAt(0).toUpperCase();

function renderAvatar(c: Chat) {
  if (c.is_favorites) {
    return (
      <div className="chat-item-avatar favorites-avatar"><Bookmark size={20} /></div>
    );
  }
  if (c.participants?.[0]?.avatar) {
    return (
      <img src={c.participants[0].avatar} alt="" style={{ width: "100%", height: "100%", borderRadius: "50%", objectFit: "cover" }} />
    );
  }
  return c.participants?.[0] ? initial(c.participants[0].username) : "#";
}

interface SwipeableChatItemProps {
  active: boolean;
  onClick: () => void;
  actionIcon: ReactNode;
  actionColor: string;
  actionLabel: string;
  onAction: () => void;
  children: ReactNode;
}

function SwipeableChatItem({
  active,
  onClick,
  actionIcon,
  actionColor,
  actionLabel,
  onAction,
  children,
}: SwipeableChatItemProps) {
  const [offset, setOffset] = useState(0);
  const [revealed, setRevealed] = useState(false);
  const [dragging, setDragging] = useState(false);
  const [elastic, setElastic] = useState(false);
  const [flyout, setFlyout] = useState(false);
  const startRef = useRef<{ x: number; y: number } | null>(null);
  const dragRef = useRef(false);
  const didDragRef = useRef(false);
  const rawRef = useRef(0);
  const commitTimer = useRef<number | null>(null);
  const elasticTimer = useRef<number | null>(null);
  const elasticClearTimer = useRef<number | null>(null);
  const mountedRef = useRef(true);

  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
      if (commitTimer.current) window.clearTimeout(commitTimer.current);
      if (elasticTimer.current) window.clearTimeout(elasticTimer.current);
      if (elasticClearTimer.current) window.clearTimeout(elasticClearTimer.current);
    };
  }, []);

  function touchStart(e: TouchEvent) {
    if (revealed) return;
    startRef.current = { x: e.touches[0].clientX, y: e.touches[0].clientY };
    dragRef.current = false;
    rawRef.current = 0;
    setElastic(false);
    setFlyout(false);
  }

  function touchMove(e: TouchEvent) {
    if (!startRef.current) {
      if (revealed) e.stopPropagation();
      return;
    }
    const dx = e.touches[0].clientX - startRef.current.x;
    const dy = e.touches[0].clientY - startRef.current.y;
    if (!dragRef.current) {
      if (Math.abs(dx) < 8 || Math.abs(dx) <= Math.abs(dy)) return;
      dragRef.current = true;
      setDragging(true);
      e.stopPropagation();
    } else {
      e.stopPropagation();
    }
    rawRef.current = dx;
    let o = dx;
    if (o > 0) o *= 0.35;
    if (o < -REVEAL_WIDTH) o = -(REVEAL_WIDTH + (o + REVEAL_WIDTH) * 0.35);
    setOffset(o);
  }

  function touchEnd() {
    if (!startRef.current) return;
    startRef.current = null;
    if (!dragRef.current) return;
    dragRef.current = false;
    setDragging(false);
    const raw = rawRef.current;
    if (raw <= -ARCHIVE_COMMIT) {
      didDragRef.current = true;
      setRevealed(false);
      setFlyout(true);
      setOffset(-560);
      commitTimer.current = window.setTimeout(() => {
        if (mountedRef.current) onAction();
      }, 230);
    } else if (raw <= -REVEAL_TRIGGER) {
      didDragRef.current = true;
      setElastic(true);
      setRevealed(true);
      setOffset(-REVEAL_WIDTH);
      elasticClearTimer.current = window.setTimeout(() => setElastic(false), 320);
    } else {
      didDragRef.current = true;
      setElastic(true);
      setRevealed(false);
      setOffset(14);
      elasticTimer.current = window.setTimeout(() => {
        setOffset(0);
        elasticClearTimer.current = window.setTimeout(() => setElastic(false), 300);
      }, 30);
    }
  }

  function handleClick() {
    if (didDragRef.current) {
      didDragRef.current = false;
      return;
    }
    if (revealed) {
      if (elasticTimer.current) window.clearTimeout(elasticTimer.current);
      if (elasticClearTimer.current) window.clearTimeout(elasticClearTimer.current);
      setElastic(true);
      setRevealed(false);
      setOffset(14);
      elasticTimer.current = window.setTimeout(() => {
        setOffset(0);
        elasticClearTimer.current = window.setTimeout(() => setElastic(false), 300);
      }, 30);
      return;
    }
    onClick();
  }

  return (
    <div className="chat-item-swipe">
      <div className="chat-item-swipe-bg" style={{ background: actionColor }}>
        <button
          type="button"
          className="chat-item-swipe-action"
          title={actionLabel}
          onClick={(e) => { e.stopPropagation(); onAction(); }}
        >
          {actionIcon}
        </button>
      </div>
      <div
        className={`chat-item${active ? " active" : ""}${dragging ? " swiping" : ""}${revealed ? " revealed" : ""}${elastic ? " elastic" : ""}${flyout ? " flyout" : ""}`}
        style={{ transform: `translateX(${offset}px)` }}
        onClick={handleClick}
        onTouchStart={touchStart}
        onTouchMove={touchMove}
        onTouchEnd={touchEnd}
        onTouchCancel={touchEnd}
      >
        {children}
      </div>
    </div>
  );
}

export default function Sidebar({
  user, chats, archivedChats, activeChat, activeTab, activeTrackId,
  onSelectChat, onOpenUserProfile, onOpenProfile, onDeleteChat,
  onArchiveChat, onUnarchiveChat, onStartAIChat, onStartFavoritesChat,
  onTabChange, onPlayTrack,
}: Props) {
  const statusLabel = user.status === "online" ? "В сети" : `Был(а) ${formatTime(user.last_login)}`;

  const activeIndex = TAB_ORDER.indexOf(activeTab);

  const [dragX, setDragX] = useState(0);
  const [dragging, setDragging] = useState(false);
  const [pageIndex, setPageIndex] = useState(activeIndex);
  const [targetTab, setTargetTab] = useState<TabId | null>(null);
  const pointerRef = useRef<{ pointerId: number; x: number; y: number } | null>(null);
  const draggingRef = useRef(false);
  const pendingCommitRef = useRef<TabId | null>(null);
  const commitTimer = useRef<number | null>(null);

  useEffect(() => () => {
    if (commitTimer.current) window.clearTimeout(commitTimer.current);
  }, []);

  useEffect(() => {
    if (activeIndex === pageIndex) return;
    setPageIndex(activeIndex);
    setDragX(0);
  }, [activeIndex, pageIndex]);

  const handleTabClick = (tab: TabId) => {
    const targetIndex = TAB_ORDER.indexOf(tab);
    if (commitTimer.current) {
      window.clearTimeout(commitTimer.current);
      commitTimer.current = null;
    }
    pendingCommitRef.current = null;
    draggingRef.current = false;
    pointerRef.current = null;
    setDragging(false);
    setTargetTab(null);
    setPageIndex(targetIndex);
    setDragX(0);
    onTabChange(tab);
  };

  const tabPointerDown = (e: PointerEvent<HTMLDivElement>) => {
    if (pendingCommitRef.current) return;
    const target = e.target as HTMLElement;
    if (target.closest(".chat-item-swipe") || target.closest(".chat-item-actions")) return;
    pointerRef.current = { pointerId: e.pointerId, x: e.clientX, y: e.clientY };
    draggingRef.current = false;
    setDragging(false);
    setDragX(0);
    setTargetTab(null);
  };

  const tabPointerMove = (e: PointerEvent<HTMLDivElement>) => {
    const start = pointerRef.current;
    if (!start || e.pointerId !== start.pointerId) return;
    if (pendingCommitRef.current) return;
    const dx = e.clientX - start.x;
    const dy = e.clientY - start.y;
    if (!draggingRef.current) {
      if (Math.abs(dx) < 8 || Math.abs(dx) <= Math.abs(dy)) return;
      draggingRef.current = true;
      setDragging(true);
      try { e.currentTarget.setPointerCapture?.(e.pointerId); } catch { /* ignore */ }
      if (/touch/gi.test(e.pointerType)) e.preventDefault();
    }
    const boundary =
      (pageIndex === 0 && dx > 0) ||
      (pageIndex === TAB_ORDER.length - 1 && dx < 0);
    setDragX(boundary ? dx * 0.35 : dx);
    setTargetTab(Math.abs(dx) >= 20 && !boundary
      ? (TAB_ORDER[pageIndex + (dx < 0 ? 1 : -1)] ?? null)
      : null);
  };

  const tabPointerEnd = (e: PointerEvent<HTMLDivElement>) => {
    const start = pointerRef.current;
    if (!start || e.pointerId !== start.pointerId) return;
    finishTabDrag(e);
  };

  const tabPointerCancel = (e: PointerEvent<HTMLDivElement>) => {
    const start = pointerRef.current;
    if (!start || e.pointerId !== start.pointerId) return;
    draggingRef.current = false;
    pointerRef.current = null;
    setDragging(false);
    setDragX(0);
    setTargetTab(null);
  };

  const finishTabDrag = (e: PointerEvent<HTMLDivElement>) => {
    const start = pointerRef.current;
    if (!start) return;
    pointerRef.current = null;
    const wasDragging = draggingRef.current;
    draggingRef.current = false;
    setDragging(false);
    try { e.currentTarget.releasePointerCapture?.(e.pointerId); } catch { /* ignore */ }
    if (!wasDragging) { setTargetTab(null); return; }
    const dx = e.clientX - start.x;
    if (Math.abs(dx) >= TAB_SWIPE_THRESHOLD) {
      const dir = dx < 0 ? 1 : -1;
      const nextIndex = pageIndex + dir;
      if (nextIndex >= 0 && nextIndex < TAB_ORDER.length) {
        const next = TAB_ORDER[nextIndex];
        pendingCommitRef.current = next;
        setTargetTab(next);
        setPageIndex(nextIndex);
        setDragX(0);
        commitTimer.current = window.setTimeout(() => {
          if (!commitTimer.current) return;
          pendingCommitRef.current = null;
          setTargetTab(null);
          onTabChange(next);
          commitTimer.current = null;
        }, TAB_FLIGHT_MS);
        return;
      }
    }
    setDragX(0);
    setTargetTab(null);
  };

  const renderChatsTab = (
    <>
      <UserSearch onOpenProfile={onOpenUserProfile} />
      {onStartFavoritesChat && (
        <div
          className="chat-item"
          style={{ borderBottom: "1px solid var(--border)", cursor: "pointer" }}
          onClick={onStartFavoritesChat}
        >
          <div className="chat-item-avatar favorites-avatar"><Bookmark size={20} /></div>
          <div className="chat-item-content">
            <div className="chat-item-name">Избранное</div>
            <div className="chat-item-preview">Сохранённые сообщения</div>
          </div>
        </div>
      )}
      {onStartAIChat && (
        <div
          className="chat-item"
          style={{ borderBottom: "1px solid var(--border)", cursor: "pointer" }}
          onClick={onStartAIChat}
        >
          <div className="chat-item-avatar" style={{ background: "#7c4dff" }}><Bot size={20} /></div>
          <div className="chat-item-content">
            <div className="chat-item-name">Ассистент</div>
            <div className="chat-item-preview">AI-помощник</div>
          </div>
        </div>
      )}
      <div className="chat-list">
        {chats.map((c) => (
          <SwipeableChatItem
            key={c.id}
            active={activeChat?.id === c.id}
            onClick={() => onSelectChat(c)}
            actionIcon={<Archive size={18} />}
            actionColor="var(--accent)"
            actionLabel="Архивировать"
            onAction={() => onArchiveChat(c.id)}
          >
            <div className="chat-item-avatar">{renderAvatar(c)}</div>
            <div className="chat-item-content">
              <div className="chat-item-name">{c.name}</div>
              <div className="chat-item-preview">
                {c.last_message ? c.last_message.text || (c.last_message.attachment_type === "image" ? "Фото" : "Файл") : "Нет сообщений"}
              </div>
            </div>
            <div className="chat-item-right">
              <div className="chat-item-time">{chatTime(c.last_message?.created_at)}</div>
              {c.unread ? <div className="chat-item-unread">{c.unread}</div> : null}
            </div>
            <div className="chat-item-actions">
              {onArchiveChat && (
                <button
                  className="chat-item-action"
                  onClick={(e) => { e.stopPropagation(); onArchiveChat(c.id); }}
                  title="Архивировать"
                >
                  <Archive size={14} />
                </button>
              )}
              <button
                className="chat-item-action danger"
                onClick={(e) => { e.stopPropagation(); onDeleteChat(c.id); }}
                title="Удалить чат"
              >
                <Trash2 size={14} />
              </button>
            </div>
          </SwipeableChatItem>
        ))}
      </div>
    </>
  );

  const renderArchivedTab = (
    <>
      <div className="archive-hint">
        <Archive size={16} />
        <span>Архивные чаты скрыты из основного списка</span>
      </div>
      <div className="chat-list">
        {archivedChats.length === 0 && (
          <div className="archive-empty">Нет архивированных чатов</div>
        )}
        {archivedChats.map((c) => (
          <SwipeableChatItem
            key={c.id}
            active={activeChat?.id === c.id}
            onClick={() => onSelectChat(c)}
            actionIcon={<ArchiveRestore size={18} />}
            actionColor="var(--success)"
            actionLabel="Разархивировать"
            onAction={() => onUnarchiveChat(c.id)}
          >
            <div className="chat-item-avatar">{renderAvatar(c)}</div>
            <div className="chat-item-content">
              <div className="chat-item-name">{c.name}</div>
              <div className="chat-item-preview">
                {c.last_message ? c.last_message.text || (c.last_message.attachment_type === "image" ? "Фото" : "Файл") : "Нет сообщений"}
              </div>
            </div>
            <div className="chat-item-right">
              <div className="chat-item-time">{chatTime(c.last_message?.created_at)}</div>
              {c.unread ? <div className="chat-item-unread">{c.unread}</div> : null}
            </div>
            <button
              className="chat-item-action"
              onClick={(e) => { e.stopPropagation(); onUnarchiveChat(c.id); }}
              title="Разархивировать"
            >
              <ArchiveRestore size={14} />
            </button>
          </SwipeableChatItem>
        ))}
      </div>
    </>
  );

  const renderMusicTab = (
    <MusicTab onPlay={onPlayTrack} activeTrackId={activeTrackId} isAdmin={user.is_admin} />
  );

  return (
    <div className="sidebar">
      <div className="sidebar-header">
        <div className="sidebar-avatar" onClick={onOpenProfile} title="Настройки">
          {user.avatar ? (
            <img src={user.avatar} alt="" style={{ width: "100%", height: "100%", borderRadius: "50%", objectFit: "cover" }} />
          ) : (
            initial(user.username)
          )}
        </div>
        <div className="sidebar-user-info">
          <div className="sidebar-user-name">{user.username}</div>
          <div className="sidebar-user-status">{statusLabel}</div>
        </div>
      </div>

      <div className="sidebar-tabs">
        <button
          className={`sidebar-tab${activeTab === "chats" ? " active" : ""}${targetTab === "chats" ? " target" : ""}`}
          onClick={() => handleTabClick("chats")}
        >
          <MessageSquare size={17} /> Чаты
        </button>
        <button
          className={`sidebar-tab${activeTab === "archived" ? " active" : ""}${targetTab === "archived" ? " target" : ""}`}
          onClick={() => handleTabClick("archived")}
          title="Архив"
        >
          <Archive size={17} /> Архив
        </button>
        <button
          className={`sidebar-tab${activeTab === "music" ? " active" : ""}${targetTab === "music" ? " target" : ""}`}
          onClick={() => handleTabClick("music")}
        >
          <Music size={17} /> Музыка
        </button>
      </div>

      <div
        className={`sidebar-content${dragging ? " dragging" : ""}`}
        style={{ transform: `translateX(calc(${-pageIndex * 100}% + ${dragX}px))` }}
        onPointerDown={tabPointerDown}
        onPointerMove={tabPointerMove}
        onPointerUp={tabPointerEnd}
        onPointerCancel={tabPointerCancel}
      >
        <div className="tab-page">{renderChatsTab}</div>
        <div className="tab-page">{renderArchivedTab}</div>
        <div className="tab-page">{renderMusicTab}</div>
      </div>
    </div>
  );
}