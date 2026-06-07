package oat

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"os"
	"sync"

	"github.com/antoniocali/oat-latte/latte"
	"github.com/gdamore/tcell/v2"
	"golang.org/x/crypto/ssh"
)

// SSHOpts configures the SSH server started by ServeSSH.
type SSHOpts struct {
	// Addr is the listen address, e.g. ":2222". Defaults to ":2222".
	Addr string

	// HostKeyFile is a path to a PEM-encoded private key file.
	// If empty and HostKeyPEM is also empty, a fresh ECDSA P-256 key is
	// generated in memory for the lifetime of this process.
	HostKeyFile string

	// HostKeyPEM is an in-memory PEM-encoded private key. Takes precedence
	// over HostKeyFile when both are set.
	HostKeyPEM []byte

	// AuthHandler, when non-nil, is called for each public-key authentication
	// attempt. Return true to allow the connection.
	// When nil, ALL connections are accepted (useful for development/internal tools).
	AuthHandler func(meta ssh.ConnMetadata, key ssh.PublicKey) (bool, error)

	// PasswordHandler, when non-nil, is called for password authentication.
	// When nil, password auth is disabled.
	PasswordHandler func(meta ssh.ConnMetadata, password []byte) (bool, error)

	// MaxConnections is the maximum number of simultaneous SSH connections.
	// 0 means no limit.
	MaxConnections int
}

// ServeSSH starts an SSH server that serves a TUI application to every
// connecting client. appBuilder is called once per connection and must
// return a fully configured Canvas (without calling Run on it — ServeSSH
// calls RunWithTty instead).
//
// Each connection runs in its own goroutine with its own Canvas instance,
// so the builder must return independent objects on every call.
//
// Example:
//
//	oat.ServeSSH(func() *oat.Canvas {
//	    return oat.NewCanvas(
//	        oat.WithBody(buildUI()),
//	        oat.WithTheme(latte.DefaultTheme()),
//	    )
//	}, oat.SSHOpts{Addr: ":2222"})
func ServeSSH(appBuilder func() *Canvas, opts SSHOpts) error {
	if opts.Addr == "" {
		opts.Addr = ":2222"
	}

	signer, err := loadOrGenerateHostKey(opts)
	if err != nil {
		return fmt.Errorf("oat/ssh: host key: %w", err)
	}

	cfg := &ssh.ServerConfig{
		NoClientAuth: opts.AuthHandler == nil && opts.PasswordHandler == nil,
	}
	cfg.AddHostKey(signer)

	if opts.AuthHandler != nil {
		cfg.PublicKeyCallback = func(meta ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			ok, err := opts.AuthHandler(meta, key)
			if err != nil {
				return nil, err
			}
			if !ok {
				return nil, fmt.Errorf("public key rejected")
			}
			return &ssh.Permissions{}, nil
		}
	}
	if opts.PasswordHandler != nil {
		cfg.PasswordCallback = func(meta ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			ok, err := opts.PasswordHandler(meta, password)
			if err != nil {
				return nil, err
			}
			if !ok {
				return nil, fmt.Errorf("password rejected")
			}
			return &ssh.Permissions{}, nil
		}
	}

	ln, err := net.Listen("tcp", opts.Addr)
	if err != nil {
		return fmt.Errorf("oat/ssh: listen %s: %w", opts.Addr, err)
	}

	var (
		mu      sync.Mutex
		current int
	)

	for {
		conn, err := ln.Accept()
		if err != nil {
			return fmt.Errorf("oat/ssh: accept: %w", err)
		}

		if opts.MaxConnections > 0 {
			mu.Lock()
			over := current >= opts.MaxConnections
			mu.Unlock()
			if over {
				_ = conn.Close()
				continue
			}
		}

		mu.Lock()
		current++
		mu.Unlock()

		go func(netConn net.Conn) {
			defer func() {
				mu.Lock()
				current--
				mu.Unlock()
			}()
			serveSSHConn(netConn, cfg, appBuilder)
		}(conn)
	}
}

// serveSSHConn handles one SSH network connection.
func serveSSHConn(netConn net.Conn, cfg *ssh.ServerConfig, appBuilder func() *Canvas) {
	defer netConn.Close()

	sshConn, chans, reqs, err := ssh.NewServerConn(netConn, cfg)
	if err != nil {
		return
	}
	defer sshConn.Close()

	// Discard global requests (keepalive, etc.).
	go ssh.DiscardRequests(reqs)

	for newCh := range chans {
		if newCh.ChannelType() != "session" {
			_ = newCh.Reject(ssh.UnknownChannelType, "unsupported channel type")
			continue
		}
		ch, requests, err := newCh.Accept()
		if err != nil {
			return
		}
		go serveSSHSession(ch, requests, appBuilder)
	}
}

// serveSSHSession handles one SSH session channel.
func serveSSHSession(ch ssh.Channel, requests <-chan *ssh.Request, appBuilder func() *Canvas) {
	defer ch.Close()

	tty := newSSHTty(ch)

	for req := range requests {
		switch req.Type {
		case "pty-req":
			w, h := parsePtyReq(req.Payload)
			tty.setSize(w, h)
			if req.WantReply {
				_ = req.Reply(true, nil)
			}
		case "window-change":
			w, h := parseWindowChange(req.Payload)
			tty.resize(w, h)
			if req.WantReply {
				_ = req.Reply(true, nil)
			}
		case "shell", "exec":
			if req.WantReply {
				_ = req.Reply(true, nil)
			}
			// Run the TUI app on this TTY.
			canvas := appBuilder()
			_ = canvas.RunWithTty(tty)
			return
		default:
			if req.WantReply {
				_ = req.Reply(false, nil)
			}
		}
	}
}

// ── sshTty ────────────────────────────────────────────────────────────────────

// sshTty wraps an ssh.Channel and satisfies tcell.Tty.
type sshTty struct {
	ch       ssh.Channel
	mu       sync.Mutex
	width    int
	height   int
	resizeCb func()
	closed   bool
	done     chan struct{}
}

func newSSHTty(ch ssh.Channel) *sshTty {
	return &sshTty{
		ch:     ch,
		width:  80,
		height: 24,
		done:   make(chan struct{}),
	}
}

func (t *sshTty) Start() error { return nil }

func (t *sshTty) Stop() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.closed {
		t.closed = true
		close(t.done)
	}
	return nil
}

func (t *sshTty) Drain() error { return nil }

func (t *sshTty) Close() error {
	_ = t.Stop()
	return t.ch.Close()
}

func (t *sshTty) Read(p []byte) (int, error) {
	// Check if we've been stopped.
	select {
	case <-t.done:
		return 0, io.EOF
	default:
	}
	return t.ch.Read(p)
}

func (t *sshTty) Write(p []byte) (int, error) {
	return t.ch.Write(p)
}

func (t *sshTty) WindowSize() (tcell.WindowSize, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return tcell.WindowSize{Width: t.width, Height: t.height}, nil
}

func (t *sshTty) NotifyResize(cb func()) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.resizeCb = cb
}

func (t *sshTty) setSize(w, h int) {
	t.mu.Lock()
	if w > 0 {
		t.width = w
	}
	if h > 0 {
		t.height = h
	}
	cb := t.resizeCb
	t.mu.Unlock()
	if cb != nil {
		cb()
	}
}

func (t *sshTty) resize(w, h int) { t.setSize(w, h) }

// ── Canvas.RunWithTty ─────────────────────────────────────────────────────────

// RunWithTty is like Run but uses the supplied Tty instead of the local
// terminal. It is the low-level entry point used by ServeSSH; applications
// normally call Run instead.
func (cv *Canvas) RunWithTty(tty tcell.Tty) error {
	screen, err := tcell.NewTerminfoScreenFromTty(tty)
	if err != nil {
		return err
	}
	if err := screen.Init(); err != nil {
		return err
	}
	defer screen.Fini()

	if cv.crashHandler != nil {
		defer func() {
			if r := recover(); r != nil {
				cv.crashHandler(r)
			}
		}()
	}

	screen.EnableMouse()
	screen.Clear()
	cv.screen = screen
	cv.quit = make(chan struct{})

	if cv.body != nil {
		cv.focus.Collect(cv.body)
	}
	if cv.theme != nil {
		cv.applyTheme()
	}
	if cv.focusStyle != (latte.Style{}) {
		cv.injectFocusStyle()
	}
	if cv.primary != nil {
		cv.focus.FocusByRef(cv.primary)
	}
	cv.updateStatusBar()

	eventCh := make(chan tcell.Event, 64)
	go func() {
		for {
			ev := screen.PollEvent()
			if ev == nil {
				return
			}
			select {
			case eventCh <- ev:
			case <-cv.quit:
				return
			}
		}
	}()

	cv.render()
	screen.Show()

	return cv.eventLoop(screen, eventCh, nil)
}



// ── helpers ───────────────────────────────────────────────────────────────────

// loadOrGenerateHostKey returns a signer from opts, or generates a fresh one.
func loadOrGenerateHostKey(opts SSHOpts) (ssh.Signer, error) {
	var pemBytes []byte

	switch {
	case len(opts.HostKeyPEM) > 0:
		pemBytes = opts.HostKeyPEM
	case opts.HostKeyFile != "":
		b, err := os.ReadFile(opts.HostKeyFile)
		if err != nil {
			return nil, err
		}
		pemBytes = b
	default:
		// Generate an ephemeral ECDSA key.
		priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return nil, err
		}
		der, err := x509.MarshalECPrivateKey(priv)
		if err != nil {
			return nil, err
		}
		pemBytes = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der})
	}

	key, err := ssh.ParsePrivateKey(pemBytes)
	if err != nil {
		return nil, err
	}
	return key, nil
}

// parsePtyReq decodes a pty-req payload and returns (width, height).
// SSH pty-req wire format: string term, uint32 width, uint32 height, ...
func parsePtyReq(payload []byte) (w, h int) {
	// Skip the TERM string: 4-byte length + data.
	if len(payload) < 4 {
		return 80, 24
	}
	termLen := int(payload[0])<<24 | int(payload[1])<<16 | int(payload[2])<<8 | int(payload[3])
	off := 4 + termLen
	if len(payload) < off+8 {
		return 80, 24
	}
	w = int(payload[off])<<24 | int(payload[off+1])<<16 | int(payload[off+2])<<8 | int(payload[off+3])
	h = int(payload[off+4])<<24 | int(payload[off+5])<<16 | int(payload[off+6])<<8 | int(payload[off+7])
	if w <= 0 {
		w = 80
	}
	if h <= 0 {
		h = 24
	}
	return w, h
}

// parseWindowChange decodes a window-change payload and returns (width, height).
// Format: uint32 width, uint32 height, uint32 pixW, uint32 pixH.
func parseWindowChange(payload []byte) (w, h int) {
	if len(payload) < 8 {
		return 80, 24
	}
	w = int(payload[0])<<24 | int(payload[1])<<16 | int(payload[2])<<8 | int(payload[3])
	h = int(payload[4])<<24 | int(payload[5])<<16 | int(payload[6])<<8 | int(payload[7])
	if w <= 0 {
		w = 80
	}
	if h <= 0 {
		h = 24
	}
	return w, h
}
