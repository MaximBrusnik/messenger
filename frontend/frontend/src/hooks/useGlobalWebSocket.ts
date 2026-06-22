import { useEffect, useRef } from "react";
import type { Message, WSMessage } from "../types";

type Handlers = {
  onNewMessage: (msg: Message) => void;
  onMessageEdited: (msg: Message) => void;
  onReactionAdded: () => void;
  onReactionRemoved: () => void;
  onUserStatus: (userId: number, status: string) => void;
  onMessageDeleted: (chatId: number, msgId: number) => void;
  onChatDeleted: (chatId: number) => void;
  onMessagesRead: (chatId: number, messageIds: number[]) => void;
};

export function useGlobalWebSocket(handlers: Handlers) {
  const h = useRef(handlers);
  h.current = handlers;

  useEffect(() => {
    const token = localStorage.getItem("token");
    if (!token) return;

    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    const host = window.location.host;
    const wsUrl = `${protocol}//${host}/ws?token=${token}`;

    let socket: WebSocket;
    let reconnectTimer: ReturnType<typeof setTimeout>;
    let closed = false;

    function connect() {
      if (closed) return;
      socket = new WebSocket(wsUrl);

      socket.onmessage = (event) => {
        try {
          const data: WSMessage = JSON.parse(event.data);
          switch (data.type) {
            case "NEW_MESSAGE": {
              const msg = data.payload.message as Message;
              h.current.onNewMessage(msg);
              break;
            }
            case "MESSAGE_EDITED": {
              const msg = data.payload.message as Message;
              h.current.onMessageEdited(msg);
              break;
            }
            case "REACTION_ADDED":
              h.current.onReactionAdded();
              break;
            case "REACTION_REMOVED":
              h.current.onReactionRemoved();
              break;
            case "USER_STATUS": {
              const { user_id, status } = data.payload as { user_id: number; status: string };
              h.current.onUserStatus(user_id, status);
              break;
            }
            case "MESSAGE_DELETED": {
              const { chat_id, message_id } = data.payload as { chat_id: number; message_id: number };
              h.current.onMessageDeleted(chat_id, message_id);
              break;
            }
            case "CHAT_DELETED": {
              const { chat_id } = data.payload as { chat_id: number };
              h.current.onChatDeleted(chat_id);
              break;
            }
            case "MESSAGES_READ": {
              const { chat_id, message_ids } = data.payload as { chat_id: number; message_ids: number[] };
              h.current.onMessagesRead(chat_id, message_ids);
              break;
            }
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
