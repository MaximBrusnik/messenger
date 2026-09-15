import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
  type ReactNode,
} from "react";
import type {
  CallPhase,
  CallSession,
  CallType,
  SignalingEvent,
  SignalingMessage,
} from "../types/call";
import { getCallConfig, getActiveCall } from "../api/client";
import { useCallSignaling } from "../hooks/useCallSignaling";
import { useWebRTCIntermediate } from "../hooks/useWebRTCIntermediate";
import IncomingCallModal from "../components/IncomingCallModal";
import CallScreen from "../components/CallScreen";

interface Peer {
  userId: number;
  username: string;
}

interface CallPayload {
  call_id?: number;
  caller_id?: number;
  callee_id?: number;
  call_type?: string;
  status?: string;
  end_reason?: string;
  error?: string;
  data?: string;
  value?: unknown;
}

interface CallCtx {
  phase: CallPhase;
  callType: CallType;
  peer: Peer | null;
  callSession: CallSession | null;
  error: string | null;
  supported: boolean;
  secureContext: boolean;
  startCall: (peer: Peer, type: CallType) => Promise<void>;
  acceptCall: () => Promise<void>;
  rejectCall: () => void;
  endCall: () => void;
  toggleMute: () => void;
  toggleVideo: () => void;
  muted: boolean;
  videoEnabled: boolean;
  durationMs: number;
  localStream: MediaStream | null;
  remoteStream: MediaStream | null;
}

const Ctx = createContext<CallCtx | null>(null);

export const useCall = () => {
  const ctx = useContext(Ctx);
  if (!ctx) throw new Error("useCall must be used within CallProvider");
  return ctx;
};

export function CallProvider({ children }: { children: ReactNode }) {
  const [phase, setPhase] = useState<CallPhase>("idle");
  const [peer, setPeer] = useState<Peer | null>(null);
  const [callType, setCallType] = useState<CallType>("audio");
  const [callSession, setCallSession] = useState<CallSession | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [durationMs, setDurationMs] = useState(0);

  const callIdRef = useRef<number | null>(null);
  const callTypeRef = useRef<CallType>("audio");
  const roleRef = useRef<"caller" | "callee" | null>(null);

  const supported =
    typeof window !== "undefined" &&
    typeof RTCPeerConnection !== "undefined" &&
    !!navigator.mediaDevices?.getUserMedia;
  const secureContext =
    typeof window !== "undefined" &&
    (window.location.protocol === "https:" ||
      ["localhost", "127.0.0.1"].includes(window.location.hostname));

  const webrtc = useWebRTCIntermediate();

  const showError = useCallback((msg: string) => {
    setError(msg);
    setTimeout(() => setError(null), 5000);
  }, []);

  // Wire WebRTC -> signaling socket output once.
  const sendRef = useRef<(msg: SignalingMessage) => void>(() => {});
  const attachOutbound = useCallback(() => {
    webrtc.setOnLocalSdp((sdp) => {
      if (callIdRef.current != null) {
        sendRef.current({ type: "CALL_SDP", payload: { call_id: callIdRef.current, data: sdp } });
      }
    });
    webrtc.setOnLocalIce((ice) => {
      if (callIdRef.current != null) {
        sendRef.current({ type: "CALL_ICE", payload: { call_id: callIdRef.current, data: ice } });
      }
    });
  }, [webrtc]);

  const send = useCallback((msg: SignalingMessage) => sendRef.current(msg), []);

  // ── server event dispatch ────────────────────────────────────────────
  const handleServerEvent = useCallback(
    (evt: SignalingEvent) => {
      const p = evt.payload as unknown as CallPayload;
      const callId = Number(p.call_id ?? 0);

      switch (evt.type) {
        case "CALL_RINGING": {
          const callerId = Number(p.caller_id ?? 0);
          callIdRef.current = callId;
          roleRef.current = "callee";
          callTypeRef.current = (p.call_type as CallType) || "audio";
          setCallType((p.call_type as CallType) || "audio");
          setCallSession(null);
          setPeer({ userId: callerId, username: "" });
          setPhase("incoming");
          break;
        }
        case "CALL_INVITE_ACK": {
          if (callId) callIdRef.current = callId;
          if (roleRef.current === "caller") void webrtc.createOffer();
          break;
        }
        case "CALL_ACCEPTED":
        case "CALL_ACCEPT": {
          setDurationMs(0);
          setPhase("active");
          break;
        }
        case "CALL_REJECTED": {
          webrtc.close();
          setPeer(null);
          setPhase("idle");
          callIdRef.current = null;
          roleRef.current = null;
          showError("Собеседник отклонил звонок");
          break;
        }
        case "CALL_ENDED": {
          webrtc.close();
          setPeer(null);
          setCallSession(null);
          setPhase("idle");
          callIdRef.current = null;
          roleRef.current = null;
          break;
        }
        case "CALL_ERROR": {
          const msg =
            String(p.error ?? p.end_reason ?? "") ||
            String((evt.payload as { value?: unknown })?.value ?? "Ошибка звонка");
          webrtc.close();
          setPeer(null);
          setPhase("idle");
          callIdRef.current = null;
          roleRef.current = null;
          showError(msg || "Ошибка звонка");
          break;
        }
        case "CALL_SDP": {
          const incoming = String(p.data ?? "");
          if (!incoming) break;
          if (roleRef.current === "caller") {
            void webrtc.setRemoteSdp(incoming);
          } else {
            void webrtc.bufferOffer(incoming);
          }
          break;
        }
        case "CALL_ICE": {
          const candidate = String(p.data ?? "");
          if (candidate) void webrtc.addIceCandidate(candidate);
          break;
        }
      }
    },
    [webrtc, showError],
  );

  // Start signal socket; forward outbound messages to it.
  const signal = useCallSignaling(handleServerEvent);

  useEffect(() => {
    sendRef.current = signal.send;
  }, [signal]);

// Load ICE config once.
  useEffect(() => {
    getCallConfig()
      .then((res) => webrtc.setIceServers(res.data.ice_servers))
      .catch(() => {});
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Resume an active call after a reload.
  useEffect(() => {
    getActiveCall()
      .then(({ data }) => {
        if (data && data.status === "active") {
          callIdRef.current = data.id;
          setCallSession(data);
          setDurationMs(0);
        }
      })
      .catch(() => {});
  }, []);

  useEffect(() => {
    if (phase !== "active") return;
    const start = Date.now();
    const t = setInterval(() => setDurationMs(Date.now() - start), 1000);
    return () => clearInterval(t);
  }, [phase]);

  // ── public actions ────────────────────────────────────────────────────
  const startCall = useCallback(
    async (target: Peer, type: CallType) => {
      if (!supported) {
        showError("Звонки недоступны в этом браузере (нужен HTTPS и поддержка WebRTC).");
        return;
      }
      attachOutbound();
      try {
        await webrtc.startLocalMedia(type);
      } catch {
        showError("Нет доступа к камере/микрофону.");
        return;
      }
      setPeer(target);
      setCallType(type);
      callTypeRef.current = type;
      roleRef.current = "caller";
      setCallSession(null);
      setPhase("outgoing");
      send({ type: "CALL_INVITE", payload: { callee_id: target.userId, call_type: type } });
    },
    [supported, webrtc, attachOutbound, send, showError],
  );

  const acceptCall = useCallback(async () => {
    if (!callIdRef.current) return;
    attachOutbound();
    try {
      await webrtc.startLocalMedia(callTypeRef.current);
    } catch {
      showError("Нет доступа к камере/микрофону.");
      return;
    }
    send({ type: "CALL_ACCEPT", payload: { call_id: callIdRef.current } });
    setDurationMs(0);
    setPhase("active");
  }, [webrtc, attachOutbound, send, showError]);

  const rejectCall = useCallback(() => {
    if (callIdRef.current) {
      send({ type: "CALL_REJECT", payload: { call_id: callIdRef.current, reason: "rejected" } });
    }
    webrtc.close();
    setPeer(null);
    setPhase("idle");
    callIdRef.current = null;
    roleRef.current = null;
  }, [webrtc, send]);

  const endCall = useCallback(() => {
    if (callIdRef.current) {
      send({ type: "CALL_END", payload: { call_id: callIdRef.current, reason: "normal" } });
    }
    webrtc.close();
    setPeer(null);
    setPhase("idle");
    callIdRef.current = null;
    roleRef.current = null;
  }, [webrtc, send]);

  const toggleMute = useCallback(() => webrtc.toggleMute(), [webrtc]);
  const toggleVideo = useCallback(() => webrtc.toggleVideo(), [webrtc]);

  const value: CallCtx = {
    phase,
    callType,
    peer,
    callSession,
    error,
    supported,
    secureContext,
    startCall,
    acceptCall,
    rejectCall,
    endCall,
    toggleMute,
    toggleVideo,
    muted: webrtc.muted,
    videoEnabled: webrtc.videoEnabled,
    durationMs,
    localStream: webrtc.localStream,
    remoteStream: webrtc.remoteStream,
  };

  return (
    <Ctx.Provider value={value}>
      {children}
      {phase === "incoming" && peer && <IncomingCallModal peer={peer} />}
      {(phase === "outgoing" || phase === "active") && <CallScreen />}
    </Ctx.Provider>
  );
}