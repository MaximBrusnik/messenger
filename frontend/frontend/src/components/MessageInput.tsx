import { useState, useRef } from "react";
import { uploadFile } from "../api/client";
import type { Message } from "../types";
import { FaceSlightlySmiling, Paperclip, Send, X } from "lucide-react";

const emojis = [
  "👍", "❤️", "🔥", "😂", "😮", "😢", "🙏",
  "🎉", "👏", "💯", "🥰", "😍", "🤣", "😭",
  "😡", "🤔", "👀", "💪", "🤝", "✨", "⭐",
];

interface Props {
  onSend: (text: string, attachment?: { url: string; name: string; size: number; type: string }, replyToId?: number) => void;
  replyTo?: Message | null;
  onCancelReply?: () => void;
}

function replySnippet(m: Message): string {
  if (m.reply_to?.system_type) return m.reply_to.system_type;
  if (m.text) return m.text;
  if (m.attachment_type === "image") return "Фото";
  if (m.attachment_name) return m.attachment_name;
  return "Сообщение";
}

export default function MessageInput({ onSend, replyTo, onCancelReply }: Props) {
  const [text, setText] = useState("");
  const [uploading, setUploading] = useState(false);
  const [showEmoji, setShowEmoji] = useState(false);
  const fileRef = useRef<HTMLInputElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  function handleSend() {
    if (!text.trim()) return;
    onSend(text.trim(), undefined, replyTo?.id);
    setText("");
    onCancelReply?.();
  }

  async function handleFile(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    setUploading(true);
    try {
      const data = await uploadFile(file);
      onSend("", {
        url: data.url,
        name: data.name,
        size: data.size,
        type: data.type,
      }, replyTo?.id);
      onCancelReply?.();
    } catch {
      alert("Ошибка загрузки файла");
    }
    setUploading(false);
    if (fileRef.current) fileRef.current.value = "";
  }

  function pickEmoji(emoji: string) {
    setText((prev) => prev + emoji);
    inputRef.current?.focus();
  }

  return (
    <div className="input-wrap">
      {replyTo && (
        <div className="reply-preview-bar">
          <div className="reply-preview-accent" />
          <div className="reply-preview-content">
            <div className="reply-preview-name">{replyTo.sender?.username ?? "Пользователь"}</div>
            <div className="reply-preview-text">{replySnippet(replyTo)}</div>
          </div>
          <button className="reply-preview-close" onClick={() => onCancelReply?.()} title="Отменить ответ">
            <X size={16} />
          </button>
        </div>
      )}
      <div className="input">
        <button className="input-btn" onClick={() => setShowEmoji(!showEmoji)} title="Эмодзи" disabled={uploading}>
          <FaceSlightlySmiling size={20} />
        </button>
        {showEmoji && (
          <div className="emoji-picker-input">
            {emojis.map((e) => (
              <span key={e} onClick={() => { pickEmoji(e); setShowEmoji(false); }}>
                {e}
              </span>
            ))}
          </div>
        )}
        <input
          ref={inputRef}
          type="text"
          value={text}
          onChange={(e) => setText(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter" && !e.shiftKey) handleSend();
            if (e.key === "Escape") { setText(""); onCancelReply?.(); }
          }}
          placeholder={uploading ? "Загрузка..." : replyTo ? "Ответить..." : "Сообщение..."}
          disabled={uploading}
        />
        <label className="input-btn file-btn" title="Прикрепить файл" style={uploading ? { pointerEvents: "none", opacity: 0.5 } : undefined}>
          <Paperclip size={20} />
          <input ref={fileRef} type="file" className="file-input-hidden" onChange={handleFile} />
        </label>
        <button className="input-btn send" onClick={handleSend} disabled={uploading || !text.trim()}>
          <Send size={20} />
        </button>
      </div>
    </div>
  );
}