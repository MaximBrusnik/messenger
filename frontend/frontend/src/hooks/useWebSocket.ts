import { useEffect, useRef } from "react";
import type { Message, WSMessage } from "../types";

export function useWebSocket(
  chatId: number | null,
  onNewMessage: (msg: Message) => void,
  onMessageEdited: (msg: Message) => void,
  onReactionChange: () => void,
  onAnyMessage?: () => void,
  onUserStatus?: (userId: number, status: string) => void,
  onMessageDeleted?: (msgId: number) => void,
  onChatDeleted?: (chatId: number) => void,
  onMessagesRead?: (chatId: number, messageIds: number[]) => void,
  onMessagePinned?: (chatId: number, msg: Message) => void,
  onMessageUnpinned?: (chatId: number) => void,
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

  const onUserStatusRef = useRef(onUserStatus);
  onUserStatusRef.current = onUserStatus;

  const onMessageDeletedRef = useRef(onMessageDeleted);
  onMessageDeletedRef.current = onMessageDeleted;

  const onChatDeletedRef = useRef(onChatDeleted);
  onChatDeletedRef.current = onChatDeleted;

  const onMessagesReadRef = useRef(onMessagesRead);
  onMessagesReadRef.current = onMessagesRead;

  const onMessagePinnedRef = useRef(onMessagePinned);
  onMessagePinnedRef.current = onMessagePinned;

  const onMessageUnpinnedRef = useRef(onMessageUnpinned);
  onMessageUnpinnedRef.current = onMessageUnpinned;

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
            case "USER_STATUS": {
              const { user_id, status } = data.payload as { user_id: number; status: string };
              onUserStatusRef.current?.(user_id, status);
              break;
            }
            case "MESSAGE_DELETED": {
              const { message_id } = data.payload as { message_id: number };
              onMessageDeletedRef.current?.(message_id);
              break;
            }
            case "CHAT_DELETED": {
              const { chat_id } = data.payload as { chat_id: number };
              onChatDeletedRef.current?.(chat_id);
              break;
            }
            case "MESSAGE_PINNED": {
              const { chat_id, message } = data.payload as { chat_id: number; message: Message };
              onMessagePinnedRef.current?.(chat_id, message);
              break;
            }
            case "MESSAGE_UNPINNED": {
              const { chat_id } = data.payload as { chat_id: number };
              onMessageUnpinnedRef.current?.(chat_id);
              break;
            }
            case "MESSAGES_READ": {
              const { chat_id, message_ids } = data.payload as { chat_id: number; message_ids: number[] };
              onMessagesReadRef.current?.(chat_id, message_ids);
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
