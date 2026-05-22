package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/joao-paulo-santos/waypoint-memory/models"
	"github.com/joao-paulo-santos/waypoint-memory/services"
)

type BoardHandler struct {
	DB         *sql.DB
	ProjectSvc *services.ProjectService
	BoardSvc   *services.BoardService
	CommentSvc *services.CommentService
}

func NewBoardHandler(db *sql.DB, projectSvc *services.ProjectService, activitySvc *services.ActivityService) *BoardHandler {
	return &BoardHandler{
		DB:         db,
		ProjectSvc: projectSvc,
		BoardSvc:   &services.BoardService{Activity: activitySvc},
		CommentSvc: &services.CommentService{Activity: activitySvc},
	}
}

func (h *BoardHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.GetBoard)

	r.Post("/buckets", h.CreateBucket)
	r.Put("/buckets/{bid}", h.UpdateBucket)
	r.Delete("/buckets/{bid}", h.DeleteBucket)
	r.Patch("/buckets/reorder", h.ReorderBuckets)

	r.Post("/tasks", h.CreateTask)
	r.Get("/tasks/{tid}", h.GetTask)
	r.Put("/tasks/{tid}", h.UpdateTask)
	r.Delete("/tasks/{tid}", h.DeleteTask)
	r.Post("/tasks/{tid}/move", h.MoveTask)
	r.Patch("/tasks/reorder", h.ReorderTasks)
	r.Post("/tasks/{tid}/comments", h.AddComment)
	r.Get("/tasks/{tid}/comments", h.ListComments)

	return r
}

func (h *BoardHandler) getProjectID(r *http.Request) (int64, error) {
	projectIDStr := chi.URLParam(r, "projectId")
	return strconv.ParseInt(projectIDStr, 10, 64)
}

func (h *BoardHandler) GetBoard(w http.ResponseWriter, r *http.Request) {
	projectID, err := h.getProjectID(r)
	if err != nil {
		http.Error(w, "invalid project id", http.StatusBadRequest)
		return
	}

	board, err := h.BoardSvc.GetBoard(h.DB, projectID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(board)
}

func (h *BoardHandler) CreateBucket(w http.ResponseWriter, r *http.Request) {
	projectID, err := h.getProjectID(r)
	if err != nil {
		http.Error(w, "invalid project id", http.StatusBadRequest)
		return
	}

	var req models.CreateBucketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	bucket, err := h.BoardSvc.CreateBucket(h.DB, projectID, req)
	if err != nil {
		writeError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(bucket)
}

func (h *BoardHandler) UpdateBucket(w http.ResponseWriter, r *http.Request) {
	projectID, err := h.getProjectID(r)
	if err != nil {
		http.Error(w, "invalid project id", http.StatusBadRequest)
		return
	}

	bid, err := strconv.ParseInt(chi.URLParam(r, "bid"), 10, 64)
	if err != nil {
		http.Error(w, "invalid bucket id", http.StatusBadRequest)
		return
	}

	var req models.UpdateBucketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	bucket, err := h.BoardSvc.UpdateBucket(h.DB, projectID, bid, req)
	if err != nil {
		writeError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bucket)
}

func (h *BoardHandler) DeleteBucket(w http.ResponseWriter, r *http.Request) {
	projectID, err := h.getProjectID(r)
	if err != nil {
		http.Error(w, "invalid project id", http.StatusBadRequest)
		return
	}

	bid, err := strconv.ParseInt(chi.URLParam(r, "bid"), 10, 64)
	if err != nil {
		http.Error(w, "invalid bucket id", http.StatusBadRequest)
		return
	}

	if err := h.BoardSvc.DeleteBucket(h.DB, projectID, bid); err != nil {
		writeError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *BoardHandler) ReorderBuckets(w http.ResponseWriter, r *http.Request) {
	projectID, err := h.getProjectID(r)
	if err != nil {
		http.Error(w, "invalid project id", http.StatusBadRequest)
		return
	}

	var req models.ReorderBucketsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.BoardSvc.ReorderBuckets(h.DB, projectID, req); err != nil {
		writeError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *BoardHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	projectID, err := h.getProjectID(r)
	if err != nil {
		http.Error(w, "invalid project id", http.StatusBadRequest)
		return
	}

	var req models.CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	task, err := h.BoardSvc.CreateTask(h.DB, projectID, req)
	if err != nil {
		writeError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(task)
}

func (h *BoardHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	tid, err := strconv.ParseInt(chi.URLParam(r, "tid"), 10, 64)
	if err != nil {
		http.Error(w, "invalid task id", http.StatusBadRequest)
		return
	}

	task, err := h.BoardSvc.GetTask(h.DB, tid)
	if err != nil {
		writeError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

func (h *BoardHandler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	projectID, err := h.getProjectID(r)
	if err != nil {
		http.Error(w, "invalid project id", http.StatusBadRequest)
		return
	}

	tid, err := strconv.ParseInt(chi.URLParam(r, "tid"), 10, 64)
	if err != nil {
		http.Error(w, "invalid task id", http.StatusBadRequest)
		return
	}

	var req models.UpdateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	task, err := h.BoardSvc.UpdateTask(h.DB, projectID, tid, req)
	if err != nil {
		writeError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

func (h *BoardHandler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	projectID, err := h.getProjectID(r)
	if err != nil {
		http.Error(w, "invalid project id", http.StatusBadRequest)
		return
	}

	tid, err := strconv.ParseInt(chi.URLParam(r, "tid"), 10, 64)
	if err != nil {
		http.Error(w, "invalid task id", http.StatusBadRequest)
		return
	}

	if err := h.BoardSvc.DeleteTask(h.DB, projectID, tid); err != nil {
		writeError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *BoardHandler) MoveTask(w http.ResponseWriter, r *http.Request) {
	projectID, err := h.getProjectID(r)
	if err != nil {
		http.Error(w, "invalid project id", http.StatusBadRequest)
		return
	}

	tid, err := strconv.ParseInt(chi.URLParam(r, "tid"), 10, 64)
	if err != nil {
		http.Error(w, "invalid task id", http.StatusBadRequest)
		return
	}

	var req models.MoveTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	task, err := h.BoardSvc.MoveTask(h.DB, projectID, tid, req.BucketID)
	if err != nil {
		writeError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

func (h *BoardHandler) ReorderTasks(w http.ResponseWriter, r *http.Request) {
	projectID, err := h.getProjectID(r)
	if err != nil {
		http.Error(w, "invalid project id", http.StatusBadRequest)
		return
	}

	var req models.ReorderTasksRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.BoardSvc.ReorderTasks(h.DB, projectID, req); err != nil {
		writeError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *BoardHandler) AddComment(w http.ResponseWriter, r *http.Request) {
	projectID, err := h.getProjectID(r)
	if err != nil {
		http.Error(w, "invalid project id", http.StatusBadRequest)
		return
	}

	tid, err := strconv.ParseInt(chi.URLParam(r, "tid"), 10, 64)
	if err != nil {
		http.Error(w, "invalid task id", http.StatusBadRequest)
		return
	}

	var req models.CreateCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	comment, err := h.CommentSvc.AddComment(h.DB, projectID, tid, req)
	if err != nil {
		writeError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(comment)
}

func (h *BoardHandler) ListComments(w http.ResponseWriter, r *http.Request) {
	tid, err := strconv.ParseInt(chi.URLParam(r, "tid"), 10, 64)
	if err != nil {
		http.Error(w, "invalid task id", http.StatusBadRequest)
		return
	}

	comments, err := h.CommentSvc.ListComments(h.DB, tid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"comments": comments})
}
