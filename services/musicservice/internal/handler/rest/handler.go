package rest

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/dhowden/tag"
	"github.com/gin-gonic/gin"

	"messengermax/musicservice/internal/entity"
	"messengermax/musicservice/internal/integration/filestore"
	"messengermax/musicservice/internal/repo"
)

type Handler struct {
	musicRepo repo.MusicRepository
	store     *filestore.Store
}

func NewHandler(musicRepo repo.MusicRepository, st *filestore.Store) *Handler {
	return &Handler{musicRepo: musicRepo, store: st}
}

var allowedExt = map[string]bool{
	".mp3": true, ".wav": true, ".ogg": true,
	".flac": true, ".aac": true, ".wma": true,
}

func (h *Handler) Upload(c *gin.Context) {
	userID := c.GetUint("user_id")
	isAdmin := c.GetBool("is_admin")

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": ""})
		return
	}
	defer file.Close()

	ext := filepath.Ext(header.Filename)
	if !allowedExt[ext] {
		c.JSON(http.StatusBadRequest, gin.H{"error": ""})
		return
	}

	total, _ := h.musicRepo.GetTotalSize()
	if total+header.Size > maxStorage {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": ""})
		return
	}

	filename, size, err := h.store.Save(ext, file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": ""})
		return
	}

	title, artist := header.Filename, ""
	if f, err := os.Open(h.store.Path(filename)); err == nil {
		if meta, err := tag.ReadFrom(f); err == nil {
			title = meta.Title()
			artist = meta.Artist()
		}
		_ = f.Close()
	}

	m := &entity.Music{
		Filename:     filename,
		OriginalName: header.Filename,
		Title:        title,
		Artist:       artist,
		Size:         size,
		MimeType:     header.Header.Get("Content-Type"),
		UploadedBy:   userID,
		Status:       entity.StatusPending,
	}
	if isAdmin {
		m.Status = entity.StatusApproved
	}

	if err := h.musicRepo.Create(m); err != nil {
		h.store.Remove(filename)
		c.JSON(http.StatusInternalServerError, gin.H{"error": ""})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": m.ID, "title": m.Title, "artist": m.Artist, "status": m.Status})
}

func (h *Handler) Stream(c *gin.Context) {
	m, ok := lookup(c, h.musicRepo)
	if !ok {
		return
	}
	c.File(h.store.Path(m.Filename))
}

func (h *Handler) Download(c *gin.Context) {
	m, ok := lookup(c, h.musicRepo)
	if !ok {
		return
	}
	c.Header("Content-Disposition", "attachment; filename=\""+m.OriginalName+"\"")
	c.File(h.store.Path(m.Filename))
}

func lookup(c *gin.Context, r repo.MusicRepository) (*entity.Music, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return nil, false
	}
	m, err := r.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": ""})
		return nil, false
	}
	return m, true
}

const maxStorage = 1 << 30 // 1 GB
