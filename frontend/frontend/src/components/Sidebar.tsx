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

interface Props {
  user: User;
  chats: Chat[];
  archivedChats: Chat[];
  activeChat: Chat | null;
  activeTab: "chats" | "music" | "archived";
  activeTrackId: number | null;
  onSelectChat: (chat: Chat) => void;
  onOpenUserProfile: (userId: number) => void;
  onOpenProfile: () => void;
  onDeleteChat: (chatId: number) => void;
  onArchiveChat: (chatId: number) => void;
  onUnarchiveChat: (chatId: number) => void;
  onStartAIChat?: () => void;
  onStartFavoritesChat?: () => void;
  onTabChange: (tab: "chats" | "music" | "archived") => void;
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

export default function Sidebar({
  user, chats, archivedChats, activeChat, activeTab, activeTrackId,
  onSelectChat, onOpenUserProfile, onOpenProfile, onDeleteChat,
  onArchiveChat, onUnarchiveChat, onStartAIChat, onStartFavoritesChat,
  onTabChange, onPlayTrack,
}: Props) {
  const statusLabel = user.status === "online" ? "В сети" : `Был(а) ${formatTime(user.last_login)}`;
  const isArchived = activeTab === "archived";

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
          className={`sidebar-tab${activeTab === "chats" ? " active" : ""}`}
          onClick={() => onTabChange("chats")}
        >
          <MessageSquare size={17} /> Чаты
        </button>
        <button
          className={`sidebar-tab${activeTab === "archived" ? " active" : ""}`}
          onClick={() => onTabChange("archived")}
          title="Архив"
        >
          <Archive size={17} /> Архив
        </button>
        <button
          className={`sidebar-tab${activeTab === "music" ? " active" : ""}`}
          onClick={() => onTabChange("music")}
        >
          <Music size={17} /> Музыка
        </button>
      </div>

      {activeTab === "music" ? (
        <MusicTab onPlay={onPlayTrack} activeTrackId={activeTrackId} isAdmin={user.is_admin} />
      ) : isArchived ? (
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
              <div
                key={c.id}
                className={`chat-item${activeChat?.id === c.id ? " active" : ""}`}
                onClick={() => onSelectChat(c)}
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
                {onUnarchiveChat && (
                  <button
                    className="chat-item-action"
                    onClick={(e) => { e.stopPropagation(); onUnarchiveChat(c.id); }}
                    title="Разархивировать"
                  >
                    <ArchiveRestore size={14} />
                  </button>
                )}
              </div>
            ))}
          </div>
        </>
      ) : (
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
              <div
                key={c.id}
                className={`chat-item${activeChat?.id === c.id ? " active" : ""}`}
                onClick={() => onSelectChat(c)}
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
              </div>
            ))}
          </div>
        </>
      )}
    </div>
  );
}