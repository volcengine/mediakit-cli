package core

import (
	"io"

	"github.com/spf13/cobra"

	"mediakit-cli/internal/output"
)

// Handler executes a local capability and returns a JSON-compatible result.
type Handler interface {
	Execute(ctx *ExecContext) (map[string]any, error)
}

// HandlerFunc adapts a function into a local handler.
type HandlerFunc func(ctx *ExecContext) (map[string]any, error)

func (f HandlerFunc) Execute(ctx *ExecContext) (map[string]any, error) {
	return f(ctx)
}

// Registration describes a local handler exposed by generated or manual code.
type Registration struct {
	Command      string
	Source       string
	Dependencies []string
	Handler      Handler
}

// ExecContext carries runtime state for local execution.
type ExecContext struct {
	Command   string
	Params    map[string]any
	WorkDir   string
	TempDir   string
	OutputDir string

	CommandIO *cobra.Command
	Writer    output.Writer
	Limits    ResourceLimits
}

// ResourceLimits carries resource guardrails reserved for local execution.
type ResourceLimits struct {
	MaxMemoryBytes    int64
	MaxDiskBytes      int64
	MaxChildProcesses int
}

func DefaultResourceLimits() ResourceLimits {
	return ResourceLimits{
		MaxMemoryBytes:    2 << 30,
		MaxDiskBytes:      10 << 30,
		MaxChildProcesses: 4,
	}
}

func (c *ExecContext) Stdout() io.Writer {
	if c.CommandIO == nil {
		return io.Discard
	}
	return c.CommandIO.OutOrStdout()
}

func (c *ExecContext) Stderr() io.Writer {
	if c.CommandIO == nil {
		return io.Discard
	}
	return c.CommandIO.ErrOrStderr()
}
