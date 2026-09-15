import { useEffect, useState } from "react";
import { apiRequest } from "./api/client";
import type { User } from "./types";
import { AuthProvider } from "./context/AuthContext";
import { CallProvider } from "./context/CallContext";
import { usePushNotifications } from "./hooks/usePushNotifications";
import AuthPage from "./components/AuthPage";
import ChatApp from "./components/ChatApp";
import VerifyEmail from "./components/VerifyEmail";

export default function App() {
  const [user, setUser] = useState<User | null>(null);
  const [route, setRoute] = useState(window.location.pathname + window.location.search);

  usePushNotifications(user);

  useEffect(() => {
    apiRequest<{ data: User }>("/auth/profile")
      .then((res) => setUser(res.data))
      .catch(() => setUser(null));
  }, []);

  useEffect(() => {
    const handlePop = () => setRoute(window.location.pathname + window.location.search);
    window.addEventListener("popstate", handlePop);
    return () => window.removeEventListener("popstate", handlePop);
  }, []);

  if (route.startsWith("/verify-email")) {
    return <VerifyEmail />;
  }

  return (
    <AuthProvider value={{ user, setUser }}>
      {user ? (
        <CallProvider>
          <ChatApp />
        </CallProvider>
      ) : (
        <AuthPage />
      )}
    </AuthProvider>
  );
}
