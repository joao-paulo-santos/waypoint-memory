package mcp

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"log"
	"net/http"
	"strings"

	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/joao-paulo-santos/waypoint-memory/services"
)

type MCPServer struct {
	Server       *mcpserver.MCPServer
	CentralDB    *sql.DB
	ProjectSvc   *services.ProjectService
	BoardSvc     *services.BoardService
	LabelSvc     *services.LabelService
	SprintSvc    *services.SprintService
	ActivitySvc  *services.ActivityService
	ContactSvc   *services.ContactService
	BirthdaySvc  *services.BirthdayService
	CalendarSvc  *services.CalendarService
	WikiSvc      *services.WikiService
	RecurringSvc *services.RecurringEventService
}

func NewMCPServer(
	centralDB *sql.DB,
	projectSvc *services.ProjectService,
	boardSvc *services.BoardService,
	labelSvc *services.LabelService,
	sprintSvc *services.SprintService,
	activitySvc *services.ActivityService,
	contactSvc *services.ContactService,
	birthdaySvc *services.BirthdayService,
	calendarSvc *services.CalendarService,
	wikiSvc *services.WikiService,
	recurringSvc *services.RecurringEventService,
) *MCPServer {
	s := &MCPServer{
		CentralDB:    centralDB,
		ProjectSvc:   projectSvc,
		BoardSvc:     boardSvc,
		LabelSvc:     labelSvc,
		SprintSvc:    sprintSvc,
		ActivitySvc:  activitySvc,
		ContactSvc:   contactSvc,
		BirthdaySvc:  birthdaySvc,
		CalendarSvc:  calendarSvc,
		WikiSvc:      wikiSvc,
		RecurringSvc: recurringSvc,
	}

	mcpSrv := mcpserver.NewMCPServer("waypoint-memory", "1.0.0")
	s.Server = mcpSrv

	s.registerTools()

	return s
}

func (s *MCPServer) Start(addr string) error {
	sseServer := mcpserver.NewSSEServer(s.Server,
		mcpserver.WithBaseURL("http://localhost"+addr),
	)
	log.Printf("MCP server starting on %s", addr)

	mux := http.NewServeMux()
	mux.Handle("/sse", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.authenticate(r) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		sseServer.SSEHandler().ServeHTTP(w, r)
	}))
	mux.Handle("/message", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	var expiresAt sql.NullString
	err := s.CentralDB.QueryRow(
		"SELECT id, expires_at FROM api_tokens WHERE token_hash = ?",
		hashHex,
	).Scan(&id, &expiresAt)
	if err != nil {
		return false
	}

	if expiresAt.Valid && expiresAt.String != "" {
		return false
	}

	s.CentralDB.Exec("UPDATE api_tokens SET last_used = datetime('now') WHERE id = ?", id)
	return true
}
