package write

import (
	"context"
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/ryannmicua/hq-cli/internal/contract"
)

type SessionEntryRenderer struct{}

func NewSessionEntryRenderer() *SessionEntryRenderer {
	return &SessionEntryRenderer{}
}

var newlinePattern = regexp.MustCompile(`\r?\n`)

func (r *SessionEntryRenderer) Render(ctx context.Context, req contract.Request, current []byte) (RenderedTarget, error) {
	var entry contract.SessionEntry
	if err := contract.DecodeStrict(req.Payload, &entry); err != nil {
		return RenderedTarget{}, fmt.Errorf("unmarshal session-entry payload: %w", err)
	}

	if entry.Summary == "" {
		return RenderedTarget{}, fmt.Errorf("invalid session-entry: summary is required")
	}

	if newlinePattern.MatchString(entry.Summary) {
		return RenderedTarget{}, fmt.Errorf("invalid session-entry: summary must not contain newlines")
	}

	if entry.Timestamp == "" {
		return RenderedTarget{}, fmt.Errorf("invalid session-entry: timestamp is required")
	}

	ts, err := time.Parse(time.RFC3339, entry.Timestamp)
	if err != nil {
		return RenderedTarget{}, fmt.Errorf("invalid session-entry timestamp: %w", err)
	}

	for _, tag := range entry.Tags {
		if !tagPattern.MatchString(tag) {
			return RenderedTarget{}, fmt.Errorf("invalid session-entry tag %q: must match ^[a-z0-9][a-z0-9-]*$", tag)
		}
	}

	wallClock := ts.Format("2006-01-02 15:04")

	tagStr := ""
	for _, tag := range entry.Tags {
		tagStr += fmt.Sprintf(" #%s", tag)
	}

	newEntry := fmt.Sprintf("%s%s %s", wallClock, tagStr, entry.Summary)

	currentStr := string(current)
	sepIndex := strings.LastIndex(currentStr, "\n---")
	if sepIndex == -1 {
		return RenderedTarget{}, fmt.Errorf("SESSION-LOG.md does not contain separator '---'")
	}

	before := currentStr[:sepIndex]
	after := currentStr[sepIndex+4:]

	result := before + "\n---\n\n" + newEntry + "\n" + strings.TrimLeft(after, "\n") + "\n"

	out := []byte(result)
	hash := fmt.Sprintf("%x", sha256.Sum256(out))

	return RenderedTarget{
		Path:    req.Target,
		Content: out,
		SHA256:  hash,
	}, nil
}

var tagPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)
