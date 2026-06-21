package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
)

type UploadHandler struct {
	uploadDir string
}

func NewUploadHandler(uploadDir string) *UploadHandler {
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		panic(fmt.Sprintf("failed to create upload dir: %v", err))
	}
	return &UploadHandler{uploadDir: uploadDir}
}

func (h *UploadHandler) Upload(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Файл не найден"})
		return
	}
	defer file.Close()

	// Определяем тип
	ext := filepath.Ext(header.Filename)
	mime := header.Header.Get("Content-Type")
	var fileType string
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		fileType = "image"
	case ".mp4", ".webm", ".mov":
		fileType = "video"
	case ".mp3", ".wav", ".ogg":
		fileType = "audio"
	default:
		fileType = "file"
	}

	// Уникальное имя
	filename := fmt.Sprintf("%d_%s", time.Now().UnixNano(), header.Filename)
	dest := filepath.Join(h.uploadDir, filename)

	out, err := os.Create(dest)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка сохранения файла"})
		return
	}
	defer out.Close()

	if _, err := io.Copy(out, file); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка записи файла"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"url":  "/uploads/" + filename,
		"name": header.Filename,
		"size": header.Size,
		"type": fileType,
		"mime": mime,
	})
}
