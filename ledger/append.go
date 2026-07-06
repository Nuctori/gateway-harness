package ledger

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type AppendRecord struct {
	Project   AppendProject `json:"project"`
	Session   AppendSession `json:"session"`
	Event     Event         `json:"event"`
	Artifacts []Artifact    `json:"artifacts,omitempty"`
}

type AppendProject struct {
	ID   string   `json:"id"`
	Name string   `json:"name,omitempty"`
	Tags []string `json:"tags,omitempty"`
}

type AppendSession struct {
	ID        string   `json:"id"`
	Title     string   `json:"title,omitempty"`
	StartedAt string   `json:"started_at,omitempty"`
	Tags      []string `json:"tags,omitempty"`
}

type AppendResult struct {
	ProjectID string `json:"project_id"`
	SessionID string `json:"session_id"`
	EventID   string `json:"event_id"`
	Events    int    `json:"events"`
	Artifacts int    `json:"artifacts"`
}

func DecodeAppendRecord(r io.Reader) (AppendRecord, error) {
	var record AppendRecord
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&record); err != nil {
		return AppendRecord{}, err
	}
	return record, nil
}

func Append(l Ledger, record AppendRecord) (Ledger, AppendResult, error) {
	l = cloneLedger(l)
	if strings.TrimSpace(record.Project.ID) == "" {
		return Ledger{}, AppendResult{}, fmt.Errorf("project id is required")
	}
	if strings.TrimSpace(record.Session.ID) == "" {
		return Ledger{}, AppendResult{}, fmt.Errorf("session id is required")
	}
	if strings.TrimSpace(record.Event.ID) == "" {
		return Ledger{}, AppendResult{}, fmt.Errorf("event id is required")
	}

	projectIndex := findProject(l.Projects, record.Project.ID)
	if projectIndex < 0 {
		project := Project{
			ID:       record.Project.ID,
			Name:     record.Project.Name,
			Tags:     appendUniqueTags(nil, record.Project.Tags),
			Sessions: []Session{},
		}
		l.Projects = append(l.Projects, project)
		projectIndex = len(l.Projects) - 1
	} else {
		project := &l.Projects[projectIndex]
		if strings.TrimSpace(project.Name) == "" && strings.TrimSpace(record.Project.Name) != "" {
			project.Name = record.Project.Name
		}
		project.Tags = appendUniqueTags(project.Tags, record.Project.Tags)
	}

	project := &l.Projects[projectIndex]
	sessionIndex := findSession(project.Sessions, record.Session.ID)
	if sessionIndex < 0 {
		if strings.TrimSpace(record.Session.StartedAt) == "" {
			return Ledger{}, AppendResult{}, fmt.Errorf("new session %q started_at is required", record.Session.ID)
		}
		session := Session{
			ID:        record.Session.ID,
			Title:     record.Session.Title,
			StartedAt: record.Session.StartedAt,
			Tags:      appendUniqueTags(nil, record.Session.Tags),
			Events:    []Event{},
			Artifacts: []Artifact{},
		}
		project.Sessions = append(project.Sessions, session)
		sessionIndex = len(project.Sessions) - 1
	} else {
		session := &project.Sessions[sessionIndex]
		if strings.TrimSpace(session.Title) == "" && strings.TrimSpace(record.Session.Title) != "" {
			session.Title = record.Session.Title
		}
		session.Tags = appendUniqueTags(session.Tags, record.Session.Tags)
	}

	session := &project.Sessions[sessionIndex]
	artifactIDs := map[string]bool{}
	for _, artifact := range session.Artifacts {
		artifactIDs[artifact.ID] = true
	}
	for _, artifact := range record.Artifacts {
		if artifactIDs[artifact.ID] {
			return Ledger{}, AppendResult{}, fmt.Errorf("session %q duplicate artifact id %q", session.ID, artifact.ID)
		}
		session.Artifacts = append(session.Artifacts, artifact)
		artifactIDs[artifact.ID] = true
	}
	for _, event := range session.Events {
		if event.ID == record.Event.ID {
			return Ledger{}, AppendResult{}, fmt.Errorf("session %q duplicate event id %q", session.ID, record.Event.ID)
		}
	}
	session.Events = append(session.Events, record.Event)

	if err := Validate(l); err != nil {
		return Ledger{}, AppendResult{}, err
	}
	return l, AppendResult{
		ProjectID: project.ID,
		SessionID: session.ID,
		EventID:   record.Event.ID,
		Events:    len(session.Events),
		Artifacts: len(session.Artifacts),
	}, nil
}

func findProject(projects []Project, id string) int {
	for i, project := range projects {
		if project.ID == id {
			return i
		}
	}
	return -1
}

func findSession(sessions []Session, id string) int {
	for i, session := range sessions {
		if session.ID == id {
			return i
		}
	}
	return -1
}

func appendUniqueTags(existing []string, tags []string) []string {
	seen := map[string]bool{}
	next := make([]string, 0, len(existing)+len(tags))
	for _, tag := range existing {
		tag = strings.TrimSpace(tag)
		if tag == "" || seen[tag] {
			continue
		}
		seen[tag] = true
		next = append(next, tag)
	}
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" || seen[tag] {
			continue
		}
		seen[tag] = true
		next = append(next, tag)
	}
	return next
}

func cloneLedger(l Ledger) Ledger {
	next := Ledger{
		Version:  l.Version,
		Projects: make([]Project, len(l.Projects)),
	}
	for i, project := range l.Projects {
		next.Projects[i] = Project{
			ID:       project.ID,
			Name:     project.Name,
			Tags:     append([]string(nil), project.Tags...),
			Sessions: make([]Session, len(project.Sessions)),
		}
		for j, session := range project.Sessions {
			next.Projects[i].Sessions[j] = Session{
				ID:        session.ID,
				Title:     session.Title,
				StartedAt: session.StartedAt,
				Tags:      append([]string(nil), session.Tags...),
				Events:    append([]Event(nil), session.Events...),
				Artifacts: append([]Artifact(nil), session.Artifacts...),
			}
		}
	}
	return next
}
