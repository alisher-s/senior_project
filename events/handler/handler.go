package handler

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	authx "github.com/nu/student-event-ticketing-platform/internal/infra/auth"
	httpx "github.com/nu/student-event-ticketing-platform/internal/infra/http"
)

type coverImageResponseDTO struct {
	CoverImageURL string `json:"cover_image_url"`
}

type invalidImageErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func writeInvalidImage(w http.ResponseWriter, msg string) {
	var resp invalidImageErrorResponse
	resp.Error.Code = "INVALID_IMAGE"
	resp.Error.Message = msg
	_ = httpx.WriteJSON(w, http.StatusBadRequest, resp)
}

func contentTypeToExt(ct string) (string, bool) {
	switch ct {
	case "image/jpeg":
		return ".jpg", true
	case "image/png":
		return ".png", true
	case "image/webp":
		return ".webp", true
	default:
		return "", false
	}
}

func extractObjectNameFromPublicURL(fullURL string) string {
	fullURL = strings.TrimSpace(fullURL)
	if fullURL == "" {
		return ""
	}

	publicURL := strings.TrimRight(strings.TrimSpace(os.Getenv("MINIO_PUBLIC_URL")), "/")
	bucket := strings.TrimSpace(os.Getenv("MINIO_BUCKET"))
	if publicURL == "" || bucket == "" {
		return ""
	}

	prefix := publicURL + "/" + bucket + "/"
	if strings.HasPrefix(fullURL, prefix) {
		return strings.TrimPrefix(fullURL, prefix)
	}

	u, err := url.Parse(fullURL)
	if err != nil {
		return ""
	}
	path := strings.TrimPrefix(u.Path, "/")
	needle := bucket + "/"
	if idx := strings.Index(path, needle); idx >= 0 {
		return strings.TrimPrefix(path[idx:], needle)
	}
	return ""
}

func (h *handler) UploadCoverImage(w http.ResponseWriter, r *http.Request) {
	if h.storage == nil {
		httpx.WriteError(w, http.StatusInternalServerError, httpx.ErrCodeInternalError, "storage not configured")
		return
	}

	eventIDStr := chi.URLParam(r, "id")
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.ErrCodeInvalidID, "invalid event id")
		return
	}

	userID, ok := authx.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, httpx.ErrCodeUnauthorized, "missing user id")
		return
	}

	existing, err := h.svc.GetByID(r.Context(), eventID)
	if err != nil {
		status, apiErr := httpx.MapDomainError(err)
		httpx.WriteError(w, status, apiErr.Code, apiErr.Message)
		return
	}

	if authx.HasRole(r.Context(), authx.RoleOrganizer) && !authx.HasRole(r.Context(), authx.RoleAdmin) {
		if existing.OrganizerID == nil || *existing.OrganizerID != userID {
			httpx.WriteError(w, http.StatusForbidden, httpx.ErrCodeForbidden, "not allowed to modify this event")
			return
		}
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeInvalidImage(w, "invalid multipart form")
		return
	}

	file, fh, err := r.FormFile("image")
	if err != nil {
		writeInvalidImage(w, "missing image")
		return
	}
	defer file.Close()

	const maxSize = 5 << 20
	if fh.Size <= 0 {
		writeInvalidImage(w, "invalid image size")
		return
	}
	if fh.Size > maxSize {
		writeInvalidImage(w, "image too large (max 5MB)")
		return
	}

	var header [512]byte
	n, _ := io.ReadFull(file, header[:])
	detectedCT := http.DetectContentType(header[:n])
	ext, ok := contentTypeToExt(detectedCT)
	if !ok {
		writeInvalidImage(w, "unsupported image type (allowed: jpeg, png, webp)")
		return
	}

	objectName := fmt.Sprintf("covers/%s/%s%s", eventID.String(), uuid.New().String(), ext)
	reader := io.MultiReader(bytes.NewReader(header[:n]), file)

	publicURL, err := h.storage.UploadImage(r.Context(), objectName, reader, fh.Size, detectedCT)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, httpx.ErrCodeInternalError, "failed to upload image")
		return
	}

	if err := h.repo.UpdateCoverImage(r.Context(), eventID, publicURL); err != nil {
		_ = h.storage.DeleteImage(r.Context(), objectName)
		status, apiErr := httpx.MapDomainError(err)
		if status >= 500 {
			httpx.WriteError(w, http.StatusInternalServerError, httpx.ErrCodeInternalError, "failed to update event")
			return
		}
		httpx.WriteError(w, status, apiErr.Code, apiErr.Message)
		return
	}

	if oldURL := strings.TrimSpace(existing.CoverImageURL); oldURL != "" {
		oldObjectName := extractObjectNameFromPublicURL(oldURL)
		if oldObjectName != "" && oldObjectName != objectName {
			_ = h.storage.DeleteImage(r.Context(), oldObjectName)
		}
	}

	_ = httpx.WriteJSON(w, http.StatusOK, coverImageResponseDTO{CoverImageURL: publicURL})
}
