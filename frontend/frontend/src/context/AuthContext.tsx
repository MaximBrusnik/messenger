import { createContext, useContext, type ReactNode } from "react";
import type { User } from "../types";

interface AuthCtx {
  user: User | null;
  setUser: (u: User | null) => void;
}

export const AuthContext = createContext<AuthCtx>({ user: null, setUser: () => {} });

export const useAuth = () => useContext(AuthContext);

export function AuthProvider({ children, value }: { children: ReactNode; value: AuthCtx }) {
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}
