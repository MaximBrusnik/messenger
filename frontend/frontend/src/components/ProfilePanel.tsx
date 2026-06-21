import { useEffect, useState } from "react";
import { apiRequest } from "../api/client";
import type { User } from "../types";

export default function ProfilePanel() {
  const [profile, setProfile] = useState<User | null>(null);
  const [username, setUsername] = useState("");

  useEffect(() => {
    apiRequest<{ data: User }>("/auth/profile").then((res) => {
      setProfile(res.data);
      setUsername(res.data.username);
    });
  }, []);

  async function save() {
    await apiRequest("/auth/profile", "PUT", { username });
    setProfile((p) => (p ? { ...p, username } : p));
  }

  if (!profile) return null;

  return (
    <>
      <h3>Профиль</h3>
      <input value={username} onChange={(e) => setUsername(e.target.value)} />
      <button onClick={save}>Сохранить</button>
    </>
  );
}
