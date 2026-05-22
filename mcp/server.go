package mcp

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"log"
	"net/http"
	"strings"
	"time"

	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/joao-paulo-santos/waypoint-memory/services"
)

type MCPServer struct {
	Server       *mcpserver.MCPServer
	DB           *sql.DB
	ProjectSvc   *services.ProjectService
	BoardSvc     *services.BoardService
	LabelSvc     *services.LabelService
	SprintSvc    *services.SprintService
	ActivitySvc  *services.ActivityService
	ContactSvc   *services.ContactService
	CalendarSvc  *services.CalendarService
	WikiSvc      *services.WikiService
	EventSvc     *services.EventService
}

func NewMCPServer(
	db *sql.DB,
	projectSvc *services.ProjectService,
	boardSvc *services.BoardService,
	labelSvc *services.LabelService,
	sprintSvc *services.SprintService,
	activitySvc *services.ActivityService,
	contactSvc *services.ContactService,
	calendarSvc *services.CalendarService,
	wikiSvc *services.WikiService,
	eventSvc *services.EventService,
) *MCPServer {
	s := &MCPServer{
		DB:           db,
		ProjectSvc:   projectSvc,
		BoardSvc:     boardSvc,
		LabelSvc:     labelSvc,
		SprintSvc:    sprintSvc,
		ActivitySvc:  activitySvc,
		ContactSvc:   contactSvc,
		CalendarSvc:  calendarSvc,
		WikiSvc:      wikiSvc,
		EventSvc:     eventSvc,
	}

	mcpSrv := mcpserver.NewMCPServer("waypoint-memory", "1.0.0")
	s.Server = mcpSrv

	s.registerTools()

	return s
}

func (s *MCPServer) Start(addr string) error {
	sseServer := mcpserver.NewSSEServer(s.Server,
		mcpserver.WithBaseURL("http://172.19.2.10"+addr),
	)
	log.Printf("MCP server starting on %s", addr)

	mux := http.NewServeMux()
	mux.Handle("/sse", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		if !s.authenticate(r) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		sseServer.SSEHandler().ServeHTTP(w, r)
	}))
	mux.Handle("/message", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		if !s.authenticate(r) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		sseServer.MessageHandler().ServeHTTP(w, r)
	}))

	return http.ListenAndServe(addr, mux)
}

func (s *MCPServer) authenticate(r *http.Request) bool {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return false
	}
	token := strings.TrimPrefix(auth, "Bearer ")

	hash := sha256.Sum256([]byte(token))
	hashHex := hex.EncodeToString(hash[:])

	var id int64
	var expiresAt sql.NullTime
	err := s.DB.QueryRow(
		"SELECT id, expires_at FROM api_tokens WHERE token_hash = $1",
		hashHex,
	).Scan(&id, &expiresAt)
	if err != nil {
		return false
	}

	if expiresAt.Valid && expiresAt.Time.Before(time.Now()) {
		return false
	}

	s.DB.Exec("UPDATE api_tokens SET last_used = NOW() WHERE id = $1", id)
	return true
}

func (s *MCPServer) getDefaultOwnerID() int64 {
	var id int64
	err := s.DB.QueryRow("SELECT id FROM users ORDER BY id LIMIT 1").Scan(&id)
	if err != nil {
		return 1
	}
	return id
}

func (s *MCPServer) resolveOwnerID(token string) int64 {
	if token == "" {
		return s.getDefaultOwnerID()
	}
	hash := sha256.Sum256([]byte(token))
	hashHex := hex.EncodeToString(hash[:])
	var userID int64
	err := s.DB.QueryRow("SELECT user_id FROM api_tokens WHERE token_hash = $1", hashHex).Scan(&userID)
	if err != nil {
		return s.getDefaultOwnerID()
	}
	return userID
}
