import { useEffect, useState } from "react";
import { CircleCheck, CircleX, LoaderCircle } from "lucide-react";
import { apiRequest } from "../api/client";

export default function VerifyEmail() {
  const [status, setStatus] = useState<"loading" | "success" | "error">("loading");
  const [message, setMessage] = useState("");

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const token = params.get("token");
    if (!token) {
      setStatus("error");
      setMessage("Токен не указан");
      return;
    }
    apiRequest<{ message: string; token?: string }>(`/auth/verify-email?token=${token}`)
      .then((res) => {
        if (res.token) {
          localStorage.setItem("token", res.token);
          window.location.href = "/";
          return;
        }
        setStatus("success");
        setMessage(res.message);
      })
      .catch((e) => {
        setStatus("error");
        setMessage(e instanceof Error ? e.message : "Ошибка верификации");
      });
  }, []);

  return (
    <div className="auth">
      <div className="auth-card" style={{ textAlign: "center" }}>
        {status === "loading" && (
          <>
            <div className="auth-status-icon info"><LoaderCircle size={34} className="spin" /></div>
            <div>Подтверждение email...</div>
          </>
        )}
        {status === "success" && (
          <>
            <div className="auth-status-icon success"><CircleCheck size={34} /></div>
            <div style={{ color: "#43a047", fontWeight: 500, fontSize: 16 }}>{message}</div>
            <a
              href="/"
              style={{
                display: "block", marginTop: 16, color: "var(--accent)",
                textDecoration: "none", fontWeight: 500,
              }}
            >
              Вернуться в приложение
            </a>
          </>
        )}
        {status === "error" && (
          <>
            <div className="auth-status-icon error"><CircleX size={34} /></div>
            <div style={{ color: "#e53935", fontWeight: 500, fontSize: 16 }}>{message}</div>
            <a
              href="/"
              style={{
                display: "block", marginTop: 16, color: "var(--accent)",
                textDecoration: "none", fontWeight: 500,
              }}
            >
              Вернуться в приложение
            </a>
          </>
        )}
      </div>
    </div>
  );
}
