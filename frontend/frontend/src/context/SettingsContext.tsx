import { createContext, useCallback, useContext, useEffect, useState, type ReactNode } from "react";
import { apiRequest } from "../api/client";
import type { UserSettings } from "../types";

interface SettingsCtx {
  settings: UserSettings | null;
  setSoundEnabled: (v: boolean) => void;
  refresh: () => void;
}

export const SettingsContext = createContext<SettingsCtx>({
  settings: null,
  setSoundEnabled: () => {},
  refresh: () => {},
});

export const useSettings = () => useContext(SettingsContext);

export function SettingsProvider({ children }: { children: ReactNode }) {
  const [settings, setSettings] = useState<UserSettings | null>(null);

  const refresh = useCallback(() => {
    apiRequest<{ data: UserSettings }>("/auth/settings")
      .then((res) => setSettings(res.data))
      .catch(() => {});
  }, []);

  useEffect(() => {
    refresh();
  }, [refresh]);

  const setSoundEnabled = useCallback((v: boolean) => {
    setSettings((s) => (s ? { ...s, sound_enabled: v } : s));
    apiRequest("/auth/settings", "PUT", { sound_enabled: v }).catch(() => {});
  }, []);

  return (
    <SettingsContext.Provider value={{ settings, setSoundEnabled, refresh }}>
      {children}
    </SettingsContext.Provider>
  );
}