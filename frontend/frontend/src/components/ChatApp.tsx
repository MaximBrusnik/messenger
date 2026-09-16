import { useEffect, useState, useCallback, useRef } from "react";
import { App } from "@capacitor/app";
import { Capacitor } from "@capacitor/core";
import { MessagesSquare } from "lucide-react";
import { apiRequest, archiveChat as apiArchiveChat, deleteChat, getArchivedChats, getFavoritesChat, getMusic, unarchiveChat as apiUnarchiveChat } from "../api/client";
import type { Chat, MusicTrack, User } from "../types";
import { useAuth } from "../context/AuthContext";
import { useCall } from "../context/CallContext";
import { useGlobalWebSocket, type ChatLiveHandlers } from "../hooks/useGlobalWebSocket";
import { loadChatAppearance, saveChatAppearance, type ChatAppearance } from "../utils/chatTheme";
import Sidebar from "./Sidebar";
import ChatArea from "./ChatArea";
import MusicPlayer from "./MusicPlayer";
import ProfileModal from "./ProfileModal";
import UserProfileModal from "./UserProfileModal";


export default function ChatApp() {
  const { user, setUser } = useAuth();
  const { phase: callPhase, rejectCall, endCall } = useCall();
  const [chats, setChats] = useState<Chat[]>([]);
  const [archivedChats, setArchivedChats] = useState<Chat[]>([]);
  const [activeChat, setActiveChat] = useState<Chat | null>(null);
  const [showProfile, setShowProfile] = useState(false);
  const [profileUserId, setProfileUserId] = useState<number | null>(null);
  const [activeTab, setActiveTab] = useState<"chats" | "music" | "archived">("chats");
  const [selectedTrack, setSelectedTrack] = useState<MusicTrack | null>(null);
  const [chatAppearance, setChatAppearance] = useState<Record<number, ChatAppearance>>(loadChatAppearance);

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

  // Handle the Android system back button inside the SPA instead of leaving the app.
  const backState = useRef({ mobileChat, mobilePlayer, activeTab, showProfile, profileUserId, selectedTrack });
  backState.current = { mobileChat, mobilePlayer, activeTab, showProfile, profileUserId, selectedTrack };
  useEffect(() => {
    if (!Capacitor.isNativePlatform()) return;
    const listener = App.addListener("backButton", () => {
      const s = backState.current;
      if (s.showProfile) { setShowProfile(false); return; }
      if (s.profileUserId !== null) { setProfileUserId(null); return; }
      if (s.mobilePlayer) { handleMusicBack(); return; }
      if (s.mobileChat) { setMobileChat(false); return; }
      if (callPhase === "incoming") { rejectCall(); return; }
      if (callPhase === "outgoing" || callPhase === "active") { endCall(); return; }
      void App.exitApp();
    });
    return () => { void listener.then((l) => l.remove()); };
  }, [callPhase, rejectCall, endCall]);

  const loadChats = useCallback(async () => {
    const res = await apiRequest<{ data: Chat[] }>("/chats");
    setChats(res?.data || []);
  }, []);

  const loadArchivedChats = useCallback(async () => {
    const res = await getArchivedChats();
    setArchivedChats(res?.data || []);
  }, []);

  useEffect(() => {
    loadChats();
    loadArchivedChats();
  }, [loadChats, loadArchivedChats]);

  async function openChat(chat: Chat) {
    setActiveChat(chat);
    setActiveTab(chat.is_archived ? "archived" : "chats");
    if (isMobile) setMobileChat(true);
    await apiRequest(`/chats/${chat.id}/read`, "POST");
  }

  function handleChatCreated(chat: Chat) {
    setChats((prev) => {
      if (prev.some((c) => c.id === chat.id)) return prev;
      return [chat, ...prev];
    });
    if (chat.is_archived) {
      setArchivedChats((prev) => {
        if (prev.some((c) => c.id === chat.id)) return prev;
        return [chat, ...prev];
      });
    }
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
    setArchivedChats((prev) => prev.filter((c) => c.id !== chatId));
    if (activeChat?.id === chatId) setActiveChat(null);
  }

  async function handleArchiveChat(chatId: number) {
    const chat = chats.find((c) => c.id === chatId);
    if (!chat) return;
    await apiArchiveChat(chatId);
    setChats((prev) => prev.filter((c) => c.id !== chatId));
    setArchivedChats((prev) => {
      if (prev.some((c) => c.id === chatId)) return prev;
      return [{ ...chat, is_archived: true }, ...prev];
    });
    if (activeChat?.id === chatId) setActiveChat(null);
  }

  async function handleUnarchiveChat(chatId: number) {
    const chat = archivedChats.find((c) => c.id === chatId);
    if (!chat) return;
    await apiUnarchiveChat(chatId);
    setArchivedChats((prev) => prev.filter((c) => c.id !== chatId));
    setChats((prev) => {
      if (prev.some((c) => c.id === chatId)) return prev;
      return [{ ...chat, is_archived: false }, ...prev];
    });
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

  async function handleStartFavoritesChat() {
    const res = await getFavoritesChat();
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

  function handleTabChange(tab: "chats" | "music" | "archived") {
    setActiveTab(tab);
    if (tab !== "music") setSelectedTrack(null);
    if (isMobile && tab === "chats") {
      setMobileChat(false);
      setMobilePlayer(false);
    }
  }

  function handleAppearanceChange(chatId: number, patch: ChatAppearance) {
    setChatAppearance((prev) => {
      const next = { ...prev, [chatId]: { ...prev[chatId], ...patch } };
      saveChatAppearance(next);
      return next;
    });
  }

  const handleUserStatus = useCallback((userId: number, status: string) => {
    if (userId === user?.id && user) {
      setUser({ ...user, status });
    }
  }, [user, setUser]);

  const liveHandlersRef = useRef<Record<number, ChatLiveHandlers>>({});

  const registerLiveHandlers = useCallback((chatId: number, handlers: ChatLiveHandlers) => {
    liveHandlersRef.current[chatId] = handlers;
  }, []);

  const unregisterLiveHandlers = useCallback((chatId: number) => {
    delete liveHandlersRef.current[chatId];
  }, []);

  const routeToChat = useCallback((chatId: number, fn: (handlers: ChatLiveHandlers) => void) => {
    const handlers = liveHandlersRef.current[chatId];
    if (handlers) fn(handlers);
  }, []);

  // The WebSocket is opened once at the app level so the user is considered
  // online from the moment the app loads (even with no chat open). After the
  // socket connects, refresh the profile so the status flips to online.
  useGlobalWebSocket({
    onConnected: async () => {
      try {
        const res = await apiRequest<{ data: User }>("/auth/profile");
        if (res?.data) setUser(res.data);
      } catch { /* keep current profile */ }
    },
    onAnyMessage: () => {
      loadChats();
      loadArchivedChats();
    },
    onUserStatus: handleUserStatus,
    onChatDeleted: (chatId) => {
      setChats((prev) => prev.filter((c) => c.id !== chatId));
      setArchivedChats((prev) => prev.filter((c) => c.id !== chatId));
      if (activeChat?.id === chatId) setActiveChat(null);
    },
    route: routeToChat,
  });

  const mobileClass = isMobile
    ? `mobile-view${mobileChat || (activeTab === "music" && mobilePlayer) ? " show-chat" : ""}`
    : "";

  return (
    <div className={`app ${mobileClass}`}>
      <Sidebar
          user={user!}
          chats={chats}
          archivedChats={archivedChats}
          activeChat={activeChat}
          activeTab={activeTab}
          isMobile={isMobile}
          activeTrackId={selectedTrack?.id ?? null}
          onSelectChat={openChat}
          onOpenUserProfile={handleOpenUserProfile}
          onOpenProfile={() => setShowProfile(true)}
          onDeleteChat={handleDeleteChat}
          onArchiveChat={handleArchiveChat}
          onUnarchiveChat={handleUnarchiveChat}
          onStartAIChat={handleStartAIChat}
          onStartFavoritesChat={handleStartFavoritesChat}
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
          <ChatArea chat={activeChat} onBack={isMobile ? handleBack : undefined} onMessage={loadChats} onOpenUserProfile={handleOpenUserProfile} onDeleteChat={handleDeleteChat} onArchiveChat={handleArchiveChat} onUnarchiveChat={handleUnarchiveChat} appearance={chatAppearance[activeChat.id]} onAppearanceChange={(patch) => handleAppearanceChange(activeChat.id, patch)} registerLiveHandlers={registerLiveHandlers} unregisterLiveHandlers={unregisterLiveHandlers} />
        ) : (
          <div className="empty-state">
            <div className="empty-icon"><MessagesSquare size={30} /></div>
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