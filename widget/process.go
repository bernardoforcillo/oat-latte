package widget

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	oat "github.com/antoniocali/oat-latte"
	"github.com/antoniocali/oat-latte/latte"
)

// Process is a widget that runs an OS command and streams its combined stdout
// and stderr output into a built-in LogViewer.
//
// Usage:
//
//	p := widget.NewProcess("kubectl", "logs", "-f", "my-pod").
//	    WithRedraw(canvas.Redraw).
//	    WithRestartOnExit(true)
//	p.Start()
type Process struct {
	oat.BaseComponent
	oat.FocusBehavior
	oat.BaseHitRegion

	// command config
	name string
	args []string
	dir  string
	env  []string // extra KEY=VALUE pairs appended to os.Environ(); nil = inherit only

	// runtime state (guarded by mu)
	mu      sync.Mutex
	cmd     *exec.Cmd
	running bool
	exitErr error

	// inner viewer
	viewer *LogViewer

	// options
	restartOnExit bool
	redrawFn      func()

	// styles
	statusStyle latte.Style
	callerStyle latte.Style
}

// NewProcess creates a Process widget for the given command and arguments.
// Call Start() to begin execution.
// Pass canvas.Redraw as the redraw function via WithRedraw so that output
// lines trigger a screen refresh.
func NewProcess(name string, args ...string) *Process {
	p := &Process{
		name:   name,
		args:   args,
		viewer: NewLogViewer().WithFollow(true),
	}
	p.EnsureID()
	return p
}

// WithID sets a user-defined identifier.
func (p *Process) WithID(id string) *Process { p.ID = id; return p }

// WithDir sets the working directory for the command.
func (p *Process) WithDir(dir string) *Process { p.dir = dir; return p }

// WithEnv appends extra environment variables in KEY=VALUE form.
// The command inherits the current process environment; these are added on top.
func (p *Process) WithEnv(env ...string) *Process { p.env = append(p.env, env...); return p }

// WithRestartOnExit automatically restarts the command when it exits.
func (p *Process) WithRestartOnExit(r bool) *Process { p.restartOnExit = r; return p }

// WithRedraw registers the function called after each output line arrives.
// Typically pass canvas.Redraw so the screen refreshes as output streams in.
func (p *Process) WithRedraw(fn func()) *Process {
	p.redrawFn = fn
	p.viewer.StreamFrom(nil, nil) // no-op; redrawFn is wired in Start
	return p
}

// WithMaxLines caps the LogViewer's line buffer.
func (p *Process) WithMaxLines(n int) *Process { p.viewer.WithMaxLines(n); return p }

// WithFollow controls whether the LogViewer auto-scrolls to new output.
func (p *Process) WithFollow(f bool) *Process { p.viewer.WithFollow(f); return p }

// ── control ──────────────────────────────────────────────────────────────────

// Start launches the command. No-op if already running.
func (p *Process) Start() {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return
	}

	cmd := exec.Command(p.name, p.args...)
	if p.dir != "" {
		cmd.Dir = p.dir
	}
	if p.env != nil {
		cmd.Env = append(os.Environ(), p.env...)
	}

	stdout, errOut := cmd.StdoutPipe()
	if errOut != nil {
		p.mu.Unlock()
		p.viewer.AddLine("ERROR: " + errOut.Error())
		p.redraw()
		return
	}
	stderr, errErr := cmd.StderrPipe()
	if errErr != nil {
		p.mu.Unlock()
		p.viewer.AddLine("ERROR: " + errErr.Error())
		p.redraw()
		return
	}

	if err := cmd.Start(); err != nil {
		p.exitErr = err
		p.mu.Unlock()
		p.viewer.AddLine("ERROR: " + err.Error())
		p.redraw()
		return
	}

	p.cmd = cmd
	p.running = true
	p.exitErr = nil
	p.mu.Unlock()

	// Stream stdout and stderr concurrently.
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); p.stream(stdout) }()
	go func() { defer wg.Done(); p.stream(stderr) }()

	// Wait goroutine: fires when both pipes are drained and the process exits.
	go func() {
		wg.Wait()
		err := cmd.Wait()

		p.mu.Lock()
		p.running = false
		p.exitErr = err
		restart := p.restartOnExit
		p.mu.Unlock()

		if err != nil && err.Error() != "signal: terminated" && err.Error() != "signal: interrupt" {
			p.viewer.AddLine("─── exited: " + err.Error() + " ───")
		} else {
			p.viewer.AddLine("─── process exited ───")
		}
		p.redraw()

		if restart {
			p.Start()
		}
	}()
}

// Stop sends an interrupt signal to the running process.
func (p *Process) Stop() {
	p.mu.Lock()
	cmd := p.cmd
	running := p.running
	p.mu.Unlock()
	if running && cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Signal(os.Interrupt)
	}
}

// Restart clears the log, stops any running instance, then starts fresh.
func (p *Process) Restart() {
	p.Stop()
	p.viewer.Clear()
	// Poll until the exit goroutine marks running=false (max ~1 s).
	go func() {
		for i := 0; i < 20; i++ {
			p.mu.Lock()
			r := p.running
			p.mu.Unlock()
			if !r {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}
		p.Start()
	}()
}

// IsRunning reports whether the process is currently running.
func (p *Process) IsRunning() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.running
}

// ExitError returns the error from the last process exit, or nil.
func (p *Process) ExitError() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.exitErr
}

// ── stream helper ─────────────────────────────────────────────────────────────

func (p *Process) stream(r io.Reader) {
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		p.viewer.AddLine(sc.Text())
		p.redraw()
	}
}

func (p *Process) redraw() {
	if p.redrawFn != nil {
		p.redrawFn()
	}
}

// ── oat.ThemeReceiver ─────────────────────────────────────────────────────────

// ApplyTheme applies theme tokens and propagates to the inner LogViewer.
func (p *Process) ApplyTheme(t latte.Theme) {
	p.Style = t.Text.Merge(p.callerStyle)
	p.FocusStyle = latte.Style{BorderFG: t.FocusBorder}
	p.statusStyle = t.Muted
	p.viewer.ApplyTheme(t)
}

// ── oat.Component ─────────────────────────────────────────────────────────────

// Measure reserves 1 row at the bottom for the status line.
func (p *Process) Measure(c oat.Constraint) oat.Size {
	inner := oat.Constraint{MaxWidth: c.MaxWidth, MaxHeight: c.MaxHeight - 1}
	if inner.MaxHeight < 0 {
		inner.MaxHeight = 0
	}
	sz := p.viewer.Measure(inner)
	return c.Clamp(oat.Size{Width: sz.Width, Height: sz.Height + 1})
}

// Render draws the LogViewer in the upper region and a status line at the bottom.
func (p *Process) Render(buf *oat.Buffer, region oat.Region) {
	style := p.EffectiveStyle(p.IsFocused())
	sub := buf.Sub(region)
	p.SetHitRegion(sub.Region())
	sub.FillBG(style)

	statusY := region.Height - 1

	// Status line.
	if statusY >= 0 {
		p.mu.Lock()
		running := p.running
		exitErr := p.exitErr
		p.mu.Unlock()

		var status string
		cmdStr := p.name
		if len(p.args) > 0 {
			cmdStr += " " + strings.Join(p.args, " ")
		}
		switch {
		case running:
			status = fmt.Sprintf("● %s  [running]", cmdStr)
		case exitErr != nil:
			status = fmt.Sprintf("✗ %s — %s", cmdStr, exitErr.Error())
		default:
			status = fmt.Sprintf("○ %s — stopped", cmdStr)
		}

		runes := []rune(status)
		if len(runes) > region.Width {
			runes = runes[:region.Width]
		}
		for len(runes) < region.Width {
			runes = append(runes, ' ')
		}
		sub.DrawText(0, statusY, string(runes), p.statusStyle)
	}

	// LogViewer above the status line.
	if statusY > 0 {
		p.viewer.SetFocused(p.IsFocused())
		p.viewer.Render(sub, oat.Region{X: 0, Y: 0, Width: region.Width, Height: statusY})
	}
}

// HandleKey delegates to the inner LogViewer.
func (p *Process) HandleKey(ev *oat.KeyEvent) bool {
	return p.viewer.HandleKey(ev)
}

// KeyBindings delegates to the inner LogViewer.
func (p *Process) KeyBindings() []oat.KeyBinding {
	return p.viewer.KeyBindings()
}

// Children exposes the inner LogViewer for focus walking and theme propagation.
func (p *Process) Children() []oat.Component { return []oat.Component{p.viewer} }

// AddChild is a no-op for Process.
func (p *Process) AddChild(_ oat.Component) {}
