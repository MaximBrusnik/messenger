import { useEffect, useState, useCallback } from "react";
import { apiRequest, deleteChat } from "../api/client";
import type { Chat } from "../types";
import { useAuth } from "../context/AuthContext";
import Sidebar from "./Sidebar";
import ChatArea from "./ChatArea";
import ProfileModal from "./ProfileModal";
import UserProfileModal from "./UserProfileModal";

export default function ChatApp() {
  const { user, setUser } = useAuth();
  const [chats, setChats] = useState<Chat[]>([]);
  const [activeChat, setActiveChat] = useState<Chat | null>(null);
  const [showProfile, setShowProfile] = useState(false);
  const [profileUserId, setProfileUserId] = useState<number | null>(null);

  const isMobile = window.innerWidth <= 768;
  const [mobileChat, setMobileChat] = useState(false);

  // Request notification permission
  useEffect(() => {
    if ("Notification" in window && Notification.permission === "default") {
      Notification.requestPermission();
    }
  }, []);

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

  async function handleDeleteChat(chatId: number) {
    if (!confirm("Удалить чат?")) return;
    await deleteChat(chatId);
    setChats((prev) => prev.filter((c) => c.id !== chatId));
    if (activeChat?.id === chatId) setActiveChat(null);
  }

  function handleOpenUserProfile(userId: number) {
    setProfileUserId(userId);
  }

  async function handleStartChatFromProfile(targetId: number) {
    const res = await apiRequest<{ data: Chat }>("/chats", "POST", { user_id: targetId });
    handleChatCreated(res.data);
    setProfileUserId(null);
  }

  const handleUserStatus = useCallback((userId: number, status: string) => {
    if (userId === user?.id && user) {
      setUser({ ...user, status });
    }
  }, [user, setUser]);

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
        onOpenUserProfile={handleOpenUserProfile}
        onOpenProfile={() => setShowProfile(true)}
        onDeleteChat={handleDeleteChat}
      />

      <div className="chat">
        {activeChat ? (
          <ChatArea chat={activeChat} onBack={isMobile ? handleBack : undefined} onMessage={loadChats} onUserStatus={handleUserStatus} onOpenUserProfile={handleOpenUserProfile} onDeleteChat={handleDeleteChat} />
        ) : (
          <div className="empty-state">
            <div className="empty-icon">💬</div>
            <div>Выберите чат, чтобы начать общение</div>
          </div>
        )}
      </div>

      {showProfile && <ProfileModal onClose={() => setShowProfile(false)} />}
      {profileUserId !== null && (
        <UserProfileModal
          userId={profileUserId}
          onClose={() => setProfileUserId(null)}
          onStartChat={handleStartChatFromProfile}
        />
      )}
    </div>
  );
}
