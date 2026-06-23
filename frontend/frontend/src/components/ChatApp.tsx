import { useEffect, useState, useCallback } from "react";
import { apiRequest, deleteChat, getMusic } from "../api/client";
import type { Chat, MusicTrack } from "../types";
import { useAuth } from "../context/AuthContext";
import Sidebar from "./Sidebar";
import ChatArea from "./ChatArea";
import MusicPlayer from "./MusicPlayer";
import ProfileModal from "./ProfileModal";
import UserProfileModal from "./UserProfileModal";


export default function ChatApp() {
  const { user, setUser } = useAuth();
  const [chats, setChats] = useState<Chat[]>([]);
  const [activeChat, setActiveChat] = useState<Chat | null>(null);
  const [showProfile, setShowProfile] = useState(false);
  const [profileUserId, setProfileUserId] = useState<number | null>(null);
  const [activeTab, setActiveTab] = useState<"chats" | "music">("chats");
  const [selectedTrack, setSelectedTrack] = useState<MusicTrack | null>(null);

  const isMobile = window.innerWidth <= 768;
  const [mobileChat, setMobileChat] = useState(false);
  const [mobilePlayer, setMobilePlayer] = useState(false);

  // Request notification permission
  useEffect(() => {
    if ("Notification" in window && Notification.permission === "default") {
      Notification.requestPermission();
    }
  }, []);

  // Sync mobileChat with activeChat
  useEffect(() => {
    if (!isMobile) { setMobileChat(false); setMobilePlayer(false); }
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
    setActiveTab("chats");
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

  async function handleStartAIChat() {
    const res = await apiRequest<{ data: Chat }>("/ai/chat");
    if (res?.data) {
      setActiveChat(res.data);
      setActiveTab("chats");
      if (isMobile) setMobileChat(true);
    }
  }

  async function handlePlayTrack(id: number) {
    try {
      const res = await getMusic();
      const track = res?.data?.find((t: MusicTrack) => t.id === id);
      if (track) {
        setSelectedTrack(track);
        if (isMobile) setMobilePlayer(true);
      }
    } catch { /* ignore */ }
  }

  function handleMusicBack() {
    setMobilePlayer(false);
    setSelectedTrack(null);
  }

  function handleTabChange(tab: "chats" | "music") {
    setActiveTab(tab);
    if (tab !== "music") setSelectedTrack(null);
    if (isMobile && tab === "chats") {
      setMobileChat(false);
      setMobilePlayer(false);
    }
  }

  const handleUserStatus = useCallback((userId: number, status: string) => {
    if (userId === user?.id && user) {
      setUser({ ...user, status });
    }
  }, [user, setUser]);

  const mobileClass = isMobile
    ? `mobile-view${mobileChat || (activeTab === "music" && mobilePlayer) ? " show-chat" : ""}`
    : "";

  return (
    <div className={`app ${mobileClass}`}>
      <Sidebar
          user={user!}
          chats={chats}
          activeChat={activeChat}
          activeTab={activeTab}
          activeTrackId={selectedTrack?.id ?? null}
          onSelectChat={openChat}
          onOpenUserProfile={handleOpenUserProfile}
          onOpenProfile={() => setShowProfile(true)}
          onDeleteChat={handleDeleteChat}
          onStartAIChat={handleStartAIChat}
          onTabChange={handleTabChange}
          onPlayTrack={handlePlayTrack}
        />

      <div className="chat">
        {activeTab === "music" && selectedTrack ? (
          <MusicPlayer
            track={selectedTrack}
            onBack={isMobile ? handleMusicBack : undefined}
          />
        ) : activeChat ? (
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
