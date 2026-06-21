import { useState } from "react";
import { apiRequest } from "../api/client";
import { useAuth } from "../context/AuthContext";

export default function AuthPage() {
  const { setUser } = useAuth();
  const [mode, setMode] = useState<"login" | "register">("login");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [email, setEmail] = useState("");
  const [error, setError] = useState<string | null>(null);

  async function submit() {
    try {
      setError(null);

      const payload =
        mode === "login"
          ? { username, password }
          : { username, password, email };

      const data = await apiRequest<{ token: string }>(
        `/auth/${mode}`,
        "POST",
        payload
      );

      localStorage.setItem("token", data.token);

      const me = await apiRequest<{ data: import("../types").User }>("/auth/profile");
      setUser(me.data);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Ошибка");
    }
  }

  return (
    <div className="auth">
      <div className="auth-card">
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

        {error && <div className="auth-error">{error}</div>}

        <button onClick={submit}>
          {mode === "login" ? "Войти" : "Создать аккаунт"}
        </button>

        <div
          className="auth-switch"
          onClick={() => setMode(mode === "login" ? "register" : "login")}
        >
          {mode === "login" ? "Нет аккаунта? Зарегистрироваться" : "Уже есть аккаунт? Войти"}
        </div>
      </div>
    </div>
  );
}
