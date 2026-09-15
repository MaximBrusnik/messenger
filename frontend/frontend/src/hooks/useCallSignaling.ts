import { useCallback, useEffect, useRef } from "react";
import type { SignalingEvent, SignalingMessage } from "../types/call";

/**
 * Owns the dedicated WebRTC signaling socket (/ws/call, proxied by the
 * gateway to call-service). Reconnects with a token-refresh on close.
 * Reconnects automatically whenever the auth token changes (account switch).
 */
export function useCallSignaling(
  onEvent: (evt: SignalingEvent) => void,
  token: string | null,
) {
  const socketRef = useRef<WebSocket | null>(null);
  const connectedRef = useRef(false);
  const onEventRef = useRef(onEvent);
  onEventRef.current = onEvent;
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const send = useCallback((msg: SignalingMessage): boolean => {
    const sock = socketRef.current;
    if (!sock || sock.readyState !== WebSocket.OPEN) return false;
    sock.send(JSON.stringify(msg));
    return true;
  }, []);

  const connected = useCallback(() => connectedRef.current, []);

  useEffect(() => {
    if (!token) return;

    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    const wsUrl = `${protocol}//${window.location.host}/ws/call?token=${token}`;
    let closed = false;

    function connect() {
      if (closed) return;
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
        connectedRef.current = false;
        if (!closed && !reconnectTimerRef.current) {
          reconnectTimerRef.current = setTimeout(() => {
            reconnectTimerRef.current = null;
            connect();
          }, 3000);
        }
      };
      sock.onerror = () => sock.close();
    }

    connect();

    return () => {
      closed = true;
      if (reconnectTimerRef.current) {
        clearTimeout(reconnectTimerRef.current);
        reconnectTimerRef.current = null;
      }
      socketRef.current?.close();
      socketRef.current = null;
      connectedRef.current = false;
    };
  }, [token]);

  return { send, connected };
}