import { useEffect, useRef } from "react";
import type { Message, WSMessage } from "../types";

// Per-chat live handlers registered by the open ChatArea. The global socket
// routes chat-scoped events to whichever chat registered them.
export interface ChatLiveHandlers {
  onNewMessage: (msg: Message) => void;
  onMessageEdited: (msg: Message) => void;
  onReactionChange: () => void;
  onMessageDeleted: (msgId: number) => void;
  onMessagesRead: (messageIds: number[]) => void;
  onMessagePinned: (msg: Message) => void;
  onMessageUnpinned: () => void;
}

type Handlers = {
  onConnected?: () => void;
  onAnyMessage?: () => void;
  onUserStatus: (userId: number, status: string) => void;
  onChatDeleted: (chatId: number) => void;
  route: (chatId: number, fn: (handlers: ChatLiveHandlers) => void) => void;
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
          h.current.onAnyMessage?.();

          switch (data.type) {
            case "CONNECTED":
              h.current.onConnected?.();
              break;
            case "NEW_MESSAGE": {
              const msg = data.payload.message as Message;
              h.current.route(msg.chat_id, (chl) => chl.onNewMessage(msg));
              break;
            }
            case "MESSAGE_EDITED": {
              const msg = data.payload.message as Message;
              h.current.route(msg.chat_id, (chl) => chl.onMessageEdited(msg));
              break;
            }
            case "REACTION_ADDED":
            case "REACTION_REMOVED": {
              const { chat_id } = data.payload as { chat_id?: number };
              if (chat_id) {
                h.current.route(chat_id, (chl) => chl.onReactionChange());
              }
              break;
            }
            case "USER_STATUS": {
              const { user_id, status } = data.payload as { user_id: number; status: string };
              h.current.onUserStatus(user_id, status);
              break;
            }
            case "MESSAGE_DELETED": {
              const { chat_id, message_id } = data.payload as { chat_id: number; message_id: number };
              h.current.route(chat_id, (chl) => chl.onMessageDeleted(message_id));
              break;
            }
            case "CHAT_DELETED": {
              const { chat_id } = data.payload as { chat_id: number };
              h.current.onChatDeleted(chat_id);
              break;
            }
            case "MESSAGE_PINNED": {
              const { chat_id, message } = data.payload as { chat_id: number; message: Message };
              h.current.route(chat_id, (chl) => chl.onMessagePinned(message));
              break;
            }
            case "MESSAGE_UNPINNED": {
              const { chat_id } = data.payload as { chat_id: number };
              h.current.route(chat_id, (chl) => chl.onMessageUnpinned());
              break;
            }
            case "MESSAGES_READ": {
              const { chat_id, message_ids } = data.payload as { chat_id: number; message_ids: number[] };
              h.current.route(chat_id, (chl) => chl.onMessagesRead(message_ids));
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