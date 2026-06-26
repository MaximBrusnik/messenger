import { useEffect, useRef } from "react";
import { PushNotifications } from "@capacitor/push-notifications";
import { apiRequest } from "../api/client";
import type { User } from "../types";

export function usePushNotifications(user: User | null) {
  const registeredRef = useRef(false);

  useEffect(() => {
    if (!user || registeredRef.current) return;
    registeredRef.current = true;

    PushNotifications.requestPermissions().then((result) => {
      if (result.receive !== "granted") return;
      PushNotifications.register();

      PushNotifications.addListener("registration", (token) => {
        const platform = /android/i.test(navigator.userAgent) ? "android" : "ios";
        apiRequest("/devices/register", "POST", {
          token: token.value,
          platform,
        }).catch(() => {
          // ignore — регистрация повторится при следующем запуске
        });
      });

      PushNotifications.addListener("pushNotificationActionPerformed", (notification) => {
        const chatId = notification.notification.data?.chat_id;
        if (chatId && typeof chatId === "string") {
          const path = window.location.pathname;
          if (!path.startsWith("/chat/")) {
            window.history.pushState({}, "", `/chat/${chatId}`);
            window.dispatchEvent(new PopStateEvent("popstate"));
          }
        }
      });
    });
  }, [user]);
}
