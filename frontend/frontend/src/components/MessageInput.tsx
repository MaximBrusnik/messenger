import { useState, useRef } from "react";
import { uploadFile } from "../api/client";

interface Props {
  onSend: (text: string, attachment?: { url: string; name: string; size: number; type: string }) => void;
}

export default function MessageInput({ onSend }: Props) {
  const [text, setText] = useState("");
  const [uploading, setUploading] = useState(false);
  const fileRef = useRef<HTMLInputElement>(null);

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

  return (
    <div className="input">
      <input
        type="text"
        value={text}
        onChange={(e) => setText(e.target.value)}
        onKeyDown={(e) => e.key === "Enter" && !e.shiftKey && handleSend()}
        placeholder={uploading ? "Загрузка..." : "Сообщение..."}
        disabled={uploading}
      />
      <input ref={fileRef} type="file" style={{ display: "none" }} onChange={handleFile} />
      <button className="input-btn" onClick={() => fileRef.current?.click()} title="Прикрепить файл" disabled={uploading}>
        📎
      </button>
      <button className="input-btn send" onClick={handleSend} disabled={uploading || !text.trim()}>
        ➤
      </button>
    </div>
  );
}
