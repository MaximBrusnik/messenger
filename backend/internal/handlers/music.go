package handlers

import (
	logic2 "MessangerMax/internal/logic"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type MusicHandler struct {
	musicService logic2.MusicService
}

func NewMusicHandler(musicService logic2.MusicService) *MusicHandler {
	return &MusicHandler{musicService: musicService}
}

func (h *MusicHandler) List(c *gin.Context) {
	music, err := h.musicService.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения списка"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": music,
	})
}

func (h *MusicHandler) Upload(c *gin.Context) {
	userID := c.GetUint("user_id")
	isAdmin := c.GetBool("is_admin")

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Файл не найден"})
		return
	}
	defer file.Close()

	result, err := h.musicService.Upload(file, header, userID, isAdmin)
	if err != nil {
		if err.Error() == "превышен лимит хранилища музыки (1 ГБ)" {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"data": result,
	})
}

func (h *MusicHandler) Stream(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID"})
		return
	}

	filePath, err := h.musicService.GetFilePath(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.File(filePath)
}

func (h *MusicHandler) Download(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID"})
		return
	}

	music, err := h.musicService.GetMusic(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	filePath, err := h.musicService.GetFilePath(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Disposition", "attachment; filename=\""+music.OriginalName+"\"")
	c.File(filePath)
}

func (h *MusicHandler) ListPending(c *gin.Context) {
	music, err := h.musicService.ListPending()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения списка"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": music})
}

func (h *MusicHandler) Approve(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID"})
		return
	}
	if err := h.musicService.Approve(uint(id)); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Трек одобрен"})
}

func (h *MusicHandler) Reject(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID"})
		return
	}
	if err := h.musicService.Reject(uint(id)); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Трек отклонён"})
}

func (h *MusicHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID"})
		return
	}

	if err := h.musicService.Delete(uint(id)); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Трек удалён",
	})
}
