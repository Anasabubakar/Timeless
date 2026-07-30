package mapping

import (
	"context"
	"fmt"
	"time"

	"github.com/timeless/backend/internal/integration"
	"github.com/timeless/backend/internal/models"
)

// NotionAdapter implements Adapter for Notion, driven entirely by a
// FieldMapping's per-field ExternalType — no entity-specific logic
// lives here, only "how do I write/read a Notion 'select' property"
// style translation, so this adapter works for arbitrary user
// databases (Companies, Contacts, Meetings, Tasks, anything) without
// per-entity-type code.
type NotionAdapter struct {
	client *integration.NotionClient
}

func NewNotionAdapter(client *integration.NotionClient) *NotionAdapter {
	return &NotionAdapter{client: client}
}

func (a *NotionAdapter) System() string { return "notion" }

func (a *NotionAdapter) ToExternal(record SyncableRecord, fm *models.FieldMapping) (map[string]interface{}, error) {
	entries, err := ParseFields(fm)
	if err != nil {
		return nil, err
	}
	out := make(map[string]interface{}, len(entries))
	for _, e := range entriesForDirection(entries, "to_external") {
		value, ok := record.Fields[e.InternalField]
		if !ok || value == nil {
			continue
		}
		prop, err := serializeNotionProperty(e.ExternalType, value)
		if err != nil {
			return nil, fmt.Errorf("field %q: %w", e.InternalField, err)
		}
		out[e.ExternalField] = prop
	}
	return out, nil
}

func (a *NotionAdapter) FromExternal(raw map[string]interface{}, fm *models.FieldMapping) (map[string]interface{}, error) {
	entries, err := ParseFields(fm)
	if err != nil {
		return nil, err
	}
	out := make(map[string]interface{}, len(entries))
	for _, e := range entriesForDirection(entries, "from_external") {
		rawProp, ok := raw[e.ExternalField]
		if !ok {
			continue
		}
		propMap, ok := rawProp.(map[string]interface{})
		if !ok {
			continue
		}
		value, ok := deserializeNotionProperty(e.ExternalType, propMap)
		if ok {
			out[e.InternalField] = value
		}
	}
	return out, nil
}

func (a *NotionAdapter) Push(ctx context.Context, credentials map[string]string, containerID, existingExternalID string, properties map[string]interface{}, expectedRemoteVersion string) (string, error) {
	if existingExternalID == "" {
		return a.client.CreatePage(ctx, credentials, containerID, properties)
	}
	if err := a.client.UpdatePageProperties(ctx, credentials, existingExternalID, properties, expectedRemoteVersion); err != nil {
		return "", err
	}
	return existingExternalID, nil
}

func (a *NotionAdapter) Archive(ctx context.Context, credentials map[string]string, externalID string) error {
	return a.client.ArchivePage(ctx, credentials, externalID)
}

func (a *NotionAdapter) Fetch(ctx context.Context, credentials map[string]string, containerID, externalID string) (*ExternalRecord, error) {
	page, err := a.client.FetchPage(ctx, credentials, externalID)
	if err != nil {
		return nil, err
	}
	props, _ := page["properties"].(map[string]interface{})
	rec := &ExternalRecord{ContainerID: containerID, ExternalID: externalID, RawProperties: props}
	if editedRaw, ok := page["last_edited_time"].(string); ok && editedRaw != "" {
		if t, err := time.Parse(time.RFC3339, editedRaw); err == nil {
			rec.LastModified = &t
		}
	}
	return rec, nil
}

// serializeNotionProperty builds one Notion property value from a plain
// Go value, per RFC: developers.notion.com/reference/property-value-object
func serializeNotionProperty(propType string, value interface{}) (map[string]interface{}, error) {
	switch propType {
	case "title":
		return map[string]interface{}{"title": richTextArray(toStr(value))}, nil
	case "rich_text":
		return map[string]interface{}{"rich_text": richTextArray(toStr(value))}, nil
	case "select":
		s := toStr(value)
		if s == "" {
			return map[string]interface{}{"select": nil}, nil
		}
		return map[string]interface{}{"select": map[string]interface{}{"name": s}}, nil
	case "status":
		s := toStr(value)
		if s == "" {
			return map[string]interface{}{"status": nil}, nil
		}
		return map[string]interface{}{"status": map[string]interface{}{"name": s}}, nil
	case "multi_select":
		names := toStrSlice(value)
		opts := make([]map[string]interface{}, len(names))
		for i, n := range names {
			opts[i] = map[string]interface{}{"name": n}
		}
		return map[string]interface{}{"multi_select": opts}, nil
	case "date":
		t, ok := toTime(value)
		if !ok {
			return map[string]interface{}{"date": nil}, nil
		}
		return map[string]interface{}{"date": map[string]interface{}{"start": t.Format("2006-01-02")}}, nil
	case "number":
		n, ok := toFloat(value)
		if !ok {
			return map[string]interface{}{"number": nil}, nil
		}
		return map[string]interface{}{"number": n}, nil
	case "checkbox":
		b, _ := value.(bool)
		return map[string]interface{}{"checkbox": b}, nil
	case "url":
		return map[string]interface{}{"url": nilIfEmpty(toStr(value))}, nil
	case "email":
		return map[string]interface{}{"email": nilIfEmpty(toStr(value))}, nil
	case "phone_number":
		return map[string]interface{}{"phone_number": nilIfEmpty(toStr(value))}, nil
	case "relation":
		ids := toStrSlice(value)
		rel := make([]map[string]interface{}, len(ids))
		for i, id := range ids {
			rel[i] = map[string]interface{}{"id": id}
		}
		return map[string]interface{}{"relation": rel}, nil
	case "people":
		ids := toStrSlice(value)
		people := make([]map[string]interface{}, len(ids))
		for i, id := range ids {
			people[i] = map[string]interface{}{"id": id}
		}
		return map[string]interface{}{"people": people}, nil
	default:
		return nil, fmt.Errorf("unsupported notion property type %q", propType)
	}
}

// deserializeNotionProperty is the inverse: given a Notion property
// value object (as returned by the API), extract a plain Go value.
// ok=false means "nothing meaningful to extract" (empty/unset), not an
// error — an empty Notion property is a normal, common state.
func deserializeNotionProperty(propType string, prop map[string]interface{}) (interface{}, bool) {
	switch propType {
	case "title":
		return richTextArrayPlain(prop["title"])
	case "rich_text":
		return richTextArrayPlain(prop["rich_text"])
	case "select":
		if sel, ok := prop["select"].(map[string]interface{}); ok {
			if name, ok := sel["name"].(string); ok {
				return name, true
			}
		}
		return nil, false
	case "status":
		if st, ok := prop["status"].(map[string]interface{}); ok {
			if name, ok := st["name"].(string); ok {
				return name, true
			}
		}
		return nil, false
	case "multi_select":
		items, ok := prop["multi_select"].([]interface{})
		if !ok {
			return nil, false
		}
		names := make([]string, 0, len(items))
		for _, it := range items {
			if m, ok := it.(map[string]interface{}); ok {
				if name, ok := m["name"].(string); ok {
					names = append(names, name)
				}
			}
		}
		return names, true
	case "date":
		if d, ok := prop["date"].(map[string]interface{}); ok {
			if start, ok := d["start"].(string); ok && start != "" {
				return start, true
			}
		}
		return nil, false
	case "number":
		if n, ok := prop["number"].(float64); ok {
			return n, true
		}
		return nil, false
	case "checkbox":
		if b, ok := prop["checkbox"].(bool); ok {
			return b, true
		}
		return nil, false
	case "url", "email", "phone_number":
		if s, ok := prop[propType].(string); ok && s != "" {
			return s, true
		}
		return nil, false
	case "relation":
		items, ok := prop["relation"].([]interface{})
		if !ok {
			return nil, false
		}
		ids := make([]string, 0, len(items))
		for _, it := range items {
			if m, ok := it.(map[string]interface{}); ok {
				if id, ok := m["id"].(string); ok {
					ids = append(ids, id)
				}
			}
		}
		return ids, true
	case "people":
		items, ok := prop["people"].([]interface{})
		if !ok {
			return nil, false
		}
		ids := make([]string, 0, len(items))
		for _, it := range items {
			if m, ok := it.(map[string]interface{}); ok {
				if id, ok := m["id"].(string); ok {
					ids = append(ids, id)
				}
			}
		}
		return ids, true
	default:
		return nil, false
	}
}

func richTextArray(s string) []map[string]interface{} {
	if s == "" {
		return []map[string]interface{}{}
	}
	return []map[string]interface{}{
		{"type": "text", "text": map[string]interface{}{"content": s}},
	}
}

func richTextArrayPlain(raw interface{}) (string, bool) {
	items, ok := raw.([]interface{})
	if !ok || len(items) == 0 {
		return "", false
	}
	var out string
	for _, it := range items {
		m, ok := it.(map[string]interface{})
		if !ok {
			continue
		}
		if plain, ok := m["plain_text"].(string); ok {
			out += plain
			continue
		}
		if text, ok := m["text"].(map[string]interface{}); ok {
			if content, ok := text["content"].(string); ok {
				out += content
			}
		}
	}
	return out, out != ""
}

func toStr(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	case *string:
		if t == nil {
			return ""
		}
		return *t
	case fmt.Stringer:
		return t.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

func toStrSlice(v interface{}) []string {
	switch t := v.(type) {
	case []string:
		return t
	case []interface{}:
		out := make([]string, 0, len(t))
		for _, item := range t {
			out = append(out, toStr(item))
		}
		return out
	default:
		return nil
	}
}

func toFloat(v interface{}) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case float32:
		return float64(t), true
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	case *float64:
		if t == nil {
			return 0, false
		}
		return *t, true
	default:
		return 0, false
	}
}

func toTime(v interface{}) (time.Time, bool) {
	switch t := v.(type) {
	case time.Time:
		return t, true
	case *time.Time:
		if t == nil {
			return time.Time{}, false
		}
		return *t, true
	case string:
		if t == "" {
			return time.Time{}, false
		}
		parsed, err := time.Parse(time.RFC3339, t)
		if err != nil {
			parsed, err = time.Parse("2006-01-02", t)
			if err != nil {
				return time.Time{}, false
			}
		}
		return parsed, true
	default:
		return time.Time{}, false
	}
}

func nilIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
