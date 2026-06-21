import { useEffect, useState, useCallback } from "react";
import { apiRequest } from "../api/client";
import type { Chat } from "../types";
import { useAuth } from "../context/AuthContext";
import Sidebar from "./Sidebar";
import ChatArea from "./ChatArea";
import ProfileModal from "./ProfileModal";

export default function ChatApp() {
  const { user, setUser } = useAuth();
  const [chats, setChats] = useState<Chat[]>([]);
  const [activeChat, setActiveChat] = useState<Chat | null>(null);
  const [showProfile, setShowProfile] = useState(false);

  const isMobile = window.innerWidth <= 768;
  const [mobileChat, setMobileChat] = useState(false);

  // Sync mobileChat with activeChat
  useEffect(() => {
    if (!isMobile) setMobileChat(false);
  }, [isMobile]);

  const loadChats = useCallback(async () => {
    const res = await apiRequest<{ data: Chat[] }>("/chats");
    setChats(res?.data || []);
  }, []);

  useEffect(() => {
    loadChats();
  }, [loadChats]);

  async function openChat(chat: Chat) {
    setActiveChat(chat);
    if (isMobile) setMobileChat(true);
    await apiRequest(`/chats/${chat.id}/read`, "POST");
  }

  function handleChatCreated(chat: Chat) {
    setChats((prev) => {
      if (prev.some((c) => c.id === chat.id)) return prev;
      return [chat, ...prev];
    });
    setActiveChat(chat);
    if (isMobile) setMobileChat(true);
  }

  function handleBack() {
    setMobileChat(false);
  }

  function logout() {
    localStorage.removeItem("token");
    setUser(null);
  }

  const mobileClass = isMobile
    ? `mobile-view${mobileChat ? " show-chat" : ""}`
    : "";

  return (
    <div className={`app ${mobileClass}`}>
      <Sidebar
        user={user!}
        chats={chats}
        activeChat={activeChat}
        onSelectChat={openChat}
        onLogout={logout}
        onChatCreated={handleChatCreated}
        onOpenProfile={() => setShowProfile(true)}
      />

      <div className="chat">
        {activeChat ? (
          <ChatArea chat={activeChat} onBack={isMobile ? handleBack : undefined} />
        ) : (
          <div className="empty-state">
            <div className="empty-icon">💬</div>
            <div>Выберите чат, чтобы начать общение</div>
          </div>
        )}
      </div>

      {showProfile && <ProfileModal onClose={() => setShowProfile(false)} />}
    </div>
  );
}
