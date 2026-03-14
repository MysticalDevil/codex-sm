package scanner

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"regexp"
	"strings"
)

var idInFilenameRe = regexp.MustCompile(`([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})\\.jsonl$`)

type metaLine struct {
	Type    string `json:"type"`
	Payload struct {
		ID        string `json:"id"`
		Timestamp string `json:"timestamp"`
		Cwd       string `json:"cwd"`
	} `json:"payload"`
}

type responseItemLine struct {
	Type    string `json:"type"`
	Payload struct {
		Type    string `json:"type"`
		Role    string `json:"role"`
		Text    string `json:"text"`
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	} `json:"payload"`
}

func conversationHeadFromLine(line []byte) string {
	var item responseItemLine

	if !jsontext.Value(line).IsValid() {
		return ""
	}

	if err := json.Unmarshal(line, &item); err != nil {
		return ""
	}

	if item.Type != "response_item" {
		return ""
	}

	if item.Payload.Type != "message" || item.Payload.Role != "user" {
		return ""
	}

	for _, c := range item.Payload.Content {
		if v := compactText(c.Text); v != "" {
			return v
		}
	}

	return compactText(item.Payload.Text)
}

func compactText(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}

	return strings.Join(strings.Fields(v), " ")
}

func sessionIDFromFilename(base string) string {
	m := idInFilenameRe.FindStringSubmatch(strings.ToLower(base))
	if len(m) != 2 {
		return ""
	}

	return m[1]
}
