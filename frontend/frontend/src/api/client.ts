import type { MusicTrack, User } from "../types";

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

export async function getMusic() {
  return apiRequest<{ data: MusicTrack[] }>("/music");
}

export async function uploadMusic(file: File) {
  const token = localStorage.getItem("token");
  const form = new FormData();
  form.append("file", file);

  const res = await fetch(`${API_URL}/music/upload`, {
    method: "POST",
    headers: token ? { Authorization: `Bearer ${token}` } : {},
    body: form,
  });

  if (!res.ok) throw new Error(await res.text());
  return res.json();
}

export async function deleteMusic(id: number) {
  return apiRequest(`/music/${id}`, "DELETE");
}

export function getMusicStreamUrl(id: number) {
  return `${API_URL}/music/${id}/stream`;
}

export function getMusicDownloadUrl(id: number) {
  return `${API_URL}/music/${id}/download`;
}

export async function getPendingMusic() {
  return apiRequest<{ data: MusicTrack[] }>("/admin/music/pending");
}

export async function approveMusic(id: number) {
  return apiRequest(`/admin/music/${id}/approve`, "PUT");
}

export async function rejectMusic(id: number) {
  return apiRequest(`/admin/music/${id}/reject`, "PUT");
}

export async function pinMessage(chatId: number, msgId: number) {
  return apiRequest(`/chats/${chatId}/pin/${msgId}`, "PUT");
}

export async function unpinMessage(chatId: number) {
  return apiRequest(`/chats/${chatId}/pin`, "DELETE");
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
