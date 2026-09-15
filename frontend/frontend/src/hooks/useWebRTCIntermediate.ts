import { useCallback, useRef, useState } from "react";
import type { IceServer } from "../types/call";

const fallbackIce: IceServer[] = [{ urls: "stun:stun.l.google.com:19302" }];

export interface WebRTCIntermediate {
  muted: boolean;
  videoEnabled: boolean;
  localStream: MediaStream | null;
  remoteStream: MediaStream | null;
  setIceServers: (servers: IceServer[]) => void;
  setOnLocalSdp: (cb: (data: string) => void) => void;
  setOnLocalIce: (cb: (data: string) => void) => void;
  /** Caller: get local track list negotiated, then send an offer. */
  createOffer: () => Promise<void>;
  /** Caller: apply the callee's answer. */
  setRemoteSdp: (data: string) => Promise<void>;
  /** Callee: buffer an offer until local media is ready, then answer. */
  bufferOffer: (data: string) => Promise<void>;
  /** Start capturing; if a buffered offer exists, produce the answer. */
  startLocalMedia: (kind: "audio" | "video") => Promise<void>;
  addIceCandidate: (data: string) => Promise<void>;
  toggleMute: () => void;
  toggleVideo: () => void;
  close: () => void;
}

/**
 * A stable imperative WebRTC controller used by the call context. Holds the
 * RTCPeerConnection and media in refs so callbacks can be rewired without
 * recreating the peer. ICE candidates are buffered until the remote
 * description is set; the callee's offer is buffered until local media is
 * ready so the answer includes real sendrecv m-lines.
 */
export function useWebRTCIntermediate(): WebRTCIntermediate {
  const pcRef = useRef<RTCPeerConnection | null>(null);
  const localRef = useRef<MediaStream | null>(null);
  const remoteRef = useRef<MediaStream | null>(null);
  const iceRef = useRef<IceServer[]>(fallbackIce);

  const onLocalSdpRef = useRef<(data: string) => void>(() => {});
  const onLocalIceRef = useRef<(data: string) => void>(() => {});

  const remoteSetRef = useRef(false);
  const mediaReadyRef = useRef(false);
  const pendingOfferRef = useRef<string | null>(null);
  const iceBufferRef = useRef<string[]>([]);

  const [muted, setMuted] = useState(false);
  const [videoEnabled, setVideoEnabled] = useState(true);
  const [localStream, setLocalStream] = useState<MediaStream | null>(null);
  const [remoteStream, setRemoteStream] = useState<MediaStream | null>(null);

  const setIceServers = useCallback((servers: IceServer[]) => {
    iceRef.current = servers.length ? servers : fallbackIce;
  }, []);

  const setOnLocalSdp = useCallback((cb: (data: string) => void) => {
    onLocalSdpRef.current = cb;
  }, []);

  const setOnLocalIce = useCallback((cb: (data: string) => void) => {
    onLocalIceRef.current = cb;
  }, []);

  const ensureRemoteStream = useCallback(() => {
    if (!remoteRef.current) {
      remoteRef.current = new MediaStream();
      setRemoteStream(remoteRef.current);
    }
    return remoteRef.current;
  }, []);

  const createPeer = useCallback(() => {
    if (pcRef.current) return pcRef.current;
    const pc = new RTCPeerConnection({ iceServers: iceRef.current });
    pcRef.current = pc;

    pc.onicecandidate = (e) => {
      if (e.candidate) onLocalIceRef.current(JSON.stringify(e.candidate.toJSON()));
    };
    pc.ontrack = (e) => {
      const stream = ensureRemoteStream();
      e.streams.forEach((s) => s.getTracks().forEach((t) => stream.addTrack(t)));
    };
    return pc;
  }, [ensureRemoteStream]);

  const flushIceBuffer = useCallback(async (pc: RTCPeerConnection) => {
    const buf = iceBufferRef.current;
    iceBufferRef.current = [];
    for (const data of buf) {
      try {
        await pc.addIceCandidate(JSON.parse(data));
      } catch {
        // already applied / invalid; ignore
      }
    }
  }, []);

  const acceptPendingOffer = useCallback(async () => {
    const data = pendingOfferRef.current;
    if (!data) return;
    pendingOfferRef.current = null;
    const pc = createPeer();
    await pc.setRemoteDescription(new RTCSessionDescription(JSON.parse(data)));
    remoteSetRef.current = true;
    await flushIceBuffer(pc);
    const answer = await pc.createAnswer();
    await pc.setLocalDescription(answer);
    onLocalSdpRef.current(JSON.stringify(answer));
  }, [createPeer, flushIceBuffer]);

  const startLocalMedia = useCallback(
    async (kind: "audio" | "video") => {
      if (!localRef.current) {
        const stream = await navigator.mediaDevices.getUserMedia({
          audio: true,
          video: kind === "video",
        });
        localRef.current = stream;
        setLocalStream(stream);
      }
      const pc = createPeer();
      localRef.current.getTracks().forEach((t) => {
        if (!pc.getSenders().some((s) => s.track === t)) pc.addTrack(t, localRef.current!);
      });
      mediaReadyRef.current = true;
      await acceptPendingOffer();
    },
    [createPeer, acceptPendingOffer],
  );

  const createOffer = useCallback(async () => {
    const pc = createPeer();
    const offer = await pc.createOffer();
    await pc.setLocalDescription(offer);
    onLocalSdpRef.current(JSON.stringify(offer));
  }, [createPeer]);

  const setRemoteSdp = useCallback(
    async (data: string) => {
      const pc = createPeer();
      await pc.setRemoteDescription(new RTCSessionDescription(JSON.parse(data)));
      remoteSetRef.current = true;
      await flushIceBuffer(pc);
    },
    [createPeer, flushIceBuffer],
  );

  const bufferOffer = useCallback(
    async (data: string) => {
      if (mediaReadyRef.current) {
        pendingOfferRef.current = data;
        await acceptPendingOffer();
        return;
      }
      pendingOfferRef.current = data;
    },
    [acceptPendingOffer],
  );

  const addIceCandidate = useCallback(
    async (data: string) => {
      const pc = pcRef.current;
      if (!pc || !remoteSetRef.current) {
        iceBufferRef.current.push(data);
        return;
      }
      try {
        await pc.addIceCandidate(JSON.parse(data));
      } catch {
        // ignore
      }
    },
    [],
  );

  const toggleMute = useCallback(() => {
    localRef.current?.getAudioTracks().forEach((t) => (t.enabled = !t.enabled));
    setMuted((m) => !m);
  }, []);

  const toggleVideo = useCallback(() => {
    localRef.current?.getVideoTracks().forEach((t) => (t.enabled = !t.enabled));
    setVideoEnabled((v) => !v);
  }, []);

  const close = useCallback(() => {
    localRef.current?.getTracks().forEach((t) => t.stop());
    localRef.current = null;
    remoteRef.current = null;
    pcRef.current?.close();
    pcRef.current = null;
    remoteSetRef.current = false;
    mediaReadyRef.current = false;
    pendingOfferRef.current = null;
    iceBufferRef.current = [];
    setMuted(false);
    setVideoEnabled(true);
    setLocalStream(null);
    setRemoteStream(null);
  }, []);

  return {
    muted,
    videoEnabled,
    localStream,
    remoteStream,
    setIceServers,
    setOnLocalSdp,
    setOnLocalIce,
    createOffer,
    setRemoteSdp,
    bufferOffer,
    startLocalMedia,
    addIceCandidate,
    toggleMute,
    toggleVideo,
    close,
  };
}