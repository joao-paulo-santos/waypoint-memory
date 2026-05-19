package services

import (
	"database/sql"
	"encoding/json"
	"sort"

	"github.com/joao-paulo-santos/waypoint-memory/models"
)

type ActivityService struct{}

func NewActivityService() *ActivityService {
	return &ActivityService{}
}

type LogActivityParams struct {
	Action     string
	EntityType string
	EntityID   int64
	Actor      string
	Details    map[string]any
}

func (s *ActivityService) LogActivity(db *sql.DB, params LogActivityParams) error {
	detailsJSON := "{}"
	if params.Details != nil {
		data, err := json.Marshal(params.Details)
		if err != nil {
			detailsJSON = "{}"
		} else {
			detailsJSON = string(data)
		}
	}

	actor := params.Actor
	if actor == "" {
		actor = "user"
	}

	_, err := db.Exec(
		`INSERT INTO activity_log (action, entity_type, entity_id, actor, details)
		 VALUES (?, ?, ?, ?, ?)`,
		params.Action, params.EntityType, params.EntityID, actor, detailsJSON,
	)
	return err
}

func (s *ActivityService) GetProjectActivity(db *sql.DB, limit int) ([]models.ActivityEntry, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := db.Query(
		`SELECT id, action, entity_type, entity_id, actor, details, created_at
		 FROM activity_log ORDER BY id DESC LIMIT ?`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanActivityEntries(rows)
}

func (s *ActivityService) GetGlobalActivity(centralDB *sql.DB, projectSvc *ProjectService, limit int) ([]models.GlobalActivityEntry, error) {
	if limit <= 0 {
		limit = 50
	}

	projects, err := projectSvc.List()
	if err != nil {
		return nil, err
	}

	var allEntries []models.GlobalActivityEntry
	for _, p := range projects {
		projDB, err := projectSvc.GetProjectDB(p.ID)
		if err != nil {
			continue
		}

		entries, err := s.GetProjectActivity(projDB, limit)
		projDB.Close()
		if err != nil {
			continue
		}

		for _, e := range entries {
			allEntries = append(allEntries, models.GlobalActivityEntry{
				ActivityEntry: e,
				ProjectID:     p.ID,
				ProjectName:   p.Name,
			})
		}
	}

	sort.Slice(allEntries, func(i, j int) bool {
		return allEntries[i].CreatedAt > allEntries[j].CreatedAt
	})

	if len(allEntries) > limit {
		allEntries = allEntries[:limit]
	}

	return allEntries, nil
}

func scanActivityEntries(rows *sql.Rows) ([]models.ActivityEntry, error) {
	var entries []models.ActivityEntry
	for rows.Next() {
		var e models.ActivityEntry
		if err := rows.Scan(&e.ID, &e.Action, &e.EntityType, &e.EntityID,
			&e.Actor, &e.Details, &e.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, nil
}
