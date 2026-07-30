// Package app implements the CLI dispatcher. Phase 0 handles read-only
// commands: version, health, context, get, list, and search. Phase 1 adds
// submit and status commands. Phase 2 adds apply, changes, and recover.
package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ryannmicua/hq-cli/internal/assets"
	"github.com/ryannmicua/hq-cli/internal/config"
	"github.com/ryannmicua/hq-cli/internal/contract"
	"github.com/ryannmicua/hq-cli/internal/fsx"
	"github.com/ryannmicua/hq-cli/internal/read"
	"github.com/ryannmicua/hq-cli/internal/write"
)

// Run parses args, dispatches to the appropriate command handler, writes
// the JSON result to stdout, and returns the exit code.
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = stderr.Write([]byte("hq: usage: hq [--root <path>] [--output json] <command> [arguments]\n"))
		return 2
	}

	// Parse global flags and extract command + args.
	rootFlag := ""
	var remaining []string

	// Find the command position (first non-flag, non-value argument).
	i := 0
	for i < len(args) {
		if args[i] == "--root" {
			if i+1 < len(args) {
				rootFlag = args[i+1]
				i += 2
			} else {
				_, _ = stderr.Write([]byte("hq: --root requires a path argument\n"))
				return 2
			}
		} else if args[i] == "--output" {
			i += 2 // consume --output and its value (ignored in Phase 0)
		} else if strings.HasPrefix(args[i], "--") {
			// Skip other flags.
			i++
		} else {
			// This is the command.
			remaining = args[i:]
			break
		}
	}

	if len(remaining) == 0 {
		_, _ = stderr.Write([]byte("hq: missing command\n"))
		return 2
	}

	cmd := remaining[0]
	cmdArgs := remaining[1:]

	// Resolve config.
	cfg, err := config.Load(rootFlag, envLookup)
	if err != nil {
		errDetail := contract.ErrDetail(contract.CodeInternalError, fmt.Sprintf("config: %v", err), false, nil)
		r := contract.NewError(cmd, errDetail)
		_ = contract.WriteJSON(stdout, r)
		return contract.ExitCode(errDetail.Code)
	}

	// Create read service.
	svc, err := read.NewService(cfg)
	if err != nil {
		errDetail := contract.ErrDetail(contract.CodeInternalError, fmt.Sprintf("init: %v", err), false, nil)
		r := contract.NewError(cmd, errDetail)
		_ = contract.WriteJSON(stdout, r)
		return contract.ExitCode(errDetail.Code)
	}

	// Create write service for submit/status/apply/changes/recover commands.
	policy, _ := write.ParsePolicyJSON(assets.PolicyV1)
	f := fsx.NewFS()
	writeSvc := NewWriteService(cfg.Root, policy)
	receipts := write.NewReceiptStore(f, cfg.Root)
	store := write.NewRequestStore(f, cfg.Root, policy)
	engine := write.NewTransactionEngine(f, cfg.Root, store, receipts, policy)
	changesSvc := write.NewChangesService(f, cfg.Root, receipts)
	recoverySvc := write.NewRecoveryService(f, cfg.Root, store, receipts, policy)

	return dispatchCommand(cmd, cmdArgs, svc, writeSvc, engine, changesSvc, recoverySvc, stdout, stderr)
}

// envLookup wraps os.LookupEnv for the config package.
var envLookup = os.LookupEnv

func dispatchCommand(cmd string, args []string, svc *read.Service, writeSvc *WriteService, engine *write.TransactionEngine, changesSvc *write.ChangesService, recoverySvc *write.RecoveryService, stdout, stderr io.Writer) int {
	ctx := context.Background()

	switch cmd {
	case "version":
		return handleVersion(ctx, args, svc, stdout, stderr)
	case "health":
		return handleHealth(ctx, args, svc, stdout, stderr)
	case "context":
		return handleContext(ctx, args, svc, stdout, stderr)
	case "get":
		return handleGet(ctx, args, svc, stdout, stderr)
	case "list":
		return handleList(ctx, args, svc, stdout, stderr)
	case "search":
		return handleSearch(ctx, args, svc, stdout, stderr)
	case "submit":
		return handleSubmit(ctx, args, writeSvc, stdout, stderr)
	case "status":
		return handleStatus(ctx, args, writeSvc, stdout, stderr)
	case "apply":
		return handleApply(ctx, args, writeSvc, engine, stdout, stderr)
	case "changes":
		return handleChanges(ctx, args, changesSvc, writeSvc, stdout, stderr)
	case "recover":
		return handleRecover(ctx, args, recoverySvc, stdout, stderr)
	default:
		_, _ = stderr.Write([]byte(fmt.Sprintf("hq: unknown command: %s\n", cmd)))
		_ = contract.WriteJSON(stdout, contract.NewError(cmd, contract.ErrDetail(contract.CodeInvalidArgument, fmt.Sprintf("unknown command: %s", cmd), false, nil)))
		return 2
	}
}

// parseFlags extracts --key value pairs from args. Consumed flags are
// removed from the returned slice.
func parseFlags(args []string) (map[string]string, []string) {
	flags := make(map[string]string)
	var positional []string
	for i := 0; i < len(args); i++ {
		if strings.HasPrefix(args[i], "--") {
			key := strings.TrimPrefix(args[i], "--")
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
				flags[key] = args[i+1]
				i++
			} else {
				flags[key] = ""
			}
		} else {
			positional = append(positional, args[i])
		}
	}
	return flags, positional
}
