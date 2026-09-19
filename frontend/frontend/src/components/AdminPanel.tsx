import { useCallback, useEffect, useRef, useState } from "react";
import {
  Ban,
  ChevronLeft,
  ChevronRight,
  KeyRound,
  MonitorSmartphone,
  Search,
  Shield,
  ShieldCheck,
  Trash2,
  X,
} from "lucide-react";
import {
  deleteUser,
  getAdminStats,
  getAdminUsers,
  getUserSessions,
  revokeUserSessions,
  setUserActive,
  setUserRole,
} from "../api/client";
import type { AdminStats, AdminUser, Device } from "../types";

interface Props {
  selfId: number;
  onClose: () => void;
}

type Filter = "all" | "active" | "banned" | "admins" | "bots";

const FILTERS: { id: Filter; label: string }[] = [
  { id: "all", label: "Все" },
  { id: "active", label: "Активные" },
  { id: "banned", label: "Забаненные" },
  { id: "admins", label: "Админы" },
  { id: "bots", label: "Боты" },
];

const PAGE_SIZE = 50;

function formatDate(iso?: string): string {
  if (!iso) return "—";
  const d = new Date(iso);
  if (isNaN(d.getTime())) return "—";
  const now = new Date();
  const diffMin = Math.floor((now.getTime() - d.getTime()) / 60000);
  if (diffMin < 1) return "только что";
  if (diffMin < 60) return `${diffMin} мин. назад`;
  if (diffMin < 1440) return `${Math.floor(diffMin / 60)} ч назад`;
  if (diffMin < 10080) return `${Math.floor(diffMin / 1440)} дн. назад`;
  return d.toLocaleDateString("ru-RU", { day: "numeric", month: "short", year: "numeric" });
}

const initial = (name: string) => name.charAt(0).toUpperCase() || "?";

export default function AdminPanel({ selfId, onClose }: Props) {
  const [stats, setStats] = useState<AdminStats | null>(null);
  const [users, setUsers] = useState<AdminUser[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [query, setQuery] = useState("");
  const [filter, setFilter] = useState<Filter>("all");
  const [loading, setLoading] = useState(false);
  const [expandedId, setExpandedId] = useState<number | null>(null);
  const [sessions, setSessions] = useState<Device[] | null>(null);
  const [sessionsLoading, setSessionsLoading] = useState(false);
  const debounceRef = useRef<number | null>(null);

  const loadStats = useCallback(() => {
    getAdminStats().then((res) => setStats(res.data)).catch(() => {});
  }, []);

  const loadUsers = useCallback(async () => {
    setLoading(true);
    try {
      const params: {
        q?: string;
        page?: number;
        page_size?: number;
        active?: boolean;
        is_admin?: boolean;
        is_bot?: boolean;
      } = { page, page_size: PAGE_SIZE };
      if (query) params.q = query;
      if (filter === "active") params.active = true;
      else if (filter === "banned") params.active = false;
      else if (filter === "admins") params.is_admin = true;
      else if (filter === "bots") params.is_bot = true;
      const res = await getAdminUsers(params);
      setUsers(res.data);
      setTotal(res.total);
    } catch (e) {
      alert(e instanceof Error ? e.message : "Ошибка загрузки");
    }
    setLoading(false);
  }, [page, query, filter]);

  useEffect(() => {
    loadUsers();
  }, [loadUsers]);

  useEffect(() => {
    loadStats();
  }, [loadStats]);

  function handleSearch(value: string) {
    setQuery(value);
    if (debounceRef.current) window.clearTimeout(debounceRef.current);
    debounceRef.current = window.setTimeout(() => setPage(1), 350);
  }

  useEffect(() => () => {
    if (debounceRef.current) window.clearTimeout(debounceRef.current);
  }, []);

  async function loadSessions(u: AdminUser) {
    setExpandedId((prev) => (prev === u.id ? null : u.id));
    if (expandedId === u.id) return;
    setSessions(null);
    setSessionsLoading(true);
    try {
      const res = await getUserSessions(u.id);
      setSessions(res.data);
    } catch {
      setSessions([]);
    }
    setSessionsLoading(false);
  }

  async function toggleBan(u: AdminUser) {
    const action = u.is_active ? "Заблокировать" : "Разблокировать";
    if (!confirm(`${action} пользователя ${u.username}?`)) return;
    try {
      await setUserActive(u.id, !u.is_active);
      setUsers((prev) => prev.map((x) => (x.id === u.id ? { ...x, is_active: !u.is_active } : x)));
      loadStats();
    } catch (e) {
      alert(e instanceof Error ? e.message : "Ошибка");
    }
  }

  async function toggleRole(u: AdminUser) {
    const action = u.is_admin ? "Снять права администратора" : "Назначить администратором";
    if (!confirm(`${action} (${u.username})?`)) return;
    try {
      await setUserRole(u.id, !u.is_admin);
      setUsers((prev) => prev.map((x) => (x.id === u.id ? { ...x, is_admin: !u.is_admin } : x)));
      loadStats();
    } catch (e) {
      alert(e instanceof Error ? e.message : "Ошибка");
    }
  }

  async function handleDelete(u: AdminUser) {
    if (!confirm(`Удалить пользователя ${u.username}? Действие необратимо.`)) return;
    try {
      await deleteUser(u.id);
      setUsers((prev) => prev.filter((x) => x.id !== u.id));
      setTotal((t) => Math.max(0, t - 1));
      loadStats();
    } catch (e) {
      alert(e instanceof Error ? e.message : "Ошибка");
    }
  }

  async function handleRevokeSessions(u: AdminUser) {
    if (!confirm(`Завершить все сессии пользователя ${u.username}?`)) return;
    try {
      await revokeUserSessions(u.id);
      setSessions([]);
    } catch (e) {
      alert(e instanceof Error ? e.message : "Ошибка");
    }
  }

  const isSelf = (id: number) => id === selfId;

  return (
    <div className="modal-overlay admin-overlay" onClick={(e) => e.target === e.currentTarget && onClose()}>
      <div className="admin-panel">
        <div className="admin-header">
          <div>
            <h2>Админ-панель</h2>
            {stats && <div className="admin-sub">Зарегистрировано: {stats.total_users}</div>}
          </div>
          <button className="close-btn" onClick={onClose} title="Закрыть"><X size={20} /></button>
        </div>

        {stats && (
          <div className="admin-stats">
            <div className="admin-stat"><b>{stats.total_users}</b><span>Всего</span></div>
            <div className="admin-stat online"><b>{stats.online_users}</b><span>Онлайн</span></div>
            <div className="admin-stat active"><b>{stats.active_users}</b><span>Активны</span></div>
            <div className="admin-stat banned"><b>{stats.banned_users}</b><span>Забанены</span></div>
            <div className="admin-stat warned"><b>{stats.unverified_users}</b><span>Без верификации</span></div>
            <div className="admin-stat new"><b>{stats.new_last_7days}</b><span>Новых за 7 дн.</span></div>
            <div className="admin-stat adm"><b>{stats.admins}</b><span>Админы</span></div>
            <div className="admin-stat bot"><b>{stats.bots}</b><span>Боты</span></div>
          </div>
        )}

        <div className="admin-toolbar">
          <div className="admin-search">
            <Search size={16} />
            <input
              placeholder="Поиск по имени или email"
              value={query}
              onChange={(e) => handleSearch(e.target.value)}
            />
          </div>
          <div className="admin-filters">
            {FILTERS.map((f) => (
              <button
                key={f.id}
                className={`admin-filter${filter === f.id ? " active" : ""}`}
                onClick={() => { setFilter(f.id); setPage(1); }}
              >
                {f.label}
              </button>
            ))}
          </div>
        </div>

        <div className="admin-list">
          {loading && users.length === 0 && <div className="admin-empty">Загрузка...</div>}
          {!loading && users.length === 0 && <div className="admin-empty">Пользователей не найдено</div>}
          {users.map((u) => (
            <div key={u.id} className={`admin-user${u.is_active ? "" : " banned"}${expandedId === u.id ? " expanded" : ""}`}>
              <div className="admin-user-main" onClick={() => loadSessions(u)}>
                <div className="admin-user-avatar" style={{ background: u.is_bot ? "#7c4dff" : u.is_admin ? "#d4a017" : "#3390ec" }}>
                  {u.avatar ? (
                    <img src={u.avatar} alt="" style={{ width: "100%", height: "100%", borderRadius: "50%", objectFit: "cover" }} />
                  ) : (
                    initial(u.username)
                  )}
                </div>
                <div className="admin-user-info">
                  <div className="admin-user-name">
                    {u.username}
                    {u.online && <span className="admin-online-dot" title="В сети" />}
                  </div>
                  <div className="admin-user-mail">{u.email}</div>
                  <div className="admin-user-meta">
                    {!u.email_verified && <span className="admin-badge warn">не верифицирован</span>}
                    {u.is_bot && <span className="admin-badge bot">бот</span>}
                    {u.is_admin && <span className="admin-badge adm">админ</span>}
                    {!u.is_active && <span className="admin-badge banned">забанен</span>}
                    <span className="admin-badge muted">рег. {formatDate(u.created_at)}</span>
                    {u.last_login && <span className="admin-badge muted">вход: {formatDate(u.last_login)}</span>}
                  </div>
                </div>
                {!isSelf(u.id) && (
                  <div className="admin-user-actions" onClick={(e) => e.stopPropagation()}>
                    <button
                      className={`admin-action${u.is_admin ? " danger" : ""}`}
                      onClick={() => toggleRole(u)}
                      title={u.is_admin ? "Снять права администратора" : "Назначить администратором"}
                    >
                      {u.is_admin ? <Shield size={15} /> : <ShieldCheck size={15} />}
                    </button>
                    <button
                      className={`admin-action${u.is_active ? "" : " danger"}`}
                      onClick={() => toggleBan(u)}
                      title={u.is_active ? "Заблокировать" : "Разблокировать"}
                    >
                      <Ban size={15} />
                    </button>
                    <button className="admin-action danger" onClick={() => handleDelete(u)} title="Удалить">
                      <Trash2 size={15} />
                    </button>
                  </div>
                )}
              </div>

              {expandedId === u.id && (
                <div className="admin-user-detail">
                  <div className="admin-detail-row">
                    <span>ID</span><b>#{u.id}</b>
                  </div>
                  <div className="admin-detail-row">
                    <span>Email подтверждён</span><b>{u.email_verified ? "Да" : "Нет"}</b>
                  </div>
                  <div className="admin-detail-row">
                    <span>Аккаунт</span><b>{u.is_active ? "Активен" : "Заблокирован"}</b>
                  </div>
                  <div className="admin-detail-row">
                    <span>Последний вход</span><b>{formatDate(u.last_login)}</b>
                  </div>

                  <div className="admin-sessions-title">
                    <MonitorSmartphone size={14} /> Сессии
                  </div>
                  {sessionsLoading ? (
                    <div className="admin-empty small">Загрузка сессий...</div>
                  ) : sessions === null ? (
                    <div className="admin-empty small">Нет данных</div>
                  ) : sessions.length === 0 ? (
                    <div className="admin-empty small">Активных сессий нет</div>
                  ) : (
                    <div className="admin-sessions">
                      {sessions.map((d, idx) => (
                        <div key={idx} className="admin-session">
                          <div className="admin-session-name">
                            {d.name || "Устройство"}
                            <span className="admin-badge muted">{d.platform}</span>
                          </div>
                          <div className="admin-session-sub">
                            IP: {d.ip || "—"} · последний вход {formatDate(d.last_login)}
                          </div>
                        </div>
                      ))}
                    </div>
                  )}

                  {!isSelf(u.id) && (
                    <div className="admin-detail-actions">
                      <button className="btn-danger" onClick={() => handleRevokeSessions(u)}>
                        <KeyRound size={15} /> Завершить все сессии
                      </button>
                    </div>
                  )}
                </div>
              )}
            </div>
          ))}
        </div>

        <div className="admin-pager">
          <button className={`admin-pager-btn${page <= 1 ? " disabled" : ""}`} onClick={() => setPage((p) => Math.max(1, p - 1))}>
            <ChevronLeft size={16} /> Назад
          </button>
          <span className="admin-pager-info">Стр. {page} · всего {total}</span>
          <button
            className={`admin-pager-btn${page * PAGE_SIZE >= total ? " disabled" : ""}`}
            onClick={() => setPage((p) => p + 1)}
          >
            Вперёд <ChevronRight size={16} />
          </button>
        </div>
      </div>
    </div>
  );
}