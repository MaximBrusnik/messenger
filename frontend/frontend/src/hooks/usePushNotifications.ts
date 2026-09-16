import { useEffect, useRef } from "react";
import { PushNotifications } from "@capacitor/push-notifications";
import { apiRequest } from "../api/client";
import type { User } from "../types";

export function usePushNotifications(user: User | null) {
  const registeredRef = useRef(false);

  useEffect(() => {
    if (!user || registeredRef.current) return;
    registeredRef.current = true;

    let removeRegistration: (() => void) | null = null;
    let removeAction: (() => void) | null = null;
    let cancelled = false;

    PushNotifications.addListener("registration", (token) => {
      const platform = /android/i.test(navigator.userAgent) ? "android" : "ios";
      apiRequest("/devices/register", "POST", {
        token: token.value,
        platform,
      }).catch(() => {
        // ignore — регистрация повторится при следующем запуске
      });
    }).then((handle) => { removeRegistration = handle.remove; });

    PushNotifications.addListener("pushNotificationActionPerformed", (notification) => {
      const chatId = notification.notification.data?.chat_id;
      if (chatId && typeof chatId === "string") {
        const path = window.location.pathname;
        if (!path.startsWith("/chat/")) {
          window.history.pushState({}, "", `/chat/${chatId}`);
          window.dispatchEvent(new PopStateEvent("popstate"));
        }
      }
    }).then((handle) => { removeAction = handle.remove; });

    PushNotifications.requestPermissions().then((result) => {
      if (cancelled) return;
      if (result.receive !== "granted") return;
      PushNotifications.register();
    });

    return () => {
      cancelled = true;
      removeRegistration?.();
      removeAction?.();
    };
  }, [user]);
}
