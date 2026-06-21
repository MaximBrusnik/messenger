const emojis = [
  "👍", "❤️", "🔥", "😂", "😮", "😢", "🙏",
  "🎉", "👏", "💯", "🥰", "😍", "🤣", "😭",
  "😡", "🤔", "👀", "💪", "🤝", "✨", "⭐",
];

interface Props {
  onSelect: (emoji: string) => void;
  onClose: () => void;
}

export default function ReactionPicker({ onSelect, onClose }: Props) {
  return (
    <div className="reaction-picker" onMouseLeave={onClose}>
      {emojis.map((e) => (
        <span
          key={e}
          onClick={() => { onSelect(e); onClose(); }}
        >
          {e}
        </span>
      ))}
    </div>
  );
}
