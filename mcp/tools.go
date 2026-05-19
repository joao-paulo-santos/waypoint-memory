package mcp

import (
	"context"
	"path/filepath"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/joao-paulo-santos/waypoint-memory/models"
	"github.com/joao-paulo-santos/waypoint-memory/services"
)

func (s *MCPServer) registerTools() {
	s.Server.AddTool(s.listProjectsTool(), s.handleListProjects)
	s.Server.AddTool(s.getProjectContextTool(), s.handleGetProjectContext)

	s.Server.AddTool(s.listBucketsTool(), s.handleListBuckets)
	s.Server.AddTool(s.createBucketTool(), s.handleCreateBucket)
	s.Server.AddTool(s.updateBucketTool(), s.handleUpdateBucket)
	s.Server.AddTool(s.deleteBucketTool(), s.handleDeleteBucket)

	s.Server.AddTool(s.createTaskTool(), s.handleCreateTask)
	s.Server.AddTool(s.updateTaskTool(), s.handleUpdateTask)
	s.Server.AddTool(s.moveTaskTool(), s.handleMoveTask)
	s.Server.AddTool(s.deleteTaskTool(), s.handleDeleteTask)

	s.Server.AddTool(s.archiveSprintTool(), s.handleArchiveSprint)
	s.Server.AddTool(s.listSprintsTool(), s.handleListSprints)
	s.Server.AddTool(s.getSprintTool(), s.handleGetSprint)

	s.Server.AddTool(s.appendProjectLogTool(), s.handleAppendProjectLog)
	s.Server.AddTool(s.getRecentActivityTool(), s.handleGetRecentActivity)

	s.Server.AddTool(s.listContactsTool(), s.handleListContacts)
	s.Server.AddTool(s.createContactTool(), s.handleCreateContact)

	s.Server.AddTool(s.getUpcomingBirthdaysTool(), s.handleGetUpcomingBirthdays)
	s.Server.AddTool(s.addBirthdayTool(), s.handleAddBirthday)

	s.Server.AddTool(s.getCalendarTool(), s.handleGetCalendar)

	s.Server.AddTool(s.listRecurringEventsTool(), s.handleListRecurringEvents)
	s.Server.AddTool(s.createRecurringEventTool(), s.handleCreateRecurringEvent)
}

// --- list_projects ---

func (s *MCPServer) listProjectsTool() mcp.Tool {
	return mcp.NewTool("list_projects",
		mcp.WithDescription("List all registered projects with task counts and overdue counts."),
		mcp.WithReadOnlyHintAnnotation(true),
	)
}

func (s *MCPServer) handleListProjects(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projects, err := s.ProjectSvc.List()
	if err != nil {
		return toolError("list projects: %v", err)
	}

	type projectWithCounts struct {
		models.Project
		TaskCount    int `json:"task_count"`
		OverdueCount int `json:"overdue_count"`
	}

	var result []projectWithCounts
	for _, p := range projects {
		pw := projectWithCounts{Project: p}
		projDB, err := s.ProjectSvc.GetProjectDB(p.ID)
		if err == nil {
			projDB.QueryRow("SELECT count(*) FROM tasks WHERE done = 0").Scan(&pw.TaskCount)
			projDB.QueryRow("SELECT count(*) FROM tasks WHERE done = 0 AND due_date < date('now')").Scan(&pw.OverdueCount)
			projDB.Close()
		}
		result = append(result, pw)
	}

	return toolResult(map[string]any{"projects": result})
}

// --- get_project_context ---

func (s *MCPServer) getProjectContextTool() mcp.Tool {
	return mcp.NewTool("get_project_context",
		mcp.WithDescription("Get full project context: board summary, overdue tasks, due-soon tasks, and recent activity."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithInteger("project_id", mcp.Required(), mcp.Description("The project ID")),
	)
}

func (s *MCPServer) handleGetProjectContext(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	pid, err := requireInt(req, "project_id")
	if err != nil {
		return toolError("%v", err)
	}

	project, err := s.ProjectSvc.GetByID(pid)
	if err != nil {
		return toolError("get project: %v", err)
	}

	projDB, err := s.ProjectSvc.GetProjectDB(pid)
	if err != nil {
		return toolError("open project db: %v", err)
	}
	defer projDB.Close()

	board, _ := s.BoardSvc.GetBoard(projDB)
	activity, _ := s.ActivitySvc.GetProjectActivity(projDB, 5)

	var overdue []map[string]any
	rows, err := projDB.Query("SELECT id, title, due_date FROM tasks WHERE done = 0 AND due_date < date('now')")
	if err == nil {
		for rows.Next() {
			var id int64
			var title, dueDate string
			if rows.Scan(&id, &title, &dueDate) == nil {
				overdue = append(overdue, map[string]any{"id": id, "title": title, "due_date": dueDate})
			}
		}
		rows.Close()
	}

	var dueSoon []map[string]any
	rows2, err := projDB.Query("SELECT id, title, due_date FROM tasks WHERE done = 0 AND due_date >= date('now') AND due_date <= date('now', '+7 days')")
	if err == nil {
		for rows2.Next() {
			var id int64
			var title, dueDate string
			if rows2.Scan(&id, &title, &dueDate) == nil {
				dueSoon = append(dueSoon, map[string]any{"id": id, "title": title, "due_date": dueDate})
			}
		}
		rows2.Close()
	}

	boardSummary := map[string]any{}
	if board != nil {
		for _, b := range board.Buckets {
			boardSummary[b.Bucket.Title] = len(b.Tasks)
		}
	}

	return toolResult(map[string]any{
		"project": map[string]any{
			"id": project.ID, "name": project.Name, "is_archived": project.IsArchived,
		},
		"board_summary":   boardSummary,
		"overdue_tasks":   overdue,
		"due_soon":        dueSoon,
		"recent_activity": activity,
	})
}

// --- list_buckets ---

func (s *MCPServer) listBucketsTool() mcp.Tool {
	return mcp.NewTool("list_buckets",
		mcp.WithDescription("List all buckets in a project's board."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithInteger("project_id", mcp.Required(), mcp.Description("The project ID")),
	)
}

func (s *MCPServer) handleListBuckets(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	pid, err := requireInt(req, "project_id")
	if err != nil {
		return toolError("%v", err)
	}
	projDB, err := s.ProjectSvc.GetProjectDB(pid)
	if err != nil {
		return toolError("open project db: %v", err)
	}
	defer projDB.Close()

	board, err := s.BoardSvc.GetBoard(projDB)
	if err != nil {
		return toolError("get board: %v", err)
	}
	return toolResult(board)
}

// --- create_bucket ---

func (s *MCPServer) createBucketTool() mcp.Tool {
	return mcp.NewTool("create_bucket",
		mcp.WithDescription("Create a new bucket in a project's board."),
		mcp.WithInteger("project_id", mcp.Required(), mcp.Description("The project ID")),
		mcp.WithString("title", mcp.Required(), mcp.Description("Bucket title")),
	)
}

func (s *MCPServer) handleCreateBucket(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	pid, err := requireInt(req, "project_id")
	if err != nil {
		return toolError("%v", err)
	}
	title, err := requireString(req, "title")
	if err != nil {
		return toolError("%v", err)
	}

	projDB, err := s.ProjectSvc.GetProjectDB(pid)
	if err != nil {
		return toolError("open project db: %v", err)
	}
	defer projDB.Close()

	bucket, err := s.BoardSvc.CreateBucket(projDB, models.CreateBucketRequest{Title: title})
	if err != nil {
		return toolError("create bucket: %v", err)
	}
	return toolResult(bucket)
}

// --- update_bucket ---

func (s *MCPServer) updateBucketTool() mcp.Tool {
	return mcp.NewTool("update_bucket",
		mcp.WithDescription("Update a bucket's properties."),
		mcp.WithInteger("project_id", mcp.Required(), mcp.Description("The project ID")),
		mcp.WithInteger("bucket_id", mcp.Required(), mcp.Description("The bucket ID")),
		mcp.WithString("title", mcp.Description("New title")),
	)
}

func (s *MCPServer) handleUpdateBucket(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	pid, err := requireInt(req, "project_id")
	if err != nil {
		return toolError("%v", err)
	}
	bid, err := requireInt(req, "bucket_id")
	if err != nil {
		return toolError("%v", err)
	}

	updateReq := models.UpdateBucketRequest{}
	if v := getNullableString(req, "title"); v != nil {
		updateReq.Title = v
	}

	projDB, err := s.ProjectSvc.GetProjectDB(pid)
	if err != nil {
		return toolError("open project db: %v", err)
	}
	defer projDB.Close()

	bucket, err := s.BoardSvc.UpdateBucket(projDB, bid, updateReq)
	if err != nil {
		return toolError("update bucket: %v", err)
	}
	return toolResult(bucket)
}

// --- delete_bucket ---

func (s *MCPServer) deleteBucketTool() mcp.Tool {
	return mcp.NewTool("delete_bucket",
		mcp.WithDescription("Delete a bucket from a project's board."),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithInteger("project_id", mcp.Required(), mcp.Description("The project ID")),
		mcp.WithInteger("bucket_id", mcp.Required(), mcp.Description("The bucket ID")),
	)
}

func (s *MCPServer) handleDeleteBucket(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	pid, err := requireInt(req, "project_id")
	if err != nil {
		return toolError("%v", err)
	}
	bid, err := requireInt(req, "bucket_id")
	if err != nil {
		return toolError("%v", err)
	}

	projDB, err := s.ProjectSvc.GetProjectDB(pid)
	if err != nil {
		return toolError("open project db: %v", err)
	}
	defer projDB.Close()

	if err := s.BoardSvc.DeleteBucket(projDB, bid); err != nil {
		return toolError("delete bucket: %v", err)
	}
	return toolResult(map[string]string{"status": "deleted"})
}

// --- create_task ---

func (s *MCPServer) createTaskTool() mcp.Tool {
	return mcp.NewTool("create_task",
		mcp.WithDescription("Create a new task in a project's Kanban board."),
		mcp.WithInteger("project_id", mcp.Required(), mcp.Description("The project ID")),
		mcp.WithInteger("bucket_id", mcp.Required(), mcp.Description("The bucket ID")),
		mcp.WithString("title", mcp.Required(), mcp.Description("Task title")),
		mcp.WithString("description", mcp.Description("Task description")),
		mcp.WithInteger("priority", mcp.Description("Task priority (0-3)")),
		mcp.WithString("due_date", mcp.Description("Due date (YYYY-MM-DD)")),
		mcp.WithString("labels", mcp.Description("Comma-separated label IDs")),
	)
}

func (s *MCPServer) handleCreateTask(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	pid, err := requireInt(req, "project_id")
	if err != nil {
		return toolError("%v", err)
	}
	bid, err := requireInt(req, "bucket_id")
	if err != nil {
		return toolError("%v", err)
	}
	title, err := requireString(req, "title")
	if err != nil {
		return toolError("%v", err)
	}

	projDB, err := s.ProjectSvc.GetProjectDB(pid)
	if err != nil {
		return toolError("open project db: %v", err)
	}
	defer projDB.Close()

	taskReq := models.CreateTaskRequest{
		BucketID:    bid,
		Title:       title,
		Description: getString(req, "description", ""),
		Priority:    getInt(req, "priority", 0),
		CreatedBy:   "ai",
	}

	if dd := getNullableString(req, "due_date"); dd != nil {
		taskReq.DueDate = dd
	}
	if lbl := getString(req, "labels", ""); lbl != "" {
		taskReq.Labels = lbl
	}

	task, err := s.BoardSvc.CreateTask(projDB, taskReq)
	if err != nil {
		return toolError("create task: %v", err)
	}
	return toolResult(task)
}

// --- update_task ---

func (s *MCPServer) updateTaskTool() mcp.Tool {
	return mcp.NewTool("update_task",
		mcp.WithDescription("Update a task's properties. Only provided fields are changed."),
		mcp.WithInteger("project_id", mcp.Required(), mcp.Description("The project ID")),
		mcp.WithInteger("task_id", mcp.Required(), mcp.Description("The task ID")),
		mcp.WithString("title", mcp.Description("New title")),
		mcp.WithString("description", mcp.Description("New description")),
		mcp.WithInteger("bucket_id", mcp.Description("Move to this bucket")),
		mcp.WithInteger("priority", mcp.Description("New priority")),
		mcp.WithString("due_date", mcp.Description("New due date (YYYY-MM-DD)")),
		mcp.WithBoolean("done", mcp.Description("Mark done/undone")),
		mcp.WithString("labels", mcp.Description("Comma-separated label IDs")),
	)
}

func (s *MCPServer) handleUpdateTask(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	pid, err := requireInt(req, "project_id")
	if err != nil {
		return toolError("%v", err)
	}
	tid, err := requireInt(req, "task_id")
	if err != nil {
		return toolError("%v", err)
	}

	projDB, err := s.ProjectSvc.GetProjectDB(pid)
	if err != nil {
		return toolError("open project db: %v", err)
	}
	defer projDB.Close()

	updateReq := models.UpdateTaskRequest{}
	if v := getNullableString(req, "title"); v != nil {
		updateReq.Title = v
	}
	if v := getNullableString(req, "description"); v != nil {
		updateReq.Description = v
	}
	if v := getNullableInt(req, "bucket_id"); v != nil {
		updateReq.BucketID = v
	}
	if v := getInt(req, "priority", 0); v != 0 {
		updateReq.Priority = &v
	}
	if v := getNullableString(req, "due_date"); v != nil {
		updateReq.DueDate = v
	}
	if v := getNullableBool(req, "done"); v != nil {
		updateReq.Done = v
	}
	if v := getNullableString(req, "labels"); v != nil {
		updateReq.Labels = v
	}

	task, err := s.BoardSvc.UpdateTask(projDB, tid, updateReq)
	if err != nil {
		return toolError("update task: %v", err)
	}
	return toolResult(task)
}

// --- move_task ---

func (s *MCPServer) moveTaskTool() mcp.Tool {
	return mcp.NewTool("move_task",
		mcp.WithDescription("Move a task to a different bucket."),
		mcp.WithInteger("project_id", mcp.Required(), mcp.Description("The project ID")),
		mcp.WithInteger("task_id", mcp.Required(), mcp.Description("The task ID")),
		mcp.WithInteger("bucket_id", mcp.Required(), mcp.Description("Target bucket ID")),
	)
}

func (s *MCPServer) handleMoveTask(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	pid, err := requireInt(req, "project_id")
	if err != nil {
		return toolError("%v", err)
	}
	tid, err := requireInt(req, "task_id")
	if err != nil {
		return toolError("%v", err)
	}
	bid, err := requireInt(req, "bucket_id")
	if err != nil {
		return toolError("%v", err)
	}

	projDB, err := s.ProjectSvc.GetProjectDB(pid)
	if err != nil {
		return toolError("open project db: %v", err)
	}
	defer projDB.Close()

	task, err := s.BoardSvc.MoveTask(projDB, tid, bid)
	if err != nil {
		return toolError("move task: %v", err)
	}
	return toolResult(task)
}

// --- delete_task ---

func (s *MCPServer) deleteTaskTool() mcp.Tool {
	return mcp.NewTool("delete_task",
		mcp.WithDescription("Delete a task."),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithInteger("project_id", mcp.Required(), mcp.Description("The project ID")),
		mcp.WithInteger("task_id", mcp.Required(), mcp.Description("The task ID")),
	)
}

func (s *MCPServer) handleDeleteTask(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	pid, err := requireInt(req, "project_id")
	if err != nil {
		return toolError("%v", err)
	}
	tid, err := requireInt(req, "task_id")
	if err != nil {
		return toolError("%v", err)
	}

	projDB, err := s.ProjectSvc.GetProjectDB(pid)
	if err != nil {
		return toolError("open project db: %v", err)
	}
	defer projDB.Close()

	if err := s.BoardSvc.DeleteTask(projDB, tid); err != nil {
		return toolError("delete task: %v", err)
	}
	return toolResult(map[string]string{"status": "deleted"})
}

// --- archive_sprint ---

func (s *MCPServer) archiveSprintTool() mcp.Tool {
	return mcp.NewTool("archive_sprint",
		mcp.WithDescription("Archive all tasks in the Done bucket as a sprint."),
		mcp.WithInteger("project_id", mcp.Required(), mcp.Description("The project ID")),
		mcp.WithString("sprint_name", mcp.Description("Optional sprint name")),
		mcp.WithString("summary", mcp.Description("Optional sprint summary")),
	)
}

func (s *MCPServer) handleArchiveSprint(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	pid, err := requireInt(req, "project_id")
	if err != nil {
		return toolError("%v", err)
	}

	projDB, err := s.ProjectSvc.GetProjectDB(pid)
	if err != nil {
		return toolError("open project db: %v", err)
	}
	defer projDB.Close()

	result, err := s.SprintSvc.EndSprint(projDB, models.EndSprintRequest{
		SprintName: getString(req, "sprint_name", ""),
		Summary:    getString(req, "summary", ""),
	})
	if err != nil {
		return toolError("archive sprint: %v", err)
	}
	return toolResult(result)
}

// --- list_sprints ---

func (s *MCPServer) listSprintsTool() mcp.Tool {
	return mcp.NewTool("list_sprints",
		mcp.WithDescription("List all archived sprints for a project."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithInteger("project_id", mcp.Required(), mcp.Description("The project ID")),
	)
}

func (s *MCPServer) handleListSprints(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	pid, err := requireInt(req, "project_id")
	if err != nil {
		return toolError("%v", err)
	}

	projDB, err := s.ProjectSvc.GetProjectDB(pid)
	if err != nil {
		return toolError("open project db: %v", err)
	}
	defer projDB.Close()

	sprints, err := s.SprintSvc.ListSprints(projDB)
	if err != nil {
		return toolError("list sprints: %v", err)
	}
	return toolResult(map[string]any{"sprints": sprints})
}

// --- get_sprint ---

func (s *MCPServer) getSprintTool() mcp.Tool {
	return mcp.NewTool("get_sprint",
		mcp.WithDescription("Get sprint detail with archived tasks."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithInteger("project_id", mcp.Required(), mcp.Description("The project ID")),
		mcp.WithInteger("sprint_id", mcp.Required(), mcp.Description("The sprint ID")),
	)
}

func (s *MCPServer) handleGetSprint(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	pid, err := requireInt(req, "project_id")
	if err != nil {
		return toolError("%v", err)
	}
	sid, err := requireInt(req, "sprint_id")
	if err != nil {
		return toolError("%v", err)
	}

	projDB, err := s.ProjectSvc.GetProjectDB(pid)
	if err != nil {
		return toolError("open project db: %v", err)
	}
	defer projDB.Close()

	detail, err := s.SprintSvc.GetSprintDetail(projDB, sid)
	if err != nil {
		return toolError("get sprint: %v", err)
	}
	return toolResult(detail)
}

// --- append_project_log ---

func (s *MCPServer) appendProjectLogTool() mcp.Tool {
	return mcp.NewTool("append_project_log",
		mcp.WithDescription("Add a custom activity log entry to a project."),
		mcp.WithInteger("project_id", mcp.Required(), mcp.Description("The project ID")),
		mcp.WithString("action", mcp.Required(), mcp.Description("Action description")),
		mcp.WithString("details", mcp.Description("JSON details")),
	)
}

func (s *MCPServer) handleAppendProjectLog(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	pid, err := requireInt(req, "project_id")
	if err != nil {
		return toolError("%v", err)
	}
	action, err := requireString(req, "action")
	if err != nil {
		return toolError("%v", err)
	}

	projDB, err := s.ProjectSvc.GetProjectDB(pid)
	if err != nil {
		return toolError("open project db: %v", err)
	}
	defer projDB.Close()

	err = s.ActivitySvc.LogActivity(projDB, services.LogActivityParams{
		Action:     action,
		EntityType: "custom",
		Actor:      "ai",
		Details:    map[string]any{"note": getString(req, "details", "")},
	})
	if err != nil {
		return toolError("log activity: %v", err)
	}
	return toolResult(map[string]string{"status": "logged"})
}

// --- get_recent_activity ---

func (s *MCPServer) getRecentActivityTool() mcp.Tool {
	return mcp.NewTool("get_recent_activity",
		mcp.WithDescription("Get recent activity log for a project."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithInteger("project_id", mcp.Required(), mcp.Description("The project ID")),
		mcp.WithInteger("limit", mcp.Description("Number of entries (default 20)")),
	)
}

func (s *MCPServer) handleGetRecentActivity(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	pid, err := requireInt(req, "project_id")
	if err != nil {
		return toolError("%v", err)
	}

	projDB, err := s.ProjectSvc.GetProjectDB(pid)
	if err != nil {
		return toolError("open project db: %v", err)
	}
	defer projDB.Close()

	entries, err := s.ActivitySvc.GetProjectActivity(projDB, getInt(req, "limit", 20))
	if err != nil {
		return toolError("get activity: %v", err)
	}
	return toolResult(map[string]any{"activity": entries})
}

// --- list_contacts ---

func (s *MCPServer) listContactsTool() mcp.Tool {
	return mcp.NewTool("list_contacts",
		mcp.WithDescription("List all contacts."),
		mcp.WithReadOnlyHintAnnotation(true),
	)
}

func (s *MCPServer) handleListContacts(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	contacts, err := s.ContactSvc.List()
	if err != nil {
		return toolError("list contacts: %v", err)
	}
	return toolResult(map[string]any{"contacts": contacts})
}

// --- create_contact ---

func (s *MCPServer) createContactTool() mcp.Tool {
	return mcp.NewTool("create_contact",
		mcp.WithDescription("Create a new contact."),
		mcp.WithString("first_name", mcp.Required(), mcp.Description("First name")),
		mcp.WithString("last_name", mcp.Description("Last name")),
		mcp.WithString("email", mcp.Description("Email")),
		mcp.WithString("phone", mcp.Description("Phone")),
		mcp.WithString("company", mcp.Description("Company")),
		mcp.WithString("role", mcp.Description("Role")),
		mcp.WithString("notes", mcp.Description("Notes")),
	)
}

func (s *MCPServer) handleCreateContact(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	contact, err := s.ContactSvc.Create(models.CreateContactRequest{
		FirstName: getString(req, "first_name", ""),
		LastName:  getString(req, "last_name", ""),
		Email:     getString(req, "email", ""),
		Phone:     getString(req, "phone", ""),
		Company:   getString(req, "company", ""),
		Role:      getString(req, "role", ""),
		Notes:     getString(req, "notes", ""),
	})
	if err != nil {
		return toolError("create contact: %v", err)
	}
	return toolResult(contact)
}

// --- get_upcoming_birthdays ---

func (s *MCPServer) getUpcomingBirthdaysTool() mcp.Tool {
	return mcp.NewTool("get_upcoming_birthdays",
		mcp.WithDescription("Get upcoming birthdays within a date range."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithInteger("days", mcp.Description("Days ahead (default 30)")),
	)
}

func (s *MCPServer) handleGetUpcomingBirthdays(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	birthdays, err := s.BirthdaySvc.GetUpcoming(getInt(req, "days", 30))
	if err != nil {
		return toolError("get upcoming birthdays: %v", err)
	}
	return toolResult(map[string]any{"birthdays": birthdays})
}

// --- add_birthday ---

func (s *MCPServer) addBirthdayTool() mcp.Tool {
	return mcp.NewTool("add_birthday",
		mcp.WithDescription("Add a birthday."),
		mcp.WithString("name", mcp.Required(), mcp.Description("Person's name")),
		mcp.WithString("date", mcp.Required(), mcp.Description("Date in MM-DD format")),
		mcp.WithInteger("year", mcp.Description("Birth year")),
		mcp.WithInteger("contact_id", mcp.Description("Linked contact ID")),
	)
}

func (s *MCPServer) handleAddBirthday(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	createReq := models.CreateBirthdayRequest{
		Name: getString(req, "name", ""),
		Date: getString(req, "date", ""),
	}

	if v := getNullableInt(req, "year"); v != nil {
		y := int(*v)
		createReq.Year = &y
	}
	if v := getNullableInt(req, "contact_id"); v != nil {
		createReq.ContactID = v
	}

	birthday, err := s.BirthdaySvc.Create(createReq)
	if err != nil {
		return toolError("add birthday: %v", err)
	}
	return toolResult(birthday)
}

// --- get_calendar ---

func (s *MCPServer) getCalendarTool() mcp.Tool {
	return mcp.NewTool("get_calendar",
		mcp.WithDescription("Get calendar events for a date range."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithString("from", mcp.Description("Start date (YYYY-MM-DD), defaults to today")),
		mcp.WithString("to", mcp.Description("End date (YYYY-MM-DD), defaults to +7 days")),
	)
}

func (s *MCPServer) handleGetCalendar(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	result, err := s.CalendarSvc.GetCalendar(
		getString(req, "from", ""),
		getString(req, "to", ""),
	)
	if err != nil {
		return toolError("get calendar: %v", err)
	}
	return toolResult(result)
}

// --- list_recurring_events ---

func (s *MCPServer) listRecurringEventsTool() mcp.Tool {
	return mcp.NewTool("list_recurring_events",
		mcp.WithDescription("List all recurring events."),
		mcp.WithReadOnlyHintAnnotation(true),
	)
}

func (s *MCPServer) handleListRecurringEvents(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	events, err := s.RecurringSvc.List()
	if err != nil {
		return toolError("list recurring events: %v", err)
	}
	return toolResult(map[string]any{"events": events})
}

// --- create_recurring_event ---

func (s *MCPServer) createRecurringEventTool() mcp.Tool {
	return mcp.NewTool("create_recurring_event",
		mcp.WithDescription("Create a recurring event (yearly or monthly)."),
		mcp.WithString("title", mcp.Required(), mcp.Description("Event title")),
		mcp.WithString("start_date", mcp.Required(), mcp.Description("Start date (YYYY-MM-DD)")),
		mcp.WithString("date", mcp.Required(), mcp.Description("MM-DD for yearly, DD for monthly")),
		mcp.WithString("recurrence", mcp.Description("yearly or monthly (default yearly)")),
		mcp.WithString("category", mcp.Description("Category")),
		mcp.WithString("description", mcp.Description("Description")),
		mcp.WithString("end_date", mcp.Description("End date (YYYY-MM-DD)")),
	)
}

func (s *MCPServer) handleCreateRecurringEvent(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	createReq := models.CreateRecurringEventRequest{
		Title:       getString(req, "title", ""),
		StartDate:   getString(req, "start_date", ""),
		Date:        getString(req, "date", ""),
		Recurrence:  getString(req, "recurrence", "yearly"),
		Category:    getString(req, "category", ""),
		Description: getString(req, "description", ""),
	}
	if v := getNullableString(req, "end_date"); v != nil {
		createReq.EndDate = v
	}

	event, err := s.RecurringSvc.Create(createReq)
	if err != nil {
		return toolError("create recurring event: %v", err)
	}
	return toolResult(event)
}

// --- Wiki helper ---

func (s *MCPServer) getWikiPage(projectID int64, slug string) (*models.WikiPage, error) {
	project, err := s.ProjectSvc.GetByID(projectID)
	if err != nil {
		return nil, err
	}
	waypointDir := filepath.Join(project.Path, ".waypoint")
	return s.WikiSvc.ReadWikiPage(waypointDir, slug)
}
