import { useEffect, useState } from "react";
import { Bookmark, X } from "lucide-react";
import { apiRequest, getFavoritesChat } from "../api/client";
import type { Chat, Message } from "../types";

interface Props {
  message: Message;
  excludeChatId: number;
  onClose: () => void;
  onForwarded?: () => void;
}

const initial = (name: string) => name.charAt(0).toUpperCase();

export default function ForwardPicker({ message, excludeChatId, onClose, onForwarded }: Props) {
  const [chats, setChats] = useState<Chat[]>([]);
  const [favoritesChat, setFavoritesChat] = useState<Chat | null>(null);
  const [loading, setLoading] = useState(true);
  const [sendingId, setSendingId] = useState<number | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    Promise.all([
      apiRequest<{ data: Chat[] }>("/chats").catch(() => ({ data: [] as Chat[] })),
      getFavoritesChat().catch(() => ({ data: null as Chat | null })),
    ]).then(([chatsRes, favRes]) => {
      const list = (chatsRes?.data ?? []).filter(
        (c) => c.id !== excludeChatId && !c.participants?.[0]?.is_bot && !c.is_favorites
      );
      setChats(list);
      if (favRes?.data && favRes.data.id !== excludeChatId) {
        setFavoritesChat(favRes.data);
      }
    }).catch(() => setError("Не удалось загрузить чаты"))
      .finally(() => setLoading(false));
  }, [excludeChatId]);

  async function handleForward(chat: Chat) {
    setSendingId(chat.id);
    setError(null);
    try {
      await apiRequest(`/chats/${chat.id}/messages/${message.id}/forward`, "POST");
      onClose();
      onForwarded?.();
    } catch {
      setError("Не удалось переслать сообщение");
    } finally {
      setSendingId(null);
    }
  }

  const preview = message.text || (message.attachment_type ? "Вложение" : "Сообщение");

  return (
    <div className="modal-overlay" onClick={(e) => e.target === e.currentTarget && onClose()}>
      <div className="modal-content forward-picker">
        <div className="profile-modal-header">
          <h2>Переслать в...</h2>
          <button className="close-btn" onClick={onClose}><X size={18} /></button>
        </div>
        <div className="forward-preview">
          <span className="forward-preview-label">Переслать</span>
          <div className="forward-preview-text">{preview.slice(0, 120)}</div>
        </div>
        {error && <div className="forward-error">{error}</div>}
        <div className="forward-list">
          {loading ? (
            <div className="forward-empty">Загрузка...</div>
          ) : (
            <>
              {favoritesChat && (
                <div
                  className={`forward-item${sendingId === favoritesChat.id ? " sending" : ""}`}
                  onClick={() => handleForward(favoritesChat)}
                >
                  <div className="forward-item-avatar favorites-avatar"><Bookmark size={18} /></div>
                  <div className="forward-item-content">
                    <div className="forward-item-name">Избранное</div>
                  </div>
                </div>
              )}
              {chats.length === 0 && !favoritesChat ? (
                <div className="forward-empty">Нет доступных чатов</div>
              ) : (
                chats.map((c) => (
                  <div
                    key={c.id}
                    className={`forward-item${sendingId === c.id ? " sending" : ""}`}
                    onClick={() => handleForward(c)}
                  >
                    <div className="forward-item-avatar">
                      {c.participants?.[0]?.avatar ? (
                        <img src={c.participants[0].avatar} alt="" style={{ width: "100%", height: "100%", borderRadius: "50%", objectFit: "cover" }} />
                      ) : c.participants?.[0]?.username ? initial(c.participants[0].username) : "#"}
                    </div>
                    <div className="forward-item-content">
                      <div className="forward-item-name">{c.name}</div>
                    </div>
                  </div>
                ))
              )}
            </>
          )}
        </div>
      </div>
    </div>
  );
}
