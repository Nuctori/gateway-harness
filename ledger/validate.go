package ledger

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

var SupportedEventTypes = map[string]bool{
	"request":        true,
	"response":       true,
	"tool_call":      true,
	"compact":        true,
	"failover":       true,
	"harness_action": true,
	"error":          true,
}

var SupportedArtifactTypes = map[string]bool{
	"compact_summary": true,
	"trace":           true,
	"request_hash":    true,
	"response_hash":   true,
}

var reservedMetadataKeys = map[string]bool{
	"content":      true,
	"input":        true,
	"message":      true,
	"messages":     true,
	"output":       true,
	"prompt":       true,
	"raw_content":  true,
	"raw_input":    true,
	"raw_output":   true,
	"raw_prompt":   true,
	"raw_response": true,
	"response":     true,
}

func Decode(r io.Reader) (Ledger, error) {
	var l Ledger
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&l); err != nil {
		return Ledger{}, err
	}
	return l, nil
}

func Validate(l Ledger) error {
	if len(l.Projects) == 0 {
		return fmt.Errorf("ledger needs at least one project")
	}
	projectIDs := map[string]bool{}
	for i, project := range l.Projects {
		if strings.TrimSpace(project.ID) == "" {
			return fmt.Errorf("project %d id is required", i)
		}
		if projectIDs[project.ID] {
			return fmt.Errorf("duplicate project id %q", project.ID)
		}
		projectIDs[project.ID] = true
		if len(project.Sessions) == 0 {
			return fmt.Errorf("project %q needs at least one session", project.ID)
		}
		if err := validateSessions(project.ID, project.Sessions); err != nil {
			return err
		}
	}
	return nil
}

func validateSessions(projectID string, sessions []Session) error {
	sessionIDs := map[string]bool{}
	for i, session := range sessions {
		if strings.TrimSpace(session.ID) == "" {
			return fmt.Errorf("project %q session %d id is required", projectID, i)
		}
		if sessionIDs[session.ID] {
			return fmt.Errorf("project %q duplicate session id %q", projectID, session.ID)
		}
		sessionIDs[session.ID] = true
		if err := validateRFC3339("started_at", session.StartedAt); err != nil {
			return fmt.Errorf("project %q session %q %w", projectID, session.ID, err)
		}
		if len(session.Events) == 0 {
			return fmt.Errorf("project %q session %q needs at least one event", projectID, session.ID)
		}
		artifactIDs, err := validateArtifacts(projectID, session.ID, session.Artifacts)
		if err != nil {
			return err
		}
		if err := validateEvents(projectID, session.ID, session.Events, artifactIDs); err != nil {
			return err
		}
	}
	return nil
}

func validateEvents(projectID string, sessionID string, events []Event, artifactIDs map[string]bool) error {
	eventIDs := map[string]bool{}
	for i, event := range events {
		if strings.TrimSpace(event.ID) == "" {
			return fmt.Errorf("project %q session %q event %d id is required", projectID, sessionID, i)
		}
		if eventIDs[event.ID] {
			return fmt.Errorf("project %q session %q duplicate event id %q", projectID, sessionID, event.ID)
		}
		eventIDs[event.ID] = true
		if !SupportedEventTypes[event.Type] {
			return fmt.Errorf("project %q session %q event %q unsupported type %q", projectID, sessionID, event.ID, event.Type)
		}
		if err := validateRFC3339("at", event.At); err != nil {
			return fmt.Errorf("project %q session %q event %q %w", projectID, sessionID, event.ID, err)
		}
		if err := validateMetadata("event", event.ID, event.Metadata); err != nil {
			return fmt.Errorf("project %q session %q %w", projectID, sessionID, err)
		}
		if err := validateArtifactRefs(event, artifactIDs); err != nil {
			return fmt.Errorf("project %q session %q event %q %w", projectID, sessionID, event.ID, err)
		}
	}
	return nil
}

func validateArtifacts(projectID string, sessionID string, artifacts []Artifact) (map[string]bool, error) {
	artifactIDs := map[string]bool{}
	for i, artifact := range artifacts {
		if strings.TrimSpace(artifact.ID) == "" {
			return nil, fmt.Errorf("project %q session %q artifact %d id is required", projectID, sessionID, i)
		}
		if artifactIDs[artifact.ID] {
			return nil, fmt.Errorf("project %q session %q duplicate artifact id %q", projectID, sessionID, artifact.ID)
		}
		artifactIDs[artifact.ID] = true
		if !SupportedArtifactTypes[artifact.Type] {
			return nil, fmt.Errorf("project %q session %q artifact %q unsupported type %q", projectID, sessionID, artifact.ID, artifact.Type)
		}
		if strings.TrimSpace(artifact.ContentHash) == "" {
			return nil, fmt.Errorf("project %q session %q artifact %q content_hash is required", projectID, sessionID, artifact.ID)
		}
		if err := validateMetadata("artifact", artifact.ID, artifact.Metadata); err != nil {
			return nil, fmt.Errorf("project %q session %q %w", projectID, sessionID, err)
		}
	}
	return artifactIDs, nil
}

func validateArtifactRefs(event Event, artifactIDs map[string]bool) error {
	for i, ref := range event.ArtifactRefs {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			return fmt.Errorf("artifact_refs[%d] is empty", i)
		}
		if !artifactIDs[ref] {
			return fmt.Errorf("artifact ref %q does not exist in session artifacts", ref)
		}
	}
	return nil
}

func validateMetadata(kind string, id string, metadata map[string]string) error {
	for key := range metadata {
		normalized := strings.ToLower(strings.TrimSpace(key))
		if reservedMetadataKeys[normalized] {
			return fmt.Errorf("%s %q metadata key %q is reserved for raw content and must be stored as a hashed artifact reference", kind, id, key)
		}
	}
	return nil
}

func validateRFC3339(field string, value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s is required", field)
	}
	if _, err := time.Parse(time.RFC3339, value); err != nil {
		return fmt.Errorf("%s must be RFC3339", field)
	}
	return nil
}

func Summarize(l Ledger) Summary {
	summary := Summary{Projects: len(l.Projects)}
	for _, project := range l.Projects {
		summary.Sessions += len(project.Sessions)
		for _, session := range project.Sessions {
			summary.Events += len(session.Events)
			summary.Artifacts += len(session.Artifacts)
		}
	}
	return summary
}
