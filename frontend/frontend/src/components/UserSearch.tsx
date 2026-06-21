import { useState } from "react";
import { apiRequest } from "../api/client";
import type { Chat, User } from "../types";

interface Props {
  onChatCreated: (chat: Chat) => void;
}

export default function UserSearch({ onChatCreated }: Props) {
  const [q, setQ] = useState("");
  const [results, setResults] = useState<User[]>([]);
  const [open, setOpen] = useState(false);

  async function search() {
    if (!q.trim()) return;
    const data = await apiRequest<{ data: User[] }>(`/users/search?q=${q}`);
    setResults(data.data ?? []);
    setOpen(true);
  }

  async function startChat(user: User) {
    const data = await apiRequest<{ data: Chat }>("/chats", "POST", {
      user_id: user.id,
      name: user.username,
      type: "private",
    });
    onChatCreated(data.data);
    setOpen(false);
    setQ("");
  }

  const initial = (name: string) => name.charAt(0).toUpperCase();

  return (
    <div className="user-search" style={{ position: "relative" }}>
      <input
        value={q}
        onChange={(e) => setQ(e.target.value)}
        onKeyDown={(e) => e.key === "Enter" && search()}
        onFocus={() => results.length > 0 && setOpen(true)}
        placeholder="Поиск пользователей..."
      />

      {open && results.length > 0 && (
        <div className="search-results">
          {results.map((u) => (
            <div key={u.id} className="search-result-item" onClick={() => startChat(u)}>
              <div style={{
                width: 36, height: 36, borderRadius: "50%", background: "#3390ec",
                color: "#fff", display: "flex", alignItems: "center", justifyContent: "center",
                fontWeight: 600, fontSize: 15, flexShrink: 0,
              }}>
                {initial(u.username)}
              </div>
              <div>
                <div style={{ fontWeight: 500, fontSize: 14 }}>{u.username}</div>
                <div style={{ fontSize: 12, color: "#888" }}>Нажми чтобы написать</div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
