import { useState } from "react";
import { MessageSquare } from "lucide-react";
import { apiRequest } from "../api/client";
import { useAuth } from "../context/AuthContext";
import VerificationSent from "./VerificationSent";

export default function AuthPage() {
  const { setUser } = useAuth();
  const [mode, setMode] = useState<"login" | "register" | "verification_sent">("login");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [email, setEmail] = useState("");
  const [error, setError] = useState<string | null>(null);

  async function submit() {
    try {
      setError(null);

      if (mode === "register" && password !== confirmPassword) {
        setError("Пароли не совпадают");
        return;
      }

      const payload =
        mode === "login"
          ? { username, password }
          : { username, password, email };

      const data = await apiRequest<{ token?: string; requires_email_verification?: boolean }>(
        `/auth/${mode}`,
        "POST",
        payload
      );

      if (data.requires_email_verification) {
        setMode("verification_sent");
        return;
      }

      localStorage.setItem("token", data.token!);

      const me = await apiRequest<{ data: import("../types").User }>("/auth/profile");
      setUser(me.data);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Ошибка");
    }
  }

  function switchMode(m: "login" | "register") {
    setMode(m);
    setError(null);
  }

  if (mode === "verification_sent") {
    return <VerificationSent email={email} onBack={() => switchMode("login")} />;
  }

  return (
    <div className="auth">
      <div className="auth-card">
        <div className="auth-logo">
          <div className="auth-logo-badge"><MessageSquare size={26} /></div>
        </div>
        <h2>{mode === "login" ? "Вход" : "Регистрация"}</h2>

        <input
          placeholder="Имя пользователя"
          value={username}
          onChange={(e) => setUsername(e.target.value)}
          onKeyDown={(e) => e.key === "Enter" && submit()}
        />

        {mode === "register" && (
          <input
            placeholder="Email"
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && submit()}
          />
        )}

        <input
          placeholder="Пароль"
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          onKeyDown={(e) => e.key === "Enter" && submit()}
        />

        {mode === "register" && (
          <input
            placeholder="Подтвердите пароль"
            type="password"
            value={confirmPassword}
            onChange={(e) => setConfirmPassword(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && submit()}
          />
        )}

        {error && <div className="auth-error">{error}</div>}

        <button onClick={submit}>
          {mode === "login" ? "Войти" : "Создать аккаунт"}
        </button>

        <div
          className="auth-switch"
          onClick={() => switchMode(mode === "login" ? "register" : "login")}
        >
          {mode === "login" ? "Нет аккаунта? Зарегистрироваться" : "Уже есть аккаунт? Войти"}
        </div>
      </div>
    </div>
  );
}
