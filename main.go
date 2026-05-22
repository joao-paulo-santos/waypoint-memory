package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/joao-paulo-santos/waypoint-memory/config"
	"github.com/joao-paulo-santos/waypoint-memory/db"
	"github.com/joao-paulo-santos/waypoint-memory/handlers"
	"github.com/joao-paulo-santos/waypoint-memory/mcp"
	"github.com/joao-paulo-santos/waypoint-memory/services"
)

var (
	buildVersion = "dev"
)

func main() {
	if config.HandleVersion() {
		fmt.Printf("waypoint %s\n", buildVersion)
		return
	}

	cfg := config.Load()

	if err := cfg.Validate(); err != nil {
		log.Fatalf("Config error: %v", err)
	}

	database, err := db.InitDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	authSvc := services.NewAuthService(database)
	authHandler := handlers.NewAuthHandler(authSvc)

	projectSvc := services.NewProjectService(database)
	activitySvc := services.NewActivityService()
	contactSvc := services.NewContactService(database)
	eventSvc := services.NewEventService(database)
	calendarSvc := services.NewCalendarService(database, projectSvc, contactSvc, eventSvc)
	wikiSvc := services.NewWikiService(database)

	boardHandler := handlers.NewBoardHandler(database, projectSvc, activitySvc)
	labelHandler := handlers.NewLabelHandler(database, projectSvc, activitySvc)
	sprintHandler := handlers.NewSprintHandler(database, projectSvc)
	wikiHandler := handlers.NewWikiHandler(database, projectSvc, wikiSvc)

	r := chi.NewRouter()

	r.Post("/api/v1/auth/login", authHandler.Login)
	r.Post("/api/v1/auth/logout", authHandler.Logout)
	r.Post("/api/v1/auth/register", authHandler.Register)
	r.Get("/api/v1/auth/status", authHandler.Status)
	r.Get("/api/v1/auth/has-users", authHandler.HasUsers)

	r.Route("/api", func(r chi.Router) {
		r.Use(handlers.AuthMiddleware(authSvc))

		r.Get("/v1/health", handleHealth)

		projectHandler := handlers.NewProjectHandler(database, projectSvc)
		r.Mount("/v1/projects", projectHandler.Routes())

		r.Mount("/v1/projects/{projectId}/board", boardHandler.Routes())
		r.Mount("/v1/projects/{projectId}/labels", labelHandler.Routes())
		r.Mount("/v1/projects/{projectId}/sprints", sprintHandler.Routes())

		activityHandler := handlers.NewActivityHandler(database, projectSvc, activitySvc)
		r.Get("/v1/projects/{projectId}/activity", activityHandler.ProjectActivity)
		r.Get("/v1/activity", activityHandler.GlobalActivity)

		contactHandler := handlers.NewContactHandler(contactSvc)
		r.Mount("/v1/contacts", contactHandler.Routes())

		eventHandler := handlers.NewEventHandler(eventSvc)
		r.Mount("/v1/events", eventHandler.Routes())

		upcomingHandler := handlers.NewUpcomingHandler(database, projectSvc, contactSvc, eventSvc)
		r.Get("/v1/upcoming", upcomingHandler.GetUpcoming)

		calendarHandler := handlers.NewCalendarHandler(calendarSvc)
		r.Get("/v1/calendar", calendarHandler.GetCalendar)
		r.Get("/v1/calendar/today", calendarHandler.GetToday)

		r.Mount("/v1/projects/{projectId}/wiki", wikiHandler.Routes())

		tokenHandler := handlers.NewTokenHandler(services.NewTokenService(database))
		r.Mount("/v1/tokens", tokenHandler.Routes())
	})

	r.Handle("/*", frontendFileServer())

	webAddr := config.ResolveAddr(cfg.WebAddr)

	if !cfg.NoMCP {
		boardSvc := &services.BoardService{Activity: activitySvc}
		labelSvc := &services.LabelService{Activity: activitySvc}
		sprintSvc := &services.SprintService{}
		mcpServer := mcp.NewMCPServer(
			database, projectSvc, boardSvc, labelSvc, sprintSvc,
			activitySvc, contactSvc, calendarSvc, wikiSvc, eventSvc,
		)
		go func() {
			if err := mcpServer.Start(cfg.MCPAddr); err != nil {
				log.Printf("MCP server error: %v", err)
			}
		}()
	}

	fmt.Printf("Waypoint Memory %s\n", buildVersion)
	fmt.Printf("Open http://localhost%s\n", webAddr)

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
