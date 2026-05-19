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

	authSvc := services.NewAuthService(centralDB)
	authHandler := handlers.NewAuthHandler(authSvc)

	cfg.EnsureProjectsDir()
	projectSvc := services.NewProjectService(centralDB, cfg.ProjectsDir())
	activitySvc := services.NewActivityService()
	birthdaySvc := services.NewBirthdayService(centralDB)
	recurringSvc := services.NewRecurringEventService(centralDB)
	calendarSvc := services.NewCalendarService(centralDB, projectSvc, birthdaySvc, recurringSvc)

	boardHandler := handlers.NewBoardHandler(projectSvc, activitySvc)
	labelHandler := handlers.NewLabelHandler(projectSvc, activitySvc)
	sprintHandler := handlers.NewSprintHandler(projectSvc)

	r := chi.NewRouter()

	r.Post("/api/v1/auth/login", authHandler.Login)
	r.Post("/api/v1/auth/logout", authHandler.Logout)
	r.Get("/api/v1/auth/status", authHandler.Status)
	r.Post("/api/v1/auth/set-password", authHandler.SetPassword)

	r.Route("/api", func(r chi.Router) {
		r.Use(handlers.AuthMiddleware(authSvc))

		r.Get("/v1/health", handleHealth)

		projectHandler := handlers.NewProjectHandler(projectSvc)
		r.Mount("/v1/projects", projectHandler.Routes())

		r.Mount("/v1/projects/{projectId}/board", boardHandler.Routes())
		r.Mount("/v1/projects/{projectId}/labels", labelHandler.Routes())
		r.Mount("/v1/projects/{projectId}/sprints", sprintHandler.Routes())

		activityHandler := handlers.NewActivityHandler(projectSvc, activitySvc)
		r.Get("/v1/projects/{projectId}/activity", activityHandler.ProjectActivity)
		r.Get("/v1/activity", activityHandler.GlobalActivity)

		contactHandler := handlers.NewContactHandler(services.NewContactService(centralDB))
		r.Mount("/v1/contacts", contactHandler.Routes())

		birthdayHandler := handlers.NewBirthdayHandler(birthdaySvc)
		r.Mount("/v1/birthdays", birthdayHandler.Routes())

		eventHandler := handlers.NewRecurringEventHandler(recurringSvc)
		r.Mount("/v1/events", eventHandler.Routes())

		calendarHandler := handlers.NewCalendarHandler(calendarSvc)
		r.Get("/v1/calendar", calendarHandler.GetCalendar)
		r.Get("/v1/calendar/today", calendarHandler.GetToday)

		wikiHandler := handlers.NewWikiHandler(projectSvc)
		r.Mount("/v1/projects/{projectId}/wiki", wikiHandler.Routes())

		tokenHandler := handlers.NewTokenHandler(services.NewTokenService(centralDB))
		r.Mount("/v1/tokens", tokenHandler.Routes())
	})

	if !cfg.Dev {
		r.Handle("/*", frontendFileServer())
	}

	webAddr := config.ResolveAddr(cfg.WebAddr)

	if !cfg.NoMCP {
		mcpAddr := config.ResolveAddr(cfg.MCPAddr)
		mcpServer := mcp.NewMCPServer(
			centralDB, projectSvc, boardHandler.BoardSvc, labelHandler.LabelSvc,
			sprintHandler.SprintSvc, activitySvc,
			services.NewContactService(centralDB), birthdaySvc,
			calendarSvc, services.NewWikiService(), recurringSvc,
		)
		go func() {
			if err := mcpServer.Start(mcpAddr); err != nil {
				log.Fatalf("MCP server failed: %v", err)
			}
		}()
		fmt.Printf("MCP server on http://localhost%s\n", mcpAddr)
	}

	fmt.Printf("Waypoint Memory %s\n", buildVersion)
	fmt.Printf("Open http://localhost%s\n", webAddr)

	if cfg.Open {
		openBrowser("http://localhost" + webAddr)
	}

	log.Printf("Starting server on %s", webAddr)
	if err := http.ListenAndServe(webAddr, r); err != nil {
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
