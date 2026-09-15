import { useEffect, useState } from "react";
import { useCall } from "../context/CallContext";
import { getUserProfile } from "../api/client";

interface Props {
  peer: { userId: number; username: string };
}

export default function IncomingCallModal({ peer }: Props) {
  const { callType, acceptCall, rejectCall } = useCall();
  const [name, setName] = useState(peer.username || null);

  useEffect(() => {
    let alive = true;
    getUserProfile(peer.userId)
      .then((res) => {
        if (alive) setName(res.data?.username || "Пользователь");
      })
      .catch(() => {
        if (alive) setName("Пользователь");
      });
    return () => {
      alive = false;
    };
  }, [peer.userId]);

  const displayName = peer.username || name || "Пользователь";

  return (
    <div className="call-screen incoming">
      <div className="call-backdrop">
        <div className="call-avatar">{displayName.charAt(0).toUpperCase()}</div>
        <div className="call-info">
          <div className="call-peer-name">{displayName}</div>
          <div className="call-status">
            {callType === "video" ? "Входящий видеозвонок" : "Входящий звонок"}
          </div>
        </div>
        <div className="call-controls">
          <button className="call-btn hangup" onClick={rejectCall} title="Отклонить">
            ✕
          </button>
          <button className="call-btn answer" onClick={acceptCall} title="Ответить">
            📞
          </button>
        </div>
      </div>
    </div>
  );
}