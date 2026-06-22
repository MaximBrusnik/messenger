import { useEffect, useRef } from "react";
import type { Message, WSMessage } from "../types";

export function useWebSocket(
  chatId: number | null,
  onNewMessage: (msg: Message) => void,
  onMessageEdited: (msg: Message) => void,
  onReactionChange: () => void,
  onAnyMessage?: () => void
) {
  const chatIdRef = useRef(chatId);
  chatIdRef.current = chatId;

  const onNewMessageRef = useRef(onNewMessage);
  onNewMessageRef.current = onNewMessage;

  const onMessageEditedRef = useRef(onMessageEdited);
  onMessageEditedRef.current = onMessageEdited;

  const onReactionChangeRef = useRef(onReactionChange);
  onReactionChangeRef.current = onReactionChange;

  const onAnyMessageRef = useRef(onAnyMessage);
  onAnyMessageRef.current = onAnyMessage;

  useEffect(() => {
    const token = localStorage.getItem("token");
    if (!token) return;

    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    const wsUrl = `${protocol}//${window.location.host}/ws?token=${token}`;

    let socket: WebSocket;
    let reconnectTimer: ReturnType<typeof setTimeout>;
    let closed = false;

    function connect() {
      if (closed) return;
      socket = new WebSocket(wsUrl);

      socket.onmessage = (event) => {
        try {
          const data: WSMessage = JSON.parse(event.data);

          onAnyMessageRef.current?.();

          switch (data.type) {
            case "NEW_MESSAGE": {
              const msg = data.payload.message as Message;
              if (msg.chat_id === chatIdRef.current) {
                onNewMessageRef.current(msg);
              }
              break;
            }
            case "MESSAGE_EDITED": {
              const msg = data.payload.message as Message;
              if (msg.chat_id === chatIdRef.current) {
                onMessageEditedRef.current(msg);
              }
              break;
            }
            case "REACTION_ADDED":
            case "REACTION_REMOVED":
              onReactionChangeRef.current();
              break;
          }
        } catch {
          // ignore parse errors
        }
      };

      socket.onclose = () => {
        if (!closed) {
          reconnectTimer = setTimeout(connect, 3000);
        }
      };

      socket.onerror = () => {
        socket.close();
      };
    }

    connect();

    return () => {
      closed = true;
      clearTimeout(reconnectTimer);
      socket.close();
    };
  }, []);
}
