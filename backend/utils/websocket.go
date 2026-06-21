package utils

//
//import (
//	"MessangerMax/internal/entity"
//	"MessangerMax/internal/handlers"
//	"encoding/json"
//	"log"
//)
//
//// Отправить новое сообщение участникам чата
//func SendNewMessage(chatID uint, message *entity.Message) {
//	// В реальном приложении здесь нужно получить ID участников чата из БД
//	// Для примера отправляем всем подключенным
//
//	wsMessage := entity.WSMessage{
//		Type: entity.WSMessageNewMessage,
//		Payload: map[string]interface{}{
//			"message": message,
//			"chat_id": chatID,
//		},
//	}
//
//	// Отправляем всем подключенным клиентам
//	// В реальности - только участникам чата
//	handlers.Broadcast(wsMessage)
//}
//
//// Обновить информацию о чате
//func SendChatUpdated(chat *entity.Chat) {
//	wsMessage := entity.WSMessage{
//		Type: entity.WSMessageChatUpdated,
//		Payload: map[string]interface{}{
//			"chat": chat,
//		},
//	}
//
//	// Получаем ID участников чата
//	var userIDs []uint
//	for _, participant := range chat.Participants {
//		userIDs = append(userIDs, participant.ID)
//	}
//
//	// Отправляем только участникам
//	handlers.SendToUsers(userIDs, wsMessage)
//}
//
//// Отправить статус пользователя
//func SendUserStatus(userID uint, status string) {
//	wsMessage := entity.WSMessage{
//		Type: entity.WSMessageUserStatus,
//		Payload: map[string]interface{}{
//			"user_id": userID,
//			"status":  status,
//		},
//	}
//
//	// В реальности нужно отправить всем, у кого пользователь в контактах
//	handlers.Broadcast(wsMessage)
//}
//
//// Вспомогательная функция для отправки JSON
//func SendJSON(userID uint, data interface{}) error {
//	jsonData, err := json.Marshal(data)
//	if err != nil {
//		return err
//	}
//
//	log.Printf("Sending to user %d: %s", userID, string(jsonData))
//	return handlers.SendToUser(userID, data)
//}
