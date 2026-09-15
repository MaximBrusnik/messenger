import { useEffect, useRef, useState } from "react";
import { ChevronLeft, Download, Music, Pause, Play } from "lucide-react";
import { getMusicStreamUrl, getMusicDownloadUrl } from "../api/client";
import type { MusicTrack } from "../types";

interface Props {
  track: MusicTrack;
  onBack?: () => void;
}

export default function MusicPlayer({ track, onBack }: Props) {
  const audioRef = useRef<HTMLAudioElement>(null);
  const [playing, setPlaying] = useState(false);
  const [currentTime, setCurrentTime] = useState(0);
  const [duration, setDuration] = useState(0);

  useEffect(() => {
    const audio = audioRef.current;
    if (!audio) return;

    audio.src = getMusicStreamUrl(track.id);
    audio.load();

    const onLoaded = () => setDuration(audio.duration);
    const onTimeUpdate = () => setCurrentTime(audio.currentTime);
    const onEnd = () => setPlaying(false);

    audio.addEventListener("loadedmetadata", onLoaded);
    audio.addEventListener("timeupdate", onTimeUpdate);
    audio.addEventListener("ended", onEnd);

    setPlaying(true);
    audio.play().catch(() => setPlaying(false));

    return () => {
      audio.removeEventListener("loadedmetadata", onLoaded);
      audio.removeEventListener("timeupdate", onTimeUpdate);
      audio.removeEventListener("ended", onEnd);
      audio.pause();
      audio.src = "";
    };
  }, [track.id]);

  function togglePlay() {
    const audio = audioRef.current;
    if (!audio) return;
    if (playing) {
      audio.pause();
    } else {
      audio.play();
    }
    setPlaying(!playing);
  }

  function seek(e: React.MouseEvent<HTMLDivElement>) {
    const audio = audioRef.current;
    if (!audio || !duration) return;
    const rect = e.currentTarget.getBoundingClientRect();
    const pct = (e.clientX - rect.left) / rect.width;
    audio.currentTime = pct * duration;
  }

  function formatTime(s: number): string {
    if (!s || isNaN(s)) return "0:00";
    const m = Math.floor(s / 60);
    const sec = Math.floor(s % 60);
    return `${m}:${sec.toString().padStart(2, "0")}`;
  }

  const progress = duration > 0 ? (currentTime / duration) * 100 : 0;

  return (
    <div className="music-player">
      <audio ref={audioRef} preload="metadata" />

      {onBack && <button className="back-btn" onClick={onBack}><ChevronLeft size={22} /></button>}

      <div className="music-player-header">
        <div className="music-player-icon"><Music size={34} /></div>
        <div className="music-player-title">{track.title}</div>
        <div className="music-player-artist">{track.artist || "Неизвестный исполнитель"}</div>
      </div>

      <div className="music-player-controls">
        <button className="music-play-btn" onClick={togglePlay}>
          {playing ? <Pause size={28} /> : <Play size={28} />}
        </button>

        <div className="music-progress-bar" onClick={seek}>
          <div className="music-progress-fill" style={{ width: `${progress}%` }} />
        </div>

        <div className="music-time">
          {formatTime(currentTime)} / {formatTime(duration)}
        </div>
      </div>

      <a
        className="music-download-btn"
        href={getMusicDownloadUrl(track.id)}
        download
      >
        <Download size={16} /> Скачать
      </a>
    </div>
  );
}
