package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"runtime"

	"github.com/go-chi/chi/v5"
	"github.com/joao-paulo-santos/waypoint-memory/config"
	"github.com/joao-paulo-santos/waypoint-memory/db"
	"github.com/joao-paulo-santos/waypoint-memory/handlers"
	"github.com/joao-paulo-santos/waypoint-memory/mcp"
	"github.com/joao-paulo-santos/waypoint-memory/services"
)

var (
	buildVersion = "dev"
	centralDB    *sql.DB
)

func main() {
	if config.HandleVersion() {
		fmt.Printf("waypoint %s\n", buildVersion)
		return
	}

	cfg := config.Load()

	if err := cfg.EnsureDataDir(); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	var err error
	centralDB, err = db.InitCentralDB(cfg.CentralDBPath())
	if err != nil {
		log.Fatalf("Failed to initialize central DB: %v", err)
	}
	defer centralDB.Close()

	r := chi.NewRouter()
	r.Get("/api/v1/health", handleHealth)

	cfg.EnsureProjectsDir()
	projectSvc := services.NewProjectService(centralDB, cfg.ProjectsDir())
	projectHandler := handlers.NewProjectHandler(projectSvc)
	r.Mount("/api/v1/projects", projectHandler.Routes())

	activitySvc := services.NewActivityService()

	boardHandler := handlers.NewBoardHandler(projectSvc, activitySvc)
	r.Mount("/api/v1/projects/{projectId}/board", boardHandler.Routes())

	labelHandler := handlers.NewLabelHandler(projectSvc, activitySvc)
	r.Mount("/api/v1/projects/{projectId}/labels", labelHandler.Routes())

	sprintHandler := handlers.NewSprintHandler(projectSvc)
	r.Mount("/api/v1/projects/{projectId}/sprints", sprintHandler.Routes())

	activityHandler := handlers.NewActivityHandler(projectSvc, activitySvc)
	r.Get("/api/v1/projects/{projectId}/activity", activityHandler.ProjectActivity)
	r.Get("/api/v1/activity", activityHandler.GlobalActivity)

	contactHandler := handlers.NewContactHandler(services.NewContactService(centralDB))
	r.Mount("/api/v1/contacts", contactHandler.Routes())

	birthdaySvc := services.NewBirthdayService(centralDB)
	recurringSvc := services.NewRecurringEventService(centralDB)

	birthdayHandler := handlers.NewBirthdayHandler(birthdaySvc)
	r.Mount("/api/v1/birthdays", birthdayHandler.Routes())

	eventHandler := handlers.NewRecurringEventHandler(recurringSvc)
	r.Mount("/api/v1/events", eventHandler.Routes())

	calendarSvc := services.NewCalendarService(centralDB, projectSvc, birthdaySvc, recurringSvc)

	calendarHandler := handlers.NewCalendarHandler(calendarSvc)
	r.Get("/api/v1/calendar", calendarHandler.GetCalendar)
	r.Get("/api/v1/calendar/today", calendarHandler.GetToday)

	wikiHandler := handlers.NewWikiHandler(projectSvc)
	r.Mount("/api/v1/projects/{projectId}/wiki", wikiHandler.Routes())

	tokenHandler := handlers.NewTokenHandler(services.NewTokenService(centralDB))
	r.Mount("/api/v1/tokens", tokenHandler.Routes())

	if !cfg.NoMCP {
		mcpServer := mcp.NewMCPServer(
			centralDB, projectSvc, boardHandler.BoardSvc, labelHandler.LabelSvc,
			sprintHandler.SprintSvc, activitySvc,
			services.NewContactService(centralDB), birthdaySvc,
			calendarSvc, services.NewWikiService(), recurringSvc,
		)
		go func() {
			if err := mcpServer.Start(cfg.MCPAddr); err != nil {
				log.Fatalf("MCP server failed: %v", err)
			}
		}()
	}

	r.Handle("/*", frontendFileServer())

	addr := cfg.WebAddr
	fmt.Printf("Waypoint Memory %s\n", buildVersion)
	fmt.Printf("Open http://localhost%s\n", addr)

	if cfg.Open {
		openBrowser("http://localhost" + addr)
	}

	log.Printf("Starting server on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}
