import { useState, useRef } from "react";
import { uploadFile } from "../api/client";
import { FaceSlightlySmiling, Paperclip, Send } from "lucide-react";

const emojis = [
  "👍", "❤️", "🔥", "😂", "😮", "😢", "🙏",
  "🎉", "👏", "💯", "🥰", "😍", "🤣", "😭",
  "😡", "🤔", "👀", "💪", "🤝", "✨", "⭐",
];

interface Props {
  onSend: (text: string, attachment?: { url: string; name: string; size: number; type: string }) => void;
}

export default function MessageInput({ onSend }: Props) {
  const [text, setText] = useState("");
  const [uploading, setUploading] = useState(false);
  const [showEmoji, setShowEmoji] = useState(false);
  const fileRef = useRef<HTMLInputElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  async function handleSend() {
    if (!text.trim()) return;
    onSend(text.trim());
    setText("");
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
      });
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
        onKeyDown={(e) => e.key === "Enter" && !e.shiftKey && handleSend()}
        placeholder={uploading ? "Загрузка..." : "Сообщение..."}
        disabled={uploading}
      />
      <input ref={fileRef} type="file" style={{ display: "none" }} onChange={handleFile} />
      <button className="input-btn" onClick={() => fileRef.current?.click()} title="Прикрепить файл" disabled={uploading}>
        <Paperclip size={20} />
      </button>
      <button className="input-btn send" onClick={handleSend} disabled={uploading || !text.trim()}>
        <Send size={20} />
      </button>
    </div>
  );
}
