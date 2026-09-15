import { useEffect, useState } from "react";
import { LoaderCircle, MessageSquare, X } from "lucide-react";
import { getUserProfile } from "../api/client";
import type { User } from "../types";

interface Props {
  userId: number;
  onClose: () => void;
  onStartChat?: (userId: number) => void;
}

function formatDate(iso?: string): string {
  if (!iso) return "";
  const d = new Date(iso);
  return d.toLocaleDateString("ru-RU", { day: "numeric", month: "long", year: "numeric" });
}

export default function UserProfileModal({ userId, onClose, onStartChat }: Props) {
  const [profile, setProfile] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    getUserProfile(userId).then((res) => {
      setProfile(res.data);
    }).catch(() => {}).finally(() => setLoading(false));
  }, [userId]);

  const initial = profile?.username?.charAt(0).toUpperCase() ?? "?";

  return (
    <div className="modal-overlay" onClick={(e) => e.target === e.currentTarget && onClose()}>
      <div className="modal-content user-profile-modal">
        <div className="profile-modal-header">
          <h2>Профиль</h2>
          <button className="close-btn" onClick={onClose}><X size={18} /></button>
        </div>

        <div className="profile-body">
          {loading ? (
            <div className="profile-loading"><LoaderCircle size={28} className="spin" /></div>
          ) : !profile ? (
            <div className="profile-loading" style={{ color: "#e53935" }}>Пользователь не найден</div>
          ) : (
            <>
              <div className="avatar-section">
                <div className="avatar-large" style={{ cursor: "default" }}>
                  {profile.avatar ? (
                    <img src={profile.avatar} alt="avatar" style={{ width: "100%", height: "100%", objectFit: "cover" }} />
                  ) : (
                    initial
                  )}
                </div>
              </div>

              <div className="user-profile-name">{profile.username}</div>
              <div className="user-profile-status">
                {profile.status === "online" ? "В сети" : profile.last_login ? `Был(а) ${formatDate(profile.last_login)}` : "Не в сети"}
              </div>

              {profile.bio && (
                <div className="form-row">
                  <label>О себе</label>
                  <div className="user-profile-bio">{profile.bio}</div>
                </div>
              )}

              {profile.date_of_birth && (
                <div className="form-row">
                  <label>Дата рождения</label>
                  <div className="user-profile-field">{formatDate(profile.date_of_birth)}</div>
                </div>
              )}

              <div className="form-row">
                <label>На сайте с</label>
                <div className="user-profile-field">{formatDate(profile.created_at)}</div>
              </div>

              {profile.common_chats !== undefined && (
                <div className="form-row">
                  <label>Общие чаты</label>
                  <div className="user-profile-field">{profile.common_chats}</div>
                </div>
              )}

              {onStartChat && (
                <button className="btn-primary" onClick={() => onStartChat(userId)}>
                  <MessageSquare size={16} /> Написать сообщение
                </button>
              )}
            </>
          )}
        </div>
      </div>
    </div>
  );
}