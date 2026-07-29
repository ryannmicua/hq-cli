package write_test

import (
	"testing"

	"github.com/ryannmicua/hq-cli/internal/contract"
	"github.com/ryannmicua/hq-cli/internal/fsx"
	"github.com/ryannmicua/hq-cli/internal/write"
)

func TestChangesAfter_NoReceipts(t *testing.T) {
	root := t.TempDir()
	f := fsx.NewFS()
	receipts := write.NewReceiptStore(f, root)
	svc := write.NewChangesService(f, root, receipts)

	page, err := svc.After(0, 10)
	if err != nil {
		t.Fatalf("After failed: %v", err)
	}

	if len(page.Receipts) != 0 {
		t.Fatalf("expected 0 receipts, got %d", len(page.Receipts))
	}
}

func TestChangesAfter_ReturnsOrdered(t *testing.T) {
	root := t.TempDir()
	f := fsx.NewFS()
	receipts := write.NewReceiptStore(f, root)
	svc := write.NewChangesService(f, root, receipts)

	for i := uint64(1); i <= 5; i++ {
		r := contract.Receipt{
			SchemaVersion:  "1.0",
			Cursor:         i,
			RequestID:      "",
			Target:         "",
			TargetSha256:   "",
			RenderedSha256: "",
			AppliedAt:      "",
		}
		if err := receipts.Store(r); err != nil {
			t.Fatalf("store receipt %d: %v", i, err)
		}
	}

	page, err := svc.After(0, 10)
	if err != nil {
		t.Fatalf("After failed: %v", err)
	}

	if len(page.Receipts) != 5 {
		t.Fatalf("expected 5 receipts, got %d", len(page.Receipts))
	}

	for i, r := range page.Receipts {
		if r.Cursor != uint64(i+1) {
			t.Fatalf("receipt %d cursor = %d, want %d", i, r.Cursor, i+1)
		}
	}
}

func TestChangesAfter_Limit(t *testing.T) {
	root := t.TempDir()
	f := fsx.NewFS()
	receipts := write.NewReceiptStore(f, root)
	svc := write.NewChangesService(f, root, receipts)

	for i := uint64(1); i <= 10; i++ {
		r := contract.Receipt{SchemaVersion: "1.0", Cursor: i}
		if err := receipts.Store(r); err != nil {
			t.Fatalf("store receipt %d: %v", i, err)
		}
	}

	page, err := svc.After(0, 3)
	if err != nil {
		t.Fatalf("After failed: %v", err)
	}

	if len(page.Receipts) != 3 {
		t.Fatalf("expected 3 receipts, got %d", len(page.Receipts))
	}

	if !page.HasMore {
		t.Fatal("expected HasMore to be true")
	}
}

func TestChangesAfter_CursorFilter(t *testing.T) {
	root := t.TempDir()
	f := fsx.NewFS()
	receipts := write.NewReceiptStore(f, root)
	svc := write.NewChangesService(f, root, receipts)

	for i := uint64(1); i <= 5; i++ {
		r := contract.Receipt{SchemaVersion: "1.0", Cursor: i}
		if err := receipts.Store(r); err != nil {
			t.Fatalf("store receipt %d: %v", i, err)
		}
	}

	page, err := svc.After(2, 10)
	if err != nil {
		t.Fatalf("After failed: %v", err)
	}

	if len(page.Receipts) != 3 {
		t.Fatalf("expected 3 receipts (cursor 3,4,5), got %d", len(page.Receipts))
	}

	expected := []uint64{3, 4, 5}
	for i, r := range page.Receipts {
		if r.Cursor != expected[i] {
			t.Fatalf("receipt %d cursor = %d, want %d", i, r.Cursor, expected[i])
		}
	}
}
