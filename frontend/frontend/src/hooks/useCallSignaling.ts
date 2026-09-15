import { useCallback, useEffect, useRef } from "react";
import type { SignalingEvent, SignalingMessage } from "../types/call";

/**
 * Owns the dedicated WebRTC signaling socket (/ws/call, proxied by the
 * gateway to call-service). Reconnects with a token-refresh on close.
 */
export function useCallSignaling(
  onEvent: (evt: SignalingEvent) => void,
) {
  const socketRef = useRef<WebSocket | null>(null);
  const connectedRef = useRef(false);
  const onEventRef = useRef(onEvent);
  onEventRef.current = onEvent;
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const closedRef = useRef(false);

  const send = useCallback((msg: SignalingMessage) => {
    const sock = socketRef.current;
    if (!sock || sock.readyState !== WebSocket.OPEN) return;
    sock.send(JSON.stringify(msg));
  }, []);

  useEffect(() => {
    const token = localStorage.getItem("token");
    if (!token) return;

    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    const wsUrl = `${protocol}//${window.location.host}/ws/call?token=${token}`;

    function connect() {
      if (closedRef.current) return;
      const sock = new WebSocket(wsUrl);
      socketRef.current = sock;
      connectedRef.current = false;

      sock.onopen = () => {
        connectedRef.current = true;
      };
      sock.onmessage = (event) => {
        try {
          const data: SignalingEvent = JSON.parse(event.data);
          onEventRef.current(data);
        } catch {
          // ignore parse errors
        }
      };
      sock.onclose = () => {
        if (!closedRef.current) {
          connectedRef.current = false;
          reconnectTimerRef.current = setTimeout(connect, 3000);
        }
      };
      sock.onerror = () => sock.close();
    }

    connect();

    return () => {
      closedRef.current = true;
      if (reconnectTimerRef.current) clearTimeout(reconnectTimerRef.current);
      socketRef.current?.close();
      socketRef.current = null;
    };
  }, []);

  return { send, connected: () => connectedRef.current };
}