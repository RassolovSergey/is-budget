package auth

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"isbudget/backend/internal/common"
	"isbudget/backend/internal/db/sqlc"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	_ "golang.org/x/image/webp"
)

type Handler struct {
	svc       *Service
	validate  *validator.Validate
	avatarDir string
}

func NewHandler(svc *Service, avatarDir ...string) *Handler {
	dir := filepath.Join("storage", "avatars")
	if len(avatarDir) > 0 && strings.TrimSpace(avatarDir[0]) != "" {
		dir = avatarDir[0]
	}
	return &Handler{svc: svc, validate: common.NewValidator(), avatarDir: dir}
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if !common.DecodeAndValidate(w, r, h.validate, &req) {
		return
	}

	result, err := h.svc.Register(r.Context(), req.Email, req.Password, req.FirstName, req.LastName, req.Nickname)
	if err != nil {
		common.WriteMappedError(w, err, http.StatusInternalServerError, "register_failed", "failed to register",
			common.ErrorMapping{Err: ErrInvalidNickname, Status: http.StatusBadRequest, Code: "invalid_request"},
			common.ErrorMapping{Err: ErrNicknameAlreadyExists, Status: http.StatusConflict, Code: "conflict"},
			common.ErrorMapping{Err: ErrEmailAlreadyExists, Status: http.StatusConflict, Code: "conflict"},
		)
		return
	}

	common.WriteJSON(w, http.StatusCreated, result)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if !common.DecodeAndValidate(w, r, h.validate, &req) {
		return
	}

	result, err := h.svc.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		common.WriteMappedError(w, err, http.StatusInternalServerError, "login_failed", "failed to login",
			common.ErrorMapping{Err: ErrInvalidCredentials, Status: http.StatusUnauthorized, Code: "unauthorized", Message: "invalid credentials"},
		)
		return
	}

	common.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		common.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	me, err := h.svc.Me(r.Context(), user.ID.String())
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "failed to load user")
		return
	}

	common.WriteJSON(w, http.StatusOK, me)
}

func (h *Handler) UpdatePreferences(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		common.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req updatePreferencesRequest
	if !common.DecodeAndValidate(w, r, h.validate, &req) {
		return
	}

	updated, err := h.svc.UpdatePreferences(r.Context(), user.ID.String(), req.DefaultCurrencyCode)
	if err != nil {
		common.WriteMappedError(w, err, http.StatusInternalServerError, "update_preferences_failed", "failed to update preferences",
			common.ErrorMapping{Err: ErrInvalidDefaultCurrency, Status: http.StatusBadRequest, Code: "invalid_request"},
		)
		return
	}

	common.WriteJSON(w, http.StatusOK, updated)
}

func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		common.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req updateProfileRequest
	if !common.DecodeAndValidate(w, r, h.validate, &req) {
		return
	}

	updated, err := h.svc.UpdateProfile(r.Context(), user.ID.String(), req.FirstName, req.LastName, req.Nickname, req.DefaultCurrencyCode)
	if err != nil {
		common.WriteMappedError(w, err, http.StatusInternalServerError, "update_profile_failed", "failed to update profile",
			common.ErrorMapping{Err: ErrInvalidName, Status: http.StatusBadRequest, Code: "invalid_request"},
			common.ErrorMapping{Err: ErrInvalidNickname, Status: http.StatusBadRequest, Code: "invalid_request"},
			common.ErrorMapping{Err: ErrNicknameAlreadyExists, Status: http.StatusBadRequest, Code: "invalid_request"},
			common.ErrorMapping{Err: ErrInvalidDefaultCurrency, Status: http.StatusBadRequest, Code: "invalid_request"},
		)
		return
	}

	common.WriteJSON(w, http.StatusOK, updated)
}

func (h *Handler) UpdateEmail(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		common.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req updateEmailRequest
	if !common.DecodeAndValidate(w, r, h.validate, &req) {
		return
	}

	result, err := h.svc.UpdateEmail(r.Context(), user.ID.String(), req.CurrentPassword, req.NewEmail)
	if err != nil {
		common.WriteMappedError(w, err, http.StatusInternalServerError, "update_email_failed", "failed to update email",
			common.ErrorMapping{Err: ErrCurrentPasswordInvalid, Status: http.StatusUnauthorized, Code: "unauthorized"},
			common.ErrorMapping{Err: ErrEmailAlreadyExists, Status: http.StatusConflict, Code: "conflict"},
		)
		return
	}
	common.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) UpdatePassword(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		common.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req updatePasswordRequest
	if !common.DecodeAndValidate(w, r, h.validate, &req) {
		return
	}

	if err := h.svc.UpdatePassword(r.Context(), user.ID.String(), req.CurrentPassword, req.NewPassword); err != nil {
		common.WriteMappedError(w, err, http.StatusInternalServerError, "update_password_failed", "failed to update password",
			common.ErrorMapping{Err: ErrCurrentPasswordInvalid, Status: http.StatusUnauthorized, Code: "unauthorized"},
			common.ErrorMapping{Err: ErrWeakPassword, Status: http.StatusBadRequest, Code: "invalid_request"},
		)
		return
	}
	common.WriteJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (h *Handler) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		common.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	current, err := h.svc.Me(r.Context(), user.ID.String())
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "failed to load current user")
		return
	}
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}
	file, header, err := r.FormFile("avatar")
	if err != nil {
		common.WriteError(w, http.StatusBadRequest, "avatar file is required")
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !isAllowedAvatarExtension(ext) {
		common.WriteError(w, http.StatusBadRequest, "supported formats: .jpg, .jpeg, .png, .webp")
		return
	}
	if header.Size > 5<<20 {
		common.WriteError(w, http.StatusBadRequest, "avatar file is too large; max 5 MB")
		return
	}

	sniff := make([]byte, 512)
	n, readErr := io.ReadFull(file, sniff)
	if readErr != nil && !errors.Is(readErr, io.EOF) && !errors.Is(readErr, io.ErrUnexpectedEOF) {
		common.WriteError(w, http.StatusBadRequest, "failed to read avatar")
		return
	}
	sniff = sniff[:n]
	if !isAllowedAvatarContent(ext, sniff) {
		common.WriteError(w, http.StatusBadRequest, "avatar content does not match allowed image format")
		return
	}
	reader := io.MultiReader(bytes.NewReader(sniff), file)

	if err := os.MkdirAll(h.avatarDir, 0o755); err != nil {
		common.WriteError(w, http.StatusInternalServerError, "failed to prepare avatars dir")
		return
	}
	fileName := user.ID.String() + "_" + fmt.Sprintf("%d", time.Now().UTC().UnixNano()) + ext
	target := filepath.Join(h.avatarDir, fileName)
	dst, err := os.Create(target)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "failed to save avatar")
		return
	}
	defer dst.Close()
	if _, err := io.Copy(dst, reader); err != nil {
		common.WriteError(w, http.StatusInternalServerError, "failed to save avatar")
		return
	}

	path := "/auth/avatar/" + fileName
	updated, err := h.svc.UpdateAvatarPath(r.Context(), user.ID.String(), &path)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "failed to update avatar path")
		return
	}

	if err := h.removeAvatarFile(previousAvatarPath(current)); err != nil {
		common.WriteError(w, http.StatusInternalServerError, "failed to replace previous avatar")
		return
	}
	common.WriteJSON(w, http.StatusOK, updated)
}

func (h *Handler) DeleteAvatar(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		common.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	current, err := h.svc.Me(r.Context(), user.ID.String())
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "failed to load current user")
		return
	}

	updated, err := h.svc.UpdateAvatarPath(r.Context(), user.ID.String(), nil)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "failed to remove avatar")
		return
	}

	if err := h.removeAvatarFile(previousAvatarPath(current)); err != nil {
		common.WriteError(w, http.StatusInternalServerError, "failed to remove avatar file")
		return
	}

	common.WriteJSON(w, http.StatusOK, updated)
}

func isAllowedAvatarExtension(ext string) bool {
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp":
		return true
	default:
		return false
	}
}

func isAllowedAvatarContent(ext string, header []byte) bool {
	contentType := http.DetectContentType(header)
	switch ext {
	case ".jpg", ".jpeg":
		return contentType == "image/jpeg"
	case ".png":
		return contentType == "image/png"
	case ".webp":
		_, format, err := image.DecodeConfig(bytes.NewReader(header))
		return err == nil && format == "webp"
	default:
		return false
	}
}

func previousAvatarPath(user sqlc.User) *string {
	if user.AvatarPath == nil {
		return nil
	}
	path := strings.TrimSpace(*user.AvatarPath)
	if path == "" {
		return nil
	}
	return &path
}

func (h *Handler) removeAvatarFile(avatarPath *string) error {
	fileName, ok := avatarFileNameFromPath(avatarPath)
	if !ok {
		return nil
	}
	target := filepath.Join(h.avatarDir, fileName)
	if err := os.Remove(target); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func avatarFileNameFromPath(avatarPath *string) (string, bool) {
	if avatarPath == nil {
		return "", false
	}
	path := strings.TrimSpace(*avatarPath)
	if path == "" {
		return "", false
	}
	const prefix = "/auth/avatar/"
	if !strings.HasPrefix(path, prefix) {
		return "", false
	}
	fileName := strings.TrimPrefix(path, prefix)
	if fileName == "" || fileName != filepath.Base(fileName) || strings.Contains(fileName, "..") {
		return "", false
	}
	return fileName, true
}

func (h *Handler) GetAvatar(w http.ResponseWriter, r *http.Request) {
	fileName := filepath.Base(strings.TrimSpace(chi.URLParam(r, "fileName")))
	if fileName == "" || fileName == "." || strings.Contains(fileName, "..") {
		common.WriteError(w, http.StatusBadRequest, "invalid avatar file")
		return
	}
	target := filepath.Join(h.avatarDir, fileName)
	if _, err := os.Stat(target); err != nil {
		common.WriteError(w, http.StatusNotFound, "avatar not found")
		return
	}
	http.ServeFile(w, r, target)
}
