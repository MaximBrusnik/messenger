import { useEffect, useRef } from "react";
import { useCall } from "../context/CallContext";

function formatDuration(ms: number): string {
  const s = Math.floor(ms / 1000);
  const m = Math.floor(s / 60);
  const h = Math.floor(m / 60);
  const ss = String(s % 60).padStart(2, "0");
  const mm = String(m % 60).padStart(2, "0");
  return h > 0 ? `${h}:${mm}:${ss}` : `${mm}:${ss}`;
}

export default function CallScreen() {
  const {
    phase,
    callType,
    peer,
    durationMs,
    muted,
    videoEnabled,
    localStream,
    remoteStream,
    toggleMute,
    toggleVideo,
    endCall,
  } = useCall();

  const localVideoRef = useRef<HTMLVideoElement>(null);
  const remoteVideoRef = useRef<HTMLVideoElement>(null);

  useEffect(() => {
    const v = localVideoRef.current;
    if (v && localStream) {
      v.srcObject = localStream;
      void v.play().catch(() => {});
    }
  }, [localStream]);

  useEffect(() => {
    const v = remoteVideoRef.current;
    if (v && remoteStream) {
      v.srcObject = remoteStream;
      void v.play().catch(() => {});
    }
  }, [remoteStream]);

  const isVideo = callType === "video";
  const ringing = phase === "outgoing";

  return (
    <div className="call-screen">
      <div className="call-backdrop">
        {isVideo && remoteStream ? (
          <video ref={remoteVideoRef} className="call-video-remote" playsInline autoPlay muted={false} />
        ) : (
          <div className="call-avatar">
            {(peer?.username ?? "").charAt(0).toUpperCase() || "?"}
          </div>
        )}

        {isVideo && localStream && (
          <video
            ref={localVideoRef}
            className="call-video-local"
            playsInline autoPlay muted
          />
        )}

        <div className="call-info">
          <div className="call-peer-name">{peer?.username || "Пользователь"}</div>
          <div className="call-status">
            {ringing ? "Вызов..." : phase === "active" ? formatDuration(durationMs) : ""}
          </div>
        </div>

        <div className="call-controls">
          {isVideo && phase === "active" && (
            <button
              className={`call-btn ${videoEnabled ? "" : "off"}`}
              onClick={toggleVideo}
              title={videoEnabled ? "Выключить камеру" : "Включить камеру"}
            >
              {videoEnabled ? "🎥" : "🚫"}
            </button>
          )}
          {phase === "active" && (
            <button
              className={`call-btn ${muted ? "off" : ""}`}
              onClick={toggleMute}
              title={muted ? "Включить микрофон" : "Выключить микрофон"}
            >
              {muted ? "🔇" : "🎤"}
            </button>
          )}
          <button className="call-btn hangup" onClick={endCall} title="Завершить">
            {phase === "outgoing" ? "✕" : "📞"}
          </button>
        </div>
      </div>
    </div>
  );
}