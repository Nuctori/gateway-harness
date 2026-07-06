package ledger

import "sort"

type QueryOptions struct {
	ProjectID  string
	SessionID  string
	Tags       []string
	EventTypes []string
}

type QueryResult struct {
	ProjectID     string         `json:"project_id"`
	ProjectName   string         `json:"project_name,omitempty"`
	SessionID     string         `json:"session_id"`
	SessionTitle  string         `json:"session_title,omitempty"`
	StartedAt     string         `json:"started_at"`
	Tags          []string       `json:"tags,omitempty"`
	Events        int            `json:"events"`
	Artifacts     int            `json:"artifacts"`
	EventCounts   map[string]int `json:"event_counts,omitempty"`
	ArtifactRefs  []string       `json:"artifact_refs,omitempty"`
	MatchedEvents []string       `json:"matched_events,omitempty"`
}

func Query(l Ledger, options QueryOptions) []QueryResult {
	results := []QueryResult{}
	for _, project := range l.Projects {
		if options.ProjectID != "" && project.ID != options.ProjectID {
			continue
		}
		for _, session := range project.Sessions {
			if options.SessionID != "" && session.ID != options.SessionID {
				continue
			}
			tags := mergeTags(project.Tags, session.Tags)
			if !hasAllTags(tags, options.Tags) {
				continue
			}
			eventCounts, matchedEvents := countEvents(session.Events, options.EventTypes)
			if len(options.EventTypes) > 0 && len(matchedEvents) == 0 {
				continue
			}
			results = append(results, QueryResult{
				ProjectID:     project.ID,
				ProjectName:   project.Name,
				SessionID:     session.ID,
				SessionTitle:  session.Title,
				StartedAt:     session.StartedAt,
				Tags:          tags,
				Events:        len(session.Events),
				Artifacts:     len(session.Artifacts),
				EventCounts:   eventCounts,
				ArtifactRefs:  artifactRefs(session.Artifacts),
				MatchedEvents: matchedEvents,
			})
		}
	}
	return results
}

func mergeTags(projectTags []string, sessionTags []string) []string {
	seen := map[string]bool{}
	tags := make([]string, 0, len(projectTags)+len(sessionTags))
	for _, tag := range append(append([]string{}, projectTags...), sessionTags...) {
		if tag == "" || seen[tag] {
			continue
		}
		seen[tag] = true
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	return tags
}

func hasAllTags(tags []string, required []string) bool {
	if len(required) == 0 {
		return true
	}
	seen := map[string]bool{}
	for _, tag := range tags {
		seen[tag] = true
	}
	for _, tag := range required {
		if tag == "" {
			continue
		}
		if !seen[tag] {
			return false
		}
	}
	return true
}

func countEvents(events []Event, eventTypes []string) (map[string]int, []string) {
	counts := map[string]int{}
	matches := []string{}
	filter := map[string]bool{}
	for _, eventType := range eventTypes {
		if eventType != "" {
			filter[eventType] = true
		}
	}
	for _, event := range events {
		counts[event.Type]++
		if len(filter) == 0 || filter[event.Type] {
			matches = append(matches, event.ID)
		}
	}
	return counts, matches
}

func artifactRefs(artifacts []Artifact) []string {
	refs := make([]string, 0, len(artifacts))
	for _, artifact := range artifacts {
		refs = append(refs, artifact.ID)
	}
	sort.Strings(refs)
	return refs
}
