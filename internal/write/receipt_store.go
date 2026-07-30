package write

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ryannmicua/hq-cli/internal/contract"
	"github.com/ryannmicua/hq-cli/internal/fsx"
)

func sha256Sum(data []byte) []byte {
	h := sha256.Sum256(data)
	return h[:]
}

type ReceiptStore struct {
	fsys fsx.FS
	root string
}

func NewReceiptStore(fsys fsx.FS, root string) *ReceiptStore {
	return &ReceiptStore{fsys: fsys, root: root}
}

func receiptsDir(root string) string {
	return filepath.Join(root, ".hq-interface", "receipts")
}

func (rs *ReceiptStore) Store(receipt contract.Receipt) error {
	dir := receiptsDir(rs.root)
	if err := rs.fsys.MkdirPrivate(dir, 0700); err != nil {
		return fmt.Errorf("create receipts dir: %w", err)
	}

	data, err := json.Marshal(receipt)
	if err != nil {
		return fmt.Errorf("marshal receipt: %w", err)
	}

	filename := fmt.Sprintf("%020d.json", receipt.Cursor)
	dstPath := filepath.Join(dir, filename)

	tmpDir := filepath.Join(rs.root, ".hq-interface", "tmp")
	tmpName := fmt.Sprintf("receipt-%020d.tmp", receipt.Cursor)
	tmpPath, err := rs.fsys.WriteDurable(tmpDir, tmpName, data)
	if err != nil {
		return fmt.Errorf("write receipt temp: %w", err)
	}

	if err := rs.fsys.RenameAtomic(tmpPath, dstPath); err != nil {
		return fmt.Errorf("rename receipt: %w", err)
	}

	if err := rs.fsys.SyncParent(dir); err != nil {
		return fmt.Errorf("sync receipt dir: %w", err)
	}

	return nil
}

func (rs *ReceiptStore) FindByRequestID(requestID string) (*contract.Receipt, error) {
	dir := receiptsDir(rs.root)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("receipt for request %q not found", requestID)
		}
		return nil, fmt.Errorf("read receipts dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}

		var r contract.Receipt
		if err := json.Unmarshal(data, &r); err != nil {
			continue
		}

		if r.RequestID == requestID {
			return &r, nil
		}
	}

	return nil, fmt.Errorf("receipt for request %q not found", requestID)
}

func (rs *ReceiptStore) FindByCursor(cursor uint64) (*contract.Receipt, error) {
	dir := receiptsDir(rs.root)
	filename := fmt.Sprintf("%020d.json", cursor)
	path := filepath.Join(dir, filename)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read receipt at cursor %d: %w", cursor, err)
	}

	var r contract.Receipt
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("unmarshal receipt at cursor %d: %w", cursor, err)
	}

	return &r, nil
}

func (rs *ReceiptStore) ListAll() ([]contract.Receipt, error) {
	dir := receiptsDir(rs.root)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read receipts dir: %w", err)
	}

	var cursors []uint64
	cursorMap := make(map[uint64]string)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".json")
		c, err := strconv.ParseUint(name, 10, 64)
		if err != nil {
			continue
		}
		if _, exists := cursorMap[c]; exists {
			log.Printf("warning: duplicate cursor %d in receipt directory, overwriting %s with %s", c, cursorMap[c], filepath.Join(dir, entry.Name()))
		}
		cursors = append(cursors, c)
		cursorMap[c] = filepath.Join(dir, entry.Name())
	}

	sort.Slice(cursors, func(i, j int) bool {
		return cursors[i] < cursors[j]
	})

	var receipts []contract.Receipt
	var skipped int
	for _, c := range cursors {
		data, err := os.ReadFile(cursorMap[c])
		if err != nil {
			skipped++
			continue
		}
		var r contract.Receipt
		if err := json.Unmarshal(data, &r); err != nil {
			skipped++
			continue
		}
		receipts = append(receipts, r)
	}

	if skipped > 0 && len(receipts) == 0 {
		return nil, fmt.Errorf("list all: %d receipt files could not be read or parsed", skipped)
	}

	return receipts, nil
}

func (rs *ReceiptStore) ListAfter(cursor uint64, limit int) ([]contract.Receipt, uint64, error) {
	dir := receiptsDir(rs.root)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, 0, nil
		}
		return nil, 0, fmt.Errorf("read receipts dir: %w", err)
	}

	var cursors []uint64
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".json")
		c, err := strconv.ParseUint(name, 10, 64)
		if err != nil {
			continue
		}
		if c > cursor {
			cursors = append(cursors, c)
		}
	}

	sort.Slice(cursors, func(i, j int) bool {
		return cursors[i] < cursors[j]
	})

	if limit > 0 && len(cursors) > limit {
		cursors = cursors[:limit]
	}

	var receipts []contract.Receipt
	for _, c := range cursors {
		r, err := rs.FindByCursor(c)
		if err != nil {
			continue
		}
		receipts = append(receipts, *r)
	}

	nextCursor := uint64(0)
	if len(cursors) > 0 {
		nextCursor = cursors[len(cursors)-1]
	}

	return receipts, nextCursor, nil
}

func (rs *ReceiptStore) NextCursor() (uint64, error) {
	dir := receiptsDir(rs.root)
	if err := rs.fsys.MkdirPrivate(dir, 0700); err != nil {
		return 0, fmt.Errorf("create receipts dir: %w", err)
	}

	lockDir := filepath.Join(dir, ".cursor.lock")
	counterPath := filepath.Join(dir, ".cursor")

	if err := os.Mkdir(lockDir, 0700); err != nil {
		if !os.IsExist(err) {
			return 0, fmt.Errorf("create cursor lock: %w", err)
		}
		for i := 0; i < 100; i++ {
			time.Sleep(10 * time.Millisecond)
			err = os.Mkdir(lockDir, 0700)
			if err == nil {
				break
			}
			if !os.IsExist(err) {
				return 0, fmt.Errorf("create cursor lock: %w", err)
			}
		}
		if err != nil {
			return 0, fmt.Errorf("cursor lock contended: %w", err)
		}
	}

	defer os.Remove(lockDir)

	data, err := os.ReadFile(counterPath)
	cursor := uint64(0)
	if err == nil {
		cursor, err = strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
		if err != nil {
			cursor = 0
		}
	} else if !os.IsNotExist(err) {
		return 0, fmt.Errorf("read cursor: %w", err)
	}

	next := cursor + 1

	tmpPath := counterPath + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(strconv.FormatUint(next, 10)+"\n"), 0600); err != nil {
		os.Remove(tmpPath)
		return 0, fmt.Errorf("write cursor: %w", err)
	}

	if err := os.Rename(tmpPath, counterPath); err != nil {
		os.Remove(tmpPath)
		return 0, fmt.Errorf("rename cursor: %w", err)
	}

	_ = os.Remove(lockDir)

	return next, nil
}

type IntentRecord struct {
	RequestID           string `json:"requestId"`
	Cursor              uint64 `json:"cursor"`
	Target              string `json:"target"`
	BackupPath          string `json:"backupPath"`
	PreHash             string `json:"preHash"`
	RenderedContentHash string `json:"renderedContentHash"`
	Timestamp           string `json:"timestamp"`
}

func intentsDir(root string) string {
	return filepath.Join(root, ".hq-interface", "intents")
}

func (rs *ReceiptStore) WriteIntent(intent IntentRecord) error {
	dir := intentsDir(rs.root)
	if err := rs.fsys.MkdirPrivate(dir, 0700); err != nil {
		return fmt.Errorf("create intents dir: %w", err)
	}

	data, err := json.Marshal(intent)
	if err != nil {
		return fmt.Errorf("marshal intent: %w", err)
	}

	path := filepath.Join(dir, intent.RequestID+".json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write intent file: %w", err)
	}

	return nil
}

func (rs *ReceiptStore) RemoveIntent(requestID string) error {
	path := filepath.Join(intentsDir(rs.root), requestID+".json")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove intent: %w", err)
	}
	return nil
}

func (rs *ReceiptStore) ReadIntent(requestID string) (*IntentRecord, error) {
	path := filepath.Join(intentsDir(rs.root), requestID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read intent: %w", err)
	}

	var intent IntentRecord
	if err := json.Unmarshal(data, &intent); err != nil {
		return nil, fmt.Errorf("unmarshal intent: %w", err)
	}

	return &intent, nil
}

func (rs *ReceiptStore) ReconcileOrphans() []string {
	var warnings []string
	intentDir := intentsDir(rs.root)
	entries, err := os.ReadDir(intentDir)
	if err != nil {
		return nil
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		requestID := strings.TrimSuffix(entry.Name(), ".json")

		intent, err := rs.ReadIntent(requestID)
		if err != nil {
			continue
		}

		_, err = rs.FindByRequestID(requestID)
		if err == nil {
			rs.RemoveIntent(requestID)
			continue
		}

		targetHash := sha256OfFile(intent.Target)

		if intent.RenderedContentHash != "" && targetHash == intent.RenderedContentHash {
			warnings = append(warnings, fmt.Sprintf("orphan intent %s: target matches rendered hash, but no receipt found", requestID))
			continue
		}

		if intent.PreHash != "" && targetHash == intent.PreHash {
			rs.RemoveIntent(requestID)
			continue
		}

		warnings = append(warnings, fmt.Sprintf("orphan intent %s: target hash %q does not match pre-hash %q or rendered %q; recovery may be needed",
			requestID, targetHash, intent.PreHash, intent.RenderedContentHash))
	}

	return warnings
}

func sha256OfFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%x", sha256Sum(data))
}
