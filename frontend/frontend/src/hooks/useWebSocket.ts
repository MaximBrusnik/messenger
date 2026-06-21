import { useEffect, useRef } from "react";
import type { Message, WSMessage } from "../types";

const WS_URL = "ws://localhost:8080/ws";

export function useWebSocket(
  activeChatId: number | null,
  onNewMessage: (msg: Message) => void,
  onMessageEdited: (msg: Message) => void,
  onReactionChange: () => void
) {
  const ws = useRef<WebSocket | null>(null);

  useEffect(() => {
    const token = localStorage.getItem("token");
    if (!token) return;

    const socket = new WebSocket(`${WS_URL}?token=${token}`);
    ws.current = socket;

    socket.onmessage = (event) => {
      try {
        const data: WSMessage = JSON.parse(event.data);

        switch (data.type) {
          case "NEW_MESSAGE": {
            const message = data.payload.message as Message;
            onNewMessage(message);
            break;
          }
          case "MESSAGE_EDITED": {
            const message = data.payload.message as Message;
            onMessageEdited(message);
            break;
          }
          case "REACTION_ADDED":
          case "REACTION_REMOVED":
            onReactionChange();
            break;
        }
      } catch {
        // ignore parse errors
      }
    };

    return () => {
      socket.close();
      ws.current = null;
    };
  }, [activeChatId, onNewMessage, onMessageEdited, onReactionChange]);
}
