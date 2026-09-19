import { useEffect, useRef, useState } from "react";
import {
  Bell,
  CircleCheck,
  CircleX,
  KeyRound,
  LogOut,
  MonitorSmartphone,
  Palette,
  Shield,
  User as UserIcon,
  X,
} from "lucide-react";
import { apiRequest, getDevices, uploadFile } from "../api/client";
import type { Device, User, UserSettings } from "../types";
import { useAuth } from "../context/AuthContext";
import { useTheme } from "../context/ThemeContext";
import { useSettings } from "../context/SettingsContext";

interface Props {
  onClose: () => void;
  onOpenAdmin?: () => void;
}

type Tab = "profile" | "privacy" | "password" | "theme" | "notifications" | "devices";

export default function ProfileModal({ onClose, onOpenAdmin }: Props) {
  const { user, setUser } = useAuth();
  const { theme, toggle } = useTheme();
  const { settings, setSoundEnabled } = useSettings();
  const [tab, setTab] = useState<Tab>("profile");
  const [username, setUsername] = useState(user?.username ?? "");
  const [email, setEmail] = useState(user?.email ?? "");
  const [bio, setBio] = useState(user?.bio ?? "");
  const [dateOfBirth, setDateOfBirth] = useState(user?.date_of_birth ?? "");
  const [saving, setSaving] = useState(false);
  const [success, setSuccess] = useState<string | null>(null);
  const fileRef = useRef<HTMLInputElement>(null);

  // Devices
  const [devices, setDevices] = useState<Device[] | null>(null);
  const [devicesErr, setDevicesErr] = useState<string | null>(null);

  // Privacy
  const [showOnline, setShowOnline] = useState(true);
  const [lastSeenPrivacy, setLastSeenPrivacy] = useState<string>("everyone");
  const [avatarPrivacy, setAvatarPrivacy] = useState<string>("everyone");

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
      setAvatarPrivacy(res.data.avatar_privacy);
    }).catch(() => {});
  }, []);

  useEffect(() => {
    if (tab !== "devices") return;
    let active = true;
    getDevices().then((res) => {
      if (!active) return;
      setDevicesErr(null);
      setDevices(res.data);
    }).catch(() => {
      if (active) setDevicesErr("Не удалось загрузить список устройств");
    });
    return () => { active = false; };
  }, [tab]);

  function formatDeviceDate(iso?: string): string {
    if (!iso) return "—";
    const date = new Date(iso);
    if (isNaN(date.getTime())) return "—";
    return date.toLocaleString("ru-RU", { day: "numeric", month: "short", hour: "2-digit", minute: "2-digit" });
  }

  async function saveProfile() {
    setSaving(true);
    setSuccess(null);
    try {
      const body: Record<string, unknown> = { username, email, bio };
      if (dateOfBirth) body.date_of_birth = dateOfBirth;
      const res = await apiRequest<{ data: User }>("/auth/profile", "PUT", body);
      setUser(res.data);
      setSuccess("Профиль обновлён");
      setTimeout(() => setSuccess(null), 2000);
    } catch (e) {
      alert(e instanceof Error ? e.message : "Ошибка");
    }
    setSaving(false);
  }

  function handleLogout() {
    localStorage.removeItem("token");
    setUser(null);
    onClose();
  }

  async function handleAvatarUpload(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    if (fileRef.current) fileRef.current.value = "";
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

  async function handleDeleteAvatar() {
    try {
      const res = await apiRequest<{ data: User }>("/auth/profile", "PUT", { avatar: "" });
      setUser(res.data);
      setSuccess("Аватар удалён");
      setTimeout(() => setSuccess(null), 2000);
    } catch {
      alert("Ошибка");
    }
  }

  async function saveSettings() {
    setSaving(true);
    try {
      await apiRequest("/auth/settings", "PUT", {
        show_online_status: showOnline,
        last_seen_privacy: lastSeenPrivacy,
        avatar_privacy: avatarPrivacy,
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
          <div style={{ display: "flex", gap: 6 }}>
            {user?.is_admin && onOpenAdmin && (
              <button
                className="admin-entry-btn"
                onClick={onOpenAdmin}
                title="Админ-панель"
              >
                <Shield size={18} />
              </button>
            )}
            <button className="close-btn" onClick={onClose}><X size={18} /></button>
          </div>
        </div>

        <div className="profile-tabs">
          <div className={`profile-tab${tab === "profile" ? " active" : ""}`} onClick={() => setTab("profile")}><UserIcon size={15} /> Профиль</div>
          <div className={`profile-tab${tab === "privacy" ? " active" : ""}`} onClick={() => setTab("privacy")}><Shield size={15} /> Приватность</div>
          <div className={`profile-tab${tab === "notifications" ? " active" : ""}`} onClick={() => setTab("notifications")}><Bell size={15} /> Уведомления</div>
          <div className={`profile-tab${tab === "devices" ? " active" : ""}`} onClick={() => setTab("devices")}><MonitorSmartphone size={15} /> Устройства</div>
          <div className={`profile-tab${tab === "password" ? " active" : ""}`} onClick={() => setTab("password")}><KeyRound size={15} /> Пароль</div>
          <div className={`profile-tab${tab === "theme" ? " active" : ""}`} onClick={() => setTab("theme")}><Palette size={15} /> Тема</div>
        </div>

        <div className="profile-body">
          {tab === "profile" && (
            <>
              <div className="avatar-section">
                <label className="avatar-large" title="Сменить аватар">
                  {user?.avatar ? (
                    <img src={user.avatar} alt="avatar" style={{ width: "100%", height: "100%", objectFit: "cover" }} />
                  ) : (
                    initial
                  )}
                  <div className="avatar-overlay">Сменить</div>
                  <input ref={fileRef} type="file" accept="image/*" onChange={handleAvatarUpload} />
                </label>
                {user?.avatar && (
                  <button
                    onClick={handleDeleteAvatar}
                    style={{ background: "none", border: "none", color: "#e53935", cursor: "pointer", fontSize: 13 }}
                  >
                    Удалить аватар
                  </button>
                )}
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
                    <span className="verified-badge"><CircleCheck size={15} /> Подтверждён</span>
                  ) : (
                    <span className="verified-badge unverified"><CircleX size={15} /> Не подтверждён</span>
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

              <div className="form-row">
                <label>О себе</label>
                <textarea value={bio} onChange={(e) => setBio(e.target.value)} rows={3} maxLength={500} />
              </div>

              <div className="form-row">
                <label>Дата рождения</label>
                <input type="date" value={dateOfBirth} onChange={(e) => setDateOfBirth(e.target.value)} />
              </div>

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
              <div className="form-row">
                <label>Кто видит аватарку</label>
                <select value={avatarPrivacy} onChange={(e) => setAvatarPrivacy(e.target.value)}>
                  <option value="everyone">Все</option>
                  <option value="contacts">Только контакты</option>
                  <option value="nobody">Никто</option>
                </select>
              </div>

              <button className="btn-primary" onClick={saveSettings} disabled={saving}>
                {saving ? "Сохранение..." : "Сохранить"}
              </button>

              <hr style={{ border: "none", borderTop: "1px solid var(--border)", margin: "12px 0" }} />

              <button className="btn-danger" onClick={handleLogout}>
                <LogOut size={16} /> Выйти из аккаунта
              </button>
            </>
          )}

          {tab === "notifications" && (
            <>
              <div className="form-row theme-row">
                <div>
                  <label>Звук новых сообщений</label>
                  <div className="form-hint">Проигрывать звук при получении нового сообщения</div>
                </div>
                <label className="theme-switch">
                  <input
                    type="checkbox"
                    checked={settings?.sound_enabled !== false}
                    onChange={(e) => setSoundEnabled(e.target.checked)}
                  />
                  <span className="theme-switch-track">
                    <span className="theme-switch-thumb" />
                  </span>
                </label>
              </div>

              <hr style={{ border: "none", borderTop: "1px solid var(--border)", margin: "12px 0" }} />

              <button className="btn-danger" onClick={handleLogout}>
                <LogOut size={16} /> Выйти из аккаунта
              </button>
            </>
          )}

          {tab === "devices" && (
            <>
              {devicesErr && <div style={{ color: "#e53935", fontSize: 14, textAlign: "center" }}>{devicesErr}</div>}
              {!devices && !devicesErr && (
                <div className="form-hint" style={{ textAlign: "center", padding: 16 }}>Загрузка...</div>
              )}
              {devices && devices.length === 0 && (
                <div className="form-hint" style={{ textAlign: "center", padding: 16 }}>Устройств нет</div>
              )}
              {devices && devices.length > 0 && (
                <div className="devices-list">
                  {devices.map((d, idx) => (
                    <div key={idx} className={`device-row${d.is_current ? " current" : ""}`}>
                      <div className="device-icon">
                        <MonitorSmartphone size={20} />
                      </div>
                      <div className="device-info">
                        <div className="device-name">
                          {d.name}
                          {d.is_current && <span className="device-current">Это устройство</span>}
                        </div>
                        <div className="device-sub">
                          {d.platform}
                          {d.ip && d.ip !== "" && <span> · {d.ip}</span>}
                        </div>
                        <div className="device-sub">
                          {d.login_count > 1 ? `Входов: ${d.login_count}` : "Первый вход"} · последний — {formatDeviceDate(d.last_login)}
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              )}

              <hr style={{ border: "none", borderTop: "1px solid var(--border)", margin: "12px 0" }} />

              <button className="btn-danger" onClick={handleLogout}>
                <LogOut size={16} /> Выйти из аккаунта
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

          {tab === "theme" && (
            <>
              <div className="form-row theme-row">
                <div>
                  <label>Тёмная тема</label>
                  <div className="form-hint">Применить тёмное оформление</div>
                </div>
                <label className="theme-switch">
                  <input type="checkbox" checked={theme === "dark"} onChange={toggle} />
                  <span className="theme-switch-track">
                    <span className="theme-switch-thumb" />
                  </span>
                </label>
              </div>

              <hr style={{ border: "none", borderTop: "1px solid var(--border)", margin: "12px 0" }} />

              <button className="btn-danger" onClick={handleLogout}>
                <LogOut size={16} /> Выйти из аккаунта
              </button>
            </>
          )}
        </div>
      </div>
    </div>
  );
}
