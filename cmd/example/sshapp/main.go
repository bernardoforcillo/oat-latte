// Command sshapp demonstrates serving an oat-latte TUI over SSH.
//
// Run it, then connect from any machine with:
//
//	ssh -p 2222 localhost
//
// No client installation is required — the full TUI runs server-side and is
// streamed over the SSH session.
package main

import (
	"fmt"
	"log"

	oat "github.com/antoniocali/oat-latte"
	"github.com/antoniocali/oat-latte/latte"
	"github.com/antoniocali/oat-latte/layout"
	"github.com/antoniocali/oat-latte/widget"
)

func buildUI() oat.Component {
	theme := latte.ThemeDefault

	title := widget.NewText("  oat-latte over SSH  ")
	title.WithStyle(theme.Accent.WithBold())

	info := widget.NewText("Welcome! You are running this TUI entirely over SSH.\nPress Ctrl+C or Esc to disconnect.")
	info.WithStyle(theme.Text)

	counter := widget.NewText("Count: 0")
	counter.WithStyle(theme.Text)

	count := 0
	inc := widget.NewButton("Increment", func() {
		count++
		counter.SetText(fmt.Sprintf("Count: %d", count))
	})
	dec := widget.NewButton("Decrement", func() {
		count--
		counter.SetText(fmt.Sprintf("Count: %d", count))
	})

	btnRow := layout.NewHBox().WithGap(2)
	btnRow.AddChild(dec)
	btnRow.AddChild(inc)

	input := widget.NewEditText().
		WithPlaceholder("Type something…")

	box := layout.NewVBox().WithGap(1)
	box.AddChild(title)
	box.AddChild(info)
	box.AddChild(widget.NewDivider(widget.AxisHorizontal))
	box.AddChild(counter)
	box.AddChild(btnRow)
	box.AddChild(widget.NewDivider(widget.AxisHorizontal))
	box.AddChild(widget.NewText("Edit text:"))
	box.AddChild(input)

	return layout.NewPaddingUniform(box, 2)
}

func main() {
	addr := ":2222"
	log.Printf("Listening for SSH connections on %s — connect with: ssh -p 2222 localhost", addr)

	err := oat.ServeSSH(func() *oat.Canvas {
		ui := buildUI()
		theme := latte.ThemeDefault
		bar := widget.NewStatusBar()

		cv := oat.NewCanvas(
			oat.WithBody(ui),
			oat.WithFooter(bar),
			oat.WithTheme(theme),
		)
		cv.SetAutoBar(bar)
		return cv
	}, oat.SSHOpts{
		Addr: addr,
		// No AuthHandler = accept all connections (good for demos).
	})
	if err != nil {
		log.Fatal(err)
	}
}
