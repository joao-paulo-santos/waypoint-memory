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
	ProjectSvc *services.ProjectService
	BoardSvc   *services.BoardService
}

func NewBoardHandler(projectSvc *services.ProjectService) *BoardHandler {
	return &BoardHandler{
		ProjectSvc: projectSvc,
		BoardSvc:   &services.BoardService{},
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

	return r
}

func (h *BoardHandler) getProjectDB(r *http.Request) (*sql.DB, func(), error) {
	projectIDStr := chi.URLParam(r, "projectId")
	projectID, err := strconv.ParseInt(projectIDStr, 10, 64)
	if err != nil {
		return nil, nil, err
	}

	db, err := h.ProjectSvc.GetProjectDB(projectID)
	if err != nil {
		return nil, nil, err
	}

	return db, func() { db.Close() }, nil
}

func (h *BoardHandler) GetBoard(w http.ResponseWriter, r *http.Request) {
	db, close, err := h.getProjectDB(r)
	if err != nil {
		writeError(w, err)
		return
	}
	defer close()

	board, err := h.BoardSvc.GetBoard(db)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(board)
}

func (h *BoardHandler) CreateBucket(w http.ResponseWriter, r *http.Request) {
	db, close, err := h.getProjectDB(r)
	if err != nil {
		writeError(w, err)
		return
	}
	defer close()

	var req models.CreateBucketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	bucket, err := h.BoardSvc.CreateBucket(db, req)
	if err != nil {
		writeError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(bucket)
}

func (h *BoardHandler) UpdateBucket(w http.ResponseWriter, r *http.Request) {
	db, close, err := h.getProjectDB(r)
	if err != nil {
		writeError(w, err)
		return
	}
	defer close()

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

	bucket, err := h.BoardSvc.UpdateBucket(db, bid, req)
	if err != nil {
		writeError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bucket)
}

func (h *BoardHandler) DeleteBucket(w http.ResponseWriter, r *http.Request) {
	db, close, err := h.getProjectDB(r)
	if err != nil {
		writeError(w, err)
		return
	}
	defer close()

	bid, err := strconv.ParseInt(chi.URLParam(r, "bid"), 10, 64)
	if err != nil {
		http.Error(w, "invalid bucket id", http.StatusBadRequest)
		return
	}

	if err := h.BoardSvc.DeleteBucket(db, bid); err != nil {
		writeError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *BoardHandler) ReorderBuckets(w http.ResponseWriter, r *http.Request) {
	db, close, err := h.getProjectDB(r)
	if err != nil {
		writeError(w, err)
		return
	}
	defer close()

	var req models.ReorderBucketsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.BoardSvc.ReorderBuckets(db, req); err != nil {
		writeError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *BoardHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	db, close, err := h.getProjectDB(r)
	if err != nil {
		writeError(w, err)
		return
	}
	defer close()

	var req models.CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	task, err := h.BoardSvc.CreateTask(db, req)
	if err != nil {
		writeError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(task)
}

func (h *BoardHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	db, close, err := h.getProjectDB(r)
	if err != nil {
		writeError(w, err)
		return
	}
	defer close()

	tid, err := strconv.ParseInt(chi.URLParam(r, "tid"), 10, 64)
	if err != nil {
		http.Error(w, "invalid task id", http.StatusBadRequest)
		return
	}

	task, err := h.BoardSvc.GetTask(db, tid)
	if err != nil {
		writeError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

func (h *BoardHandler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	db, close, err := h.getProjectDB(r)
	if err != nil {
		writeError(w, err)
		return
	}
	defer close()

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

	task, err := h.BoardSvc.UpdateTask(db, tid, req)
	if err != nil {
		writeError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

func (h *BoardHandler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	db, close, err := h.getProjectDB(r)
	if err != nil {
		writeError(w, err)
		return
	}
	defer close()

	tid, err := strconv.ParseInt(chi.URLParam(r, "tid"), 10, 64)
	if err != nil {
		http.Error(w, "invalid task id", http.StatusBadRequest)
		return
	}

	if err := h.BoardSvc.DeleteTask(db, tid); err != nil {
		writeError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *BoardHandler) MoveTask(w http.ResponseWriter, r *http.Request) {
	db, close, err := h.getProjectDB(r)
	if err != nil {
		writeError(w, err)
		return
	}
	defer close()

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

	task, err := h.BoardSvc.MoveTask(db, tid, req.BucketID)
	if err != nil {
		writeError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

func (h *BoardHandler) ReorderTasks(w http.ResponseWriter, r *http.Request) {
	db, close, err := h.getProjectDB(r)
	if err != nil {
		writeError(w, err)
		return
	}
	defer close()

	var req models.ReorderTasksRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.BoardSvc.ReorderTasks(db, req); err != nil {
		writeError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
