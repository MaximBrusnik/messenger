import { useEffect } from "react";
import { X } from "lucide-react";

interface Props {
  url: string;
  name?: string;
  onClose: () => void;
}

export default function ImageModal({ url, name, onClose }: Props) {
  useEffect(() => {
    function handleKey(e: KeyboardEvent) {
      if (e.key === "Escape") onClose();
    }
    document.addEventListener("keydown", handleKey);
    return () => document.removeEventListener("keydown", handleKey);
  }, [onClose]);

  return (
    <div className="image-modal-overlay" onClick={(e) => e.target === e.currentTarget && onClose()}>
      <button className="image-modal-close" onClick={onClose}><X size={24} /></button>
      <img className="image-modal-img" src={url} alt={name || ""} />
      {name && <div className="image-modal-name">{name}</div>}
    </div>
  );
}
