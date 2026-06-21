import UserSearch from "./UserSearch";
import type { Chat, User } from "../types";

interface Props {
  user: User;
  chats: Chat[];
  activeChat: Chat | null;
  onSelectChat: (chat: Chat) => void;
  onLogout: () => void;
  onChatCreated: (chat: Chat) => void;
  onOpenProfile: () => void;
}

function formatTime(iso?: string): string {
  if (!iso) return "";
  const d = new Date(iso);
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

export default function Sidebar({ user, chats, activeChat, onSelectChat, onLogout, onChatCreated, onOpenProfile }: Props) {
  const statusLabel = user.status === "online" ? "В сети" : `Был(а) ${formatTime(user.last_login)}`;

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
        <button className="logout-btn" onClick={onLogout} title="Выйти">🚪</button>
      </div>

      <UserSearch onChatCreated={onChatCreated} />

      <div className="chat-list">
        {chats.map((c) => (
          <div
            key={c.id}
            className={`chat-item${activeChat?.id === c.id ? " active" : ""}`}
            onClick={() => onSelectChat(c)}
          >
            <div className="chat-item-avatar">
              {c.participants?.[0] ? initial(c.participants[0].username) : "#"}
            </div>
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
          </div>
        ))}
      </div>
    </div>
  );
}
