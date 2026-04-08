package kickstarter

import (
	"io"
	"net/http"
	"strconv"

	"isbudget/backend/internal/auth"
	"isbudget/backend/internal/common"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		common.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	projectID, ok := common.ParseUUIDParam(w, r, "projectId", "project")
	if !ok {
		return
	}
	if err := r.ParseMultipartForm(16 << 20); err != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		common.WriteError(w, http.StatusBadRequest, "csv file is required")
		return
	}
	defer file.Close()

	content, err := io.ReadAll(io.LimitReader(file, 16<<20))
	if err != nil {
		common.WriteError(w, http.StatusBadRequest, "failed to read csv file")
		return
	}
	replaceExisting, _ := strconv.ParseBool(r.FormValue("replace_existing"))

	item, err := h.svc.ImportCSV(r.Context(), user.ID, projectID, header.Filename, content, replaceExisting)
	if err != nil {
		common.WriteMappedError(w, err, http.StatusInternalServerError, "kickstarter_import_failed", "failed to import kickstarter csv",
			common.ErrorMapping{Err: ErrImportWriteDenied, Status: http.StatusForbidden, Code: "forbidden"},
			common.ErrorMapping{Err: ErrInvalidCSVStructure, Status: http.StatusBadRequest, Code: "invalid_csv"},
			common.ErrorMapping{Err: ErrNoEligibleRows, Status: http.StatusBadRequest, Code: "invalid_csv"},
			common.ErrorMapping{Err: ErrImportConflictExistingIncome, Status: http.StatusConflict, Code: "conflict"},
		)
		return
	}
	common.WriteJSON(w, http.StatusCreated, item)
}

func (h *Handler) ListRuns(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		common.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	projectID, ok := common.ParseUUIDParam(w, r, "projectId", "project")
	if !ok {
		return
	}
	items, err := h.svc.ListRuns(r.Context(), user.ID, projectID)
	if err != nil {
		common.WriteMappedError(w, err, http.StatusInternalServerError, "import_list_failed", "failed to list imports",
			common.ErrorMapping{Err: ErrImportReadDenied, Status: http.StatusForbidden, Code: "forbidden"},
		)
		return
	}
	common.WriteJSON(w, http.StatusOK, items)
}

func (h *Handler) GetRun(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		common.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	runID, ok := common.ParseUUIDParam(w, r, "id", "import")
	if !ok {
		return
	}
	item, err := h.svc.GetRun(r.Context(), user.ID, runID)
	if err != nil {
		common.WriteMappedError(w, err, http.StatusInternalServerError, "import_get_failed", "failed to load import",
			common.ErrorMapping{Err: ErrImportReadDenied, Status: http.StatusForbidden, Code: "forbidden"},
			common.ErrorMapping{Err: ErrImportRunNotFound, Status: http.StatusNotFound, Code: "not_found"},
		)
		return
	}
	common.WriteJSON(w, http.StatusOK, item)
}
