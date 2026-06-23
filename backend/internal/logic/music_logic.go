package logic

import (
	"MessangerMax/internal/entity"
	"MessangerMax/internal/repo"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dhowden/tag"
)

const MaxMusicStorage int64 = 1 << 30

type MusicService interface {
	List() ([]entity.MusicResponse, error)
	ListPending() ([]entity.MusicResponse, error)
	Upload(file multipart.File, header *multipart.FileHeader, userID uint, isAdmin bool) (*entity.MusicUploadResponse, error)
	GetFilePath(id uint) (string, error)
	GetMusic(id uint) (*entity.Music, error)
	Delete(id uint) error
	Approve(id uint) error
	Reject(id uint) error
}

type musicService struct {
	musicRepo repo.MusicRepository
	musicDir  string
}

func NewMusicService(musicRepo repo.MusicRepository, musicDir string) MusicService {
	return &musicService{musicRepo: musicRepo, musicDir: musicDir}
}

func toMusicResponse(m entity.Music) entity.MusicResponse {
	return entity.MusicResponse{
		ID:           m.ID,
		Title:        m.Title,
		Artist:       m.Artist,
		OriginalName: m.OriginalName,
		Size:         m.Size,
		MimeType:     m.MimeType,
		UploadedBy:   m.UploadedBy,
		Status:       m.Status,
		CreatedAt:    m.CreatedAt,
	}
}

func (s *musicService) List() ([]entity.MusicResponse, error) {
	all, err := s.musicRepo.FindAll()
	if err != nil {
		return nil, err
	}

	resp := make([]entity.MusicResponse, 0, len(all))
	for _, m := range all {
		if m.Status != "" && m.Status != entity.MusicStatusApproved {
			continue
		}
		resp = append(resp, toMusicResponse(m))
	}
	return resp, nil
}

func (s *musicService) ListPending() ([]entity.MusicResponse, error) {
	all, err := s.musicRepo.FindByStatus(entity.MusicStatusPending)
	if err != nil {
		return nil, err
	}
	resp := make([]entity.MusicResponse, 0, len(all))
	for _, m := range all {
		resp = append(resp, toMusicResponse(m))
	}
	return resp, nil
}

func (s *musicService) Upload(file multipart.File, header *multipart.FileHeader, userID uint, isAdmin bool) (*entity.MusicUploadResponse, error) {
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !entity.MusicExts[ext] {
		return nil, errors.New("неподдерживаемый формат аудио")
	}

	totalSize, err := s.musicRepo.GetTotalSize()
	if err != nil {
		return nil, errors.New("ошибка проверки хранилища")
	}

	if totalSize+header.Size > MaxMusicStorage {
		return nil, errors.New("превышен лимит хранилища музыки (1 ГБ)")
	}

	title := header.Filename
	artist := ""

	tagData, tagErr := tag.ReadFrom(file)
	if tagErr == nil {
		if t := tagData.Title(); t != "" {
			title = t
		}
		if a := tagData.Artist(); a != "" {
			artist = a
		}
	}
	if _, seekErr := file.Seek(0, io.SeekStart); seekErr != nil {
		return nil, errors.New("ошибка чтения файла")
	}

	filename := fmt.Sprintf("%d_%s", time.Now().UnixNano(), header.Filename)
	dest := filepath.Join(s.musicDir, filename)

	if err := os.MkdirAll(s.musicDir, 0755); err != nil {
		log.Printf("failed to create music dir: %v", err)
		return nil, errors.New("ошибка сервера")
	}

	out, err := os.Create(dest)
	if err != nil {
		log.Printf("failed to create music file: %v", err)
		return nil, errors.New("ошибка сохранения файла")
	}
	defer out.Close()

	if _, err := io.Copy(out, file); err != nil {
		return nil, errors.New("ошибка записи файла")
	}

	mime := header.Header.Get("Content-Type")
	if mime == "" {
		mime = "audio/mpeg"
	}

	status := entity.MusicStatusPending
	if isAdmin {
		status = entity.MusicStatusApproved
	}

	music := &entity.Music{
		Filename:     filename,
		OriginalName: header.Filename,
		Title:        title,
		Artist:       artist,
		Size:         header.Size,
		MimeType:     mime,
		UploadedBy:   userID,
		Status:       status,
	}

	if err := s.musicRepo.Create(music); err != nil {
		os.Remove(dest)
		return nil, errors.New("ошибка сохранения метаданных")
	}

	return &entity.MusicUploadResponse{
		ID:     music.ID,
		Title:  title,
		Artist: artist,
	}, nil
}

func (s *musicService) GetFilePath(id uint) (string, error) {
	music, err := s.musicRepo.FindByID(id)
	if err != nil {
		return "", errors.New("трек не найден")
	}
	return filepath.Join(s.musicDir, music.Filename), nil
}

func (s *musicService) GetMusic(id uint) (*entity.Music, error) {
	return s.musicRepo.FindByID(id)
}

func (s *musicService) Approve(id uint) error {
	if err := s.musicRepo.UpdateStatus(id, entity.MusicStatusApproved); err != nil {
		return errors.New("ошибка одобрения трека")
	}
	return nil
}

func (s *musicService) Reject(id uint) error {
	music, err := s.musicRepo.FindByID(id)
	if err != nil {
		return errors.New("трек не найден")
	}
	if err := s.musicRepo.UpdateStatus(id, entity.MusicStatusRejected); err != nil {
		return err
	}
	os.Remove(filepath.Join(s.musicDir, music.Filename))
	return nil
}

func (s *musicService) Delete(id uint) error {
	music, err := s.musicRepo.FindByID(id)
	if err != nil {
		return errors.New("трек не найден")
	}

	if err := s.musicRepo.Delete(id); err != nil {
		return err
	}

	os.Remove(filepath.Join(s.musicDir, music.Filename))
	return nil
}
