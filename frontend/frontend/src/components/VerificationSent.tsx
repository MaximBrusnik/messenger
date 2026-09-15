import { Mail } from "lucide-react";

export default function VerificationSent({ email, onBack }: { email: string; onBack: () => void }) {
  return (
    <div className="auth">
      <div className="auth-card" style={{ textAlign: "center" }}>
        <div className="auth-status-icon info"><Mail size={34} /></div>
        <h2>Проверьте почту</h2>
        <p>
          Письмо со ссылкой для подтверждения отправлено на <strong>{email}</strong>
        </p>
        <p style={{ fontSize: 14, color: "#666" }}>
          Перейдите по ссылке в письме, чтобы активировать аккаунт
        </p>
        <button onClick={onBack} style={{ marginTop: 8 }}>
          На страницу входа
        </button>
      </div>
    </div>
  );
}
