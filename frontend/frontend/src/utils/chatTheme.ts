const STORAGE_KEY = "mm_chat_appearance";

export interface ChatAppearance {
  color?: string;
  bg?: string;
}

export const DEFAULT_ACCENT = "#2a9efb";

export const CHAT_COLORS = [
  "#2a9efb",
  "#7c5cff",
  "#4f46e5",
  "#db2777",
  "#e11d48",
  "#ea580c",
  "#f59e0b",
  "#16a34a",
  "#0d9488",
  "#9333ea",
  "#ef4444",
  "#64748b",
];

export const CHAT_WALLPAPERS = [
  { id: "default", name: "Классика" },
  { id: "sunset", name: "Закат" },
  { id: "ocean", name: "Океан" },
  { id: "aurora", name: "Аврора" },
  { id: "forest", name: "Лес" },
  { id: "midnight", name: "Полночь" },
  { id: "peach", name: "Персик" },
];

export function loadChatAppearance(): Record<number, ChatAppearance> {
  try {
    return JSON.parse(localStorage.getItem(STORAGE_KEY) || "{}");
  } catch {
    return {};
  }
}

export function saveChatAppearance(map: Record<number, ChatAppearance>) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(map));
}
