import { useEffect, useState, useRef } from "react";
import { CircleCheck, CircleX, Download, Music, Trash } from "lucide-react";
import { getMusic, getPendingMusic, uploadMusic, deleteMusic, approveMusic, rejectMusic, getMusicDownloadUrl } from "../api/client";
import type { MusicTrack } from "../types";

interface Props {
  onPlay: (id: number) => void;
  activeTrackId: number | null;
  isAdmin?: boolean;
}

function formatSize(bytes: number): string {
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + " KB";
  return (bytes / (1024 * 1024)).toFixed(1) + " MB";
}

export default function MusicTab({ onPlay, activeTrackId, isAdmin }: Props) {
  const [tracks, setTracks] = useState<MusicTrack[]>([]);
  const [pendingTracks, setPendingTracks] = useState<MusicTrack[]>([]);
  const [loading, setLoading] = useState(false);
  const [tab, setTab] = useState<"all" | "pending">("all");
  const fileRef = useRef<HTMLInputElement>(null);

  async function load() {
    setLoading(true);
    try {
      const [allRes, pendingRes] = await Promise.all([
        getMusic(),
        isAdmin ? getPendingMusic() : Promise.resolve(undefined),
      ]);
      setTracks(allRes?.data || []);
      if (pendingRes) setPendingTracks(pendingRes?.data || []);
    } catch { /* ignore */ }
    setLoading(false);
  }

  useEffect(() => { load(); }, [isAdmin]);

  async function handleUpload(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    try {
      await uploadMusic(file);
      load();
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Ошибка загрузки";
      alert(msg);
    }
    if (fileRef.current) fileRef.current.value = "";
  }

  async function handleDelete(id: number, e: React.MouseEvent) {
    e.stopPropagation();
    if (!confirm("Удалить трек?")) return;
    try {
      await deleteMusic(id);
      setTracks((prev) => prev.filter((t) => t.id !== id));
    } catch { /* ignore */ }
  }

  async function handleApprove(id: number) {
    try {
      await approveMusic(id);
      setPendingTracks((prev) => prev.filter((t) => t.id !== id));
      load();
    } catch { /* ignore */ }
  }

  async function handleReject(id: number) {
    if (!confirm("Отклонить трек?")) return;
    try {
      await rejectMusic(id);
      setPendingTracks((prev) => prev.filter((t) => t.id !== id));
    } catch { /* ignore */ }
  }

  if (isAdmin) {
    return (
      <div className="music-tab">
        <div className="music-header">
          <span style={{ fontWeight: 600, fontSize: 15 }}>Моя музыка</span>
          <label className="music-upload-btn">
            + Загрузить
            <input ref={fileRef} type="file" accept=".mp3,audio/*" onChange={handleUpload} hidden />
          </label>
        </div>

        <div style={{ display: "flex", gap: 8, padding: "8px 12px", borderBottom: "1px solid #e0e0e0" }}>
          <button
            className={`sidebar-tab${tab === "all" ? " active" : ""}`}
            onClick={() => setTab("all")}
          >
            Все треки
          </button>
          <button
            className={`sidebar-tab${tab === "pending" ? " active" : ""}`}
            onClick={() => setTab("pending")}
          >
            На модерации {pendingTracks.length > 0 && `(${pendingTracks.length})`}
          </button>
        </div>

        {tab === "pending" && pendingTracks.length > 0 && (
          <div className="music-list">
            {pendingTracks.map((t) => (
              <div key={t.id} className="music-item">
                <div className="music-item-icon"><Music size={18} /></div>
                <div className="music-item-info">
                  <div className="music-item-title">{t.title}</div>
                  <div className="music-item-artist">{t.artist || "Неизвестный исполнитель"}</div>
                </div>
                <div className="music-item-size">{formatSize(t.size)}</div>
                <button
                  className="music-item-approve"
                  onClick={() => handleApprove(t.id)}
                  title="Одобрить"
                >
                  <CircleCheck size={18} />
                </button>
                <button
                  className="music-item-delete"
                  onClick={() => handleReject(t.id)}
                  title="Отклонить"
                >
                  <CircleX size={18} />
                </button>
              </div>
            ))}
          </div>
        )}

        {tab === "pending" && pendingTracks.length === 0 && !loading && (
          <div className="music-empty">Нет треков на модерации</div>
        )}

        {tab === "all" && (
          loading && tracks.length === 0 ? (
            <div className="music-empty">Загрузка...</div>
          ) : tracks.length === 0 ? (
            <div className="music-empty">Нет треков. Нажмите «+ Загрузить»</div>
          ) : (
            <div className="music-list">
              {tracks.map((t) => (
                <div
                  key={t.id}
                  className={`music-item${activeTrackId === t.id ? " active" : ""}`}
                  onClick={() => onPlay(t.id)}
                >
                  <div className="music-item-icon"><Music size={18} /></div>
                  <div className="music-item-info">
                    <div className="music-item-title">{t.title}</div>
                    <div className="music-item-artist">{t.artist || "Неизвестный исполнитель"}</div>
                  </div>
                  <div className="music-item-size">{formatSize(t.size)}</div>
                  <a
                    className="music-item-download"
                    href={getMusicDownloadUrl(t.id)}
                    download
                    onClick={(e) => e.stopPropagation()}
                    title="Скачать"
                  >
                    <Download size={16} />
                  </a>
                  <button
                    className="music-item-delete"
                    onClick={(e) => handleDelete(t.id, e)}
                    title="Удалить"
                  >
                    <Trash size={16} />
                  </button>
                </div>
              ))}
            </div>
          )
        )}
      </div>
    );
  }

  return (
    <div className="music-tab">
      <div className="music-header">
        <span style={{ fontWeight: 600, fontSize: 15 }}>Моя музыка</span>
        <label className="music-upload-btn">
          + Загрузить
          <input ref={fileRef} type="file" accept=".mp3,audio/*" onChange={handleUpload} hidden />
        </label>
      </div>

      {loading && tracks.length === 0 ? (
        <div className="music-empty">Загрузка...</div>
      ) : tracks.length === 0 ? (
        <div className="music-empty">Нет треков. Нажмите «+ Загрузить»</div>
      ) : (
        <div className="music-list">
          {tracks.map((t) => (
            <div
              key={t.id}
              className={`music-item${activeTrackId === t.id ? " active" : ""}`}
              onClick={() => onPlay(t.id)}
            >
              <div className="music-item-icon"><Music size={18} /></div>
              <div className="music-item-info">
                <div className="music-item-title">{t.title}</div>
                <div className="music-item-artist">{t.artist || "Неизвестный исполнитель"}</div>
              </div>
              <div className="music-item-size">{formatSize(t.size)}</div>
              <a
                className="music-item-download"
                href={getMusicDownloadUrl(t.id)}
                download
                onClick={(e) => e.stopPropagation()}
                title="Скачать"
              >
                <Download size={16} />
              </a>
              <button
                className="music-item-delete"
                onClick={(e) => handleDelete(t.id, e)}
                title="Удалить"
              >
                <Trash size={16} />
              </button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
