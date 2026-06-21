import { useEffect, useRef, useState } from "react";
import { apiRequest, uploadFile } from "../api/client";
import type { User, UserSettings } from "../types";
import { useAuth } from "../context/AuthContext";

interface Props {
  onClose: () => void;
}

type Tab = "profile" | "privacy" | "password";

export default function ProfileModal({ onClose }: Props) {
  const { user, setUser } = useAuth();
  const [tab, setTab] = useState<Tab>("profile");
  const [username, setUsername] = useState(user?.username ?? "");
  const [email, setEmail] = useState(user?.email ?? "");
  const [saving, setSaving] = useState(false);
  const [success, setSuccess] = useState<string | null>(null);
  const fileRef = useRef<HTMLInputElement>(null);

  // Privacy
  const [showOnline, setShowOnline] = useState(true);
  const [lastSeenPrivacy, setLastSeenPrivacy] = useState<string>("everyone");

  // Password
  const [oldPwd, setOldPwd] = useState("");
  const [newPwd, setNewPwd] = useState("");
  const [pwdSuccess, setPwdSuccess] = useState<string | null>(null);

  const [verifyMsg, setVerifyMsg] = useState<string | null>(null);
  const [verifyErr, setVerifyErr] = useState<string | null>(null);

  useEffect(() => {
    apiRequest<{ data: UserSettings }>("/auth/settings").then((res) => {
      setShowOnline(res.data.show_online_status);
      setLastSeenPrivacy(res.data.last_seen_privacy);
    }).catch(() => {});
  }, []);

  async function saveProfile() {
    setSaving(true);
    setSuccess(null);
    try {
      const res = await apiRequest<{ data: User }>("/auth/profile", "PUT", { username, email });
      setUser(res.data);
      setSuccess("Профиль обновлён");
      setTimeout(() => setSuccess(null), 2000);
    } catch (e) {
      alert(e instanceof Error ? e.message : "Ошибка");
    }
    setSaving(false);
  }

  async function handleAvatarUpload(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    try {
      const data = await uploadFile(file);
      const res = await apiRequest<{ data: User }>("/auth/profile", "PUT", { avatar: data.url });
      setUser(res.data);
      setSuccess("Аватар обновлён");
      setTimeout(() => setSuccess(null), 2000);
    } catch {
      alert("Ошибка загрузки");
    }
  }

  async function saveSettings() {
    setSaving(true);
    try {
      await apiRequest("/auth/settings", "PUT", {
        show_online_status: showOnline,
        last_seen_privacy: lastSeenPrivacy,
      });
      setSuccess("Настройки сохранены");
      setTimeout(() => setSuccess(null), 2000);
    } catch {
      alert("Ошибка сохранения");
    }
    setSaving(false);
  }

  async function changePassword() {
    setSaving(true);
    setPwdSuccess(null);
    try {
      await apiRequest("/auth/change-password", "POST", {
        old_password: oldPwd,
        new_password: newPwd,
      });
      setPwdSuccess("Пароль изменён");
      setOldPwd("");
      setNewPwd("");
      setTimeout(() => setPwdSuccess(null), 2000);
    } catch (e) {
      alert(e instanceof Error ? e.message : "Ошибка");
    }
    setSaving(false);
  }

  async function resendVerification() {
    setVerifyMsg(null);
    setVerifyErr(null);
    try {
      const res = await apiRequest<{ message: string }>("/auth/resend-verification", "POST");
      setVerifyMsg(res.message);
      setTimeout(() => setVerifyMsg(null), 4000);
    } catch (e) {
      setVerifyErr(e instanceof Error ? e.message : "Ошибка");
      setTimeout(() => setVerifyErr(null), 4000);
    }
  }

  const initial = user?.username?.charAt(0).toUpperCase() ?? "?";

  return (
    <div className="modal-overlay" onClick={(e) => e.target === e.currentTarget && onClose()}>
      <div className="modal-content">
        <div className="profile-modal-header">
          <h2>Настройки</h2>
          <button className="close-btn" onClick={onClose}>✕</button>
        </div>

        <div className="profile-tabs">
          <div className={`profile-tab${tab === "profile" ? " active" : ""}`} onClick={() => setTab("profile")}>Профиль</div>
          <div className={`profile-tab${tab === "privacy" ? " active" : ""}`} onClick={() => setTab("privacy")}>Приватность</div>
          <div className={`profile-tab${tab === "password" ? " active" : ""}`} onClick={() => setTab("password")}>Пароль</div>
        </div>

        <div className="profile-body">
          {tab === "profile" && (
            <>
              <div className="avatar-section">
                <div className="avatar-large" onClick={() => fileRef.current?.click()}>
                  {user?.avatar ? (
                    <img src={user.avatar} alt="avatar" style={{ width: "100%", height: "100%", objectFit: "cover" }} />
                  ) : (
                    initial
                  )}
                  <div className="avatar-overlay">Сменить</div>
                  <input ref={fileRef} type="file" accept="image/*" onChange={handleAvatarUpload} />
                </div>
              </div>

              {success && <div className="success-msg">{success}</div>}

              <div className="form-row">
                <label>Имя пользователя</label>
                <input value={username} onChange={(e) => setUsername(e.target.value)} />
              </div>

              <div className="form-row">
                <label>Email</label>
                <div style={{ display: "flex", gap: 8, alignItems: "center" }}>
                  <input value={email} onChange={(e) => setEmail(e.target.value)} type="email" style={{ flex: 1 }} />
                  {user?.email_verified ? (
                    <span style={{ color: "#43a047", fontSize: 13, whiteSpace: "nowrap" }}>✅ Подтверждён</span>
                  ) : (
                    <span style={{ color: "#e53935", fontSize: 13, whiteSpace: "nowrap" }}>❌ Не подтверждён</span>
                  )}
                </div>
              </div>

              {!user?.email_verified && (
                <div style={{ textAlign: "center" }}>
                  <button
                    className="btn-primary"
                    onClick={resendVerification}
                    style={{ background: "none", color: "#3390ec", padding: 6, fontSize: 13 }}
                  >
                    Отправить письмо повторно
                  </button>
                  {verifyMsg && <div className="success-msg">{verifyMsg}</div>}
                  {verifyErr && <div style={{ color: "#e53935", fontSize: 14, textAlign: "center" }}>{verifyErr}</div>}
                </div>
              )}

              <button className="btn-primary" onClick={saveProfile} disabled={saving}>
                {saving ? "Сохранение..." : "Сохранить"}
              </button>
            </>
          )}

          {tab === "privacy" && (
            <>
              {success && <div className="success-msg">{success}</div>}

              <div className="form-row">
                <label>Показывать статус онлайн</label>
                <select value={showOnline ? "true" : "false"} onChange={(e) => setShowOnline(e.target.value === "true")}>
                  <option value="true">Да</option>
                  <option value="false">Нет</option>
                </select>
              </div>

              <div className="form-row">
                <label>Кто видит время последнего захода</label>
                <select value={lastSeenPrivacy} onChange={(e) => setLastSeenPrivacy(e.target.value)}>
                  <option value="everyone">Все</option>
                  <option value="contacts">Только контакты</option>
                  <option value="nobody">Никто</option>
                </select>
              </div>

              <button className="btn-primary" onClick={saveSettings} disabled={saving}>
                {saving ? "Сохранение..." : "Сохранить"}
              </button>
            </>
          )}

          {tab === "password" && (
            <>
              {pwdSuccess && <div className="success-msg">{pwdSuccess}</div>}

              <div className="form-row">
                <label>Старый пароль</label>
                <input type="password" value={oldPwd} onChange={(e) => setOldPwd(e.target.value)} />
              </div>

              <div className="form-row">
                <label>Новый пароль</label>
                <input type="password" value={newPwd} onChange={(e) => setNewPwd(e.target.value)} />
              </div>

              <button className="btn-primary" onClick={changePassword} disabled={saving || !oldPwd || !newPwd}>
                {saving ? "Сохранение..." : "Сменить пароль"}
              </button>
            </>
          )}
        </div>
      </div>
    </div>
  );
}
