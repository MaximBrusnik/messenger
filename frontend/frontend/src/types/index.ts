export interface User {
  id: number;
  username: string;
  email: string;
  created_at: string;
  last_login?: string;
  avatar?: string;
  status?: string;
  email_verified: boolean;
  bio?: string;
  date_of_birth?: string;
  common_chats?: number;
  is_bot?: boolean;
}

export interface Chat {
  id: number;
  name: string;
  type: string;
  created_at: string;
  updated_at: string;
  participants?: User[];
  last_message?: Message;
  unread?: number;
}

export interface Message {
  id: number;
  chat_id: number;
  sender_id: number;
  text: string;
  is_read: boolean;
  read_at?: string;
  created_at: string;
  edited?: boolean;
  edited_at?: string;
  sender?: User;
  reactions?: Reaction[];
  attachment_type?: string;
  attachment_url?: string;
  attachment_name?: string;
  attachment_size?: number;
}

export interface Reaction {
  message_id: number;
  user_id: number;
  reaction: string;
  created_at?: string;
  username?: string;
}

export interface UserSettings {
  show_online_status: boolean;
  last_seen_privacy: "everyone" | "contacts" | "nobody";
  avatar_privacy: "everyone" | "contacts" | "nobody";
}

export interface WSMessage {
  type: "NEW_MESSAGE" | "MESSAGE_EDITED" | "REACTION_ADDED" | "REACTION_REMOVED" | "USER_STATUS" | "CONNECTED" | "MESSAGE_DELETED" | "CHAT_DELETED" | "MESSAGES_READ";
  payload: Record<string, unknown>;
}
