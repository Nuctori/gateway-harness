package schema

import "github.com/Nuctori/gateway-harness/hooks"

func HookCatalog() []map[string]any {
	entries := hooks.Catalog()
	out := make([]map[string]any, 0, len(entries))
	for _, entry := range entries {
		out = append(out, hookCatalogEntry(entry))
	}
	return out
}

func goalGateHookOptions() []map[string]any {
	entries := hooks.GoalGateSelectable()
	out := make([]map[string]any, 0, len(entries))
	for _, entry := range entries {
		en := localizedHook(entry, "en-US")
		zh := localizedHook(entry, "zh-CN")
		out = append(out, map[string]any{
			"value":       entry.ID,
			"title":       en.Title,
			"description": en.Summary,
			"labels": map[string]string{
				"zh-CN": zh.Title,
				"en-US": en.Title,
			},
			"descriptions": map[string]string{
				"zh-CN": zh.Summary,
				"en-US": en.Summary,
			},
			"example": en.Example,
			"badges":  []string{entry.Family, entry.Kind},
		})
	}
	return out
}

func hookCatalogEntry(entry hooks.CatalogEntry) map[string]any {
	return map[string]any{
		"id":                       entry.ID,
		"family":                   entry.Family,
		"kind":                     entry.Kind,
		"supports_request_payload": entry.SupportsRequestPayload,
		"goal_gate_selectable":     entry.GoalGateSelectable,
		"host_surfaces":            append([]string(nil), entry.HostSurfaces...),
		"locales": map[string]any{
			"zh-CN": map[string]any{
				"title":   localizedHook(entry, "zh-CN").Title,
				"summary": localizedHook(entry, "zh-CN").Summary,
				"example": localizedHook(entry, "zh-CN").Example,
			},
			"en-US": map[string]any{
				"title":   localizedHook(entry, "en-US").Title,
				"summary": localizedHook(entry, "en-US").Summary,
				"example": localizedHook(entry, "en-US").Example,
			},
		},
	}
}

func localizedHook(entry hooks.CatalogEntry, locale string) hooks.LocaleText {
	if text, ok := entry.Locales[locale]; ok {
		return text
	}
	if text, ok := entry.Locales["en-US"]; ok {
		return text
	}
	for _, text := range entry.Locales {
		return text
	}
	return hooks.LocaleText{}
}
