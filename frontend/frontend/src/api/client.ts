import type { User } from "../types";

const API_URL = "/api/v1";

export async function apiRequest<T = unknown>(
  path: string,
  method = "GET",
  body?: unknown
): Promise<T> {
  const token = localStorage.getItem("token");

  const res = await fetch(`${API_URL}${path}`, {
    method,
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: body ? JSON.stringify(body) : undefined,
  });

  if (!res.ok) {
    const text = await res.text();
    throw new Error(text);
  }

  return res.json();
}

export async function getUserProfile(id: number) {
  return apiRequest<{ data: User }>(`/users/${id}`);
}

export async function deleteChat(chatId: number) {
  return apiRequest(`/chats/${chatId}`, "DELETE");
}

export async function deleteMessage(chatId: number, msgId: number) {
  return apiRequest(`/chats/${chatId}/messages/${msgId}`, "DELETE");
}

export async function uploadFile(file: File) {
  const token = localStorage.getItem("token");
  const form = new FormData();
  form.append("file", file);

  const res = await fetch(`${API_URL}/upload`, {
    method: "POST",
    headers: token ? { Authorization: `Bearer ${token}` } : {},
    body: form,
  });

  if (!res.ok) throw new Error(await res.text());
  return res.json();
}
