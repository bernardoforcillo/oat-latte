// Package main demonstrates the Flutter-inspired reactive widget system.
//
// Patterns shown:
//   - StatefulWidget + State[T]  — stateful counter driven by button presses
//   - ValueNotifier[T] + Consumer — shared live clock updated from a goroutine
//   - ChangeNotifier (embedded)   — custom domain model with listener support
//   - canvas.Redraw               — wiring background mutations to the event loop
package main

import (
	"fmt"
	"log"
	"time"

	oat "github.com/antoniocali/oat-latte"
	"github.com/antoniocali/oat-latte/latte"
	"github.com/antoniocali/oat-latte/layout"
	"github.com/antoniocali/oat-latte/widget"
)

// ── Domain model using ChangeNotifier ────────────────────────────────────────

// TodoList is a simple domain model with embedded ChangeNotifier.
// Any mutation calls NotifyListeners() so wired-up components rebuild.
type TodoList struct {
	oat.ChangeNotifier
	items []string
}

func (t *TodoList) Add(item string) {
	t.items = append(t.items, item)
	t.NotifyListeners()
}

func (t *TodoList) Remove(i int) {
	if i < 0 || i >= len(t.items) {
		return
	}
	t.items = append(t.items[:i], t.items[i+1:]...)
	t.NotifyListeners()
}

func (t *TodoList) Len() int      { return len(t.items) }
func (t *TodoList) Get(i int) string { return t.items[i] }

// ── Counter section (StatefulWidget + State[T]) ───────────────────────────────

type counterState struct{ n int }

func buildCounterSection(canvas *oat.Canvas) oat.Component {
	state := oat.NewState(counterState{}, canvas.Redraw)

	return oat.NewStatefulWidget(state, func(s counterState) oat.Component {
		decBtn := widget.NewButton(" − ", func() {
			state.SetState(func(cs *counterState) { cs.n-- })
		})
		incBtn := widget.NewButton(" + ", func() {
			state.SetState(func(cs *counterState) { cs.n++ })
		})
		resetBtn := widget.NewButton("Reset", func() {
			state.SetState(func(cs *counterState) { cs.n = 0 })
		})

		label := widget.NewText(fmt.Sprintf("Count: %d", s.n)).
			WithStyle(latte.Style{Bold: true})

		row := layout.NewHBox(decBtn, layout.NewHGap(1), label, layout.NewHGap(1), incBtn)
		return layout.NewVBox(
			widget.NewTitle("Counter  (StatefulWidget)").WithSeparator(true),
			layout.NewVGap(1),
			row,
			layout.NewVGap(1),
			resetBtn,
		)
	})
}

// ── Live clock section (ValueNotifier + Consumer) ────────────────────────────

func buildClockSection(canvas *oat.Canvas) oat.Component {
	clock := oat.NewValueNotifier(time.Now())

	// Tick the clock from a background goroutine every second.
	go func() {
		for range time.Tick(time.Second) {
			clock.Set(time.Now())
			canvas.Redraw()
		}
	}()

	return oat.NewConsumer(clock, func(t time.Time) oat.Component {
		return layout.NewVBox(
			widget.NewTitle("Live Clock  (ValueNotifier + Consumer)").WithSeparator(true),
			layout.NewVGap(1),
			widget.NewText(t.Format("15:04:05")),
		)
	}, canvas.Redraw)
}

// ── Todo section (ChangeNotifier) ────────────────────────────────────────────

func buildTodoSection(canvas *oat.Canvas) oat.Component {
	todos := &TodoList{}
	todos.Add("Buy oat milk")
	todos.Add("Write tests")
	todos.AddListener(canvas.Redraw)

	input := widget.NewEditText().WithHint("New item…")

	addBtn := widget.NewButton("Add", func() {
		text := input.GetText()
		if text == "" {
			return
		}
		todos.Add(text)
		input.SetText("")
		canvas.InvalidateLayout()
	})

	inputRow := layout.NewHBox(input, layout.NewHGap(1), addBtn)

	// The item list is a StatelessWidget that reads from todos.
	// It is invalidated via the ChangeNotifier listener (canvas.Redraw).
	itemList := oat.NewStatelessWidget(func() oat.Component {
		rows := layout.NewVBox()
		for i := 0; i < todos.Len(); i++ {
			idx := i // capture for closure
			removeBtn := widget.NewButton("✕", func() {
				todos.Remove(idx)
				canvas.InvalidateLayout()
			})
			row := layout.NewHBox(
				widget.NewText(fmt.Sprintf("%d. %s", idx+1, todos.Get(idx))),
				layout.NewHFill(),
				removeBtn,
			)
			rows.AddChild(row)
		}
		return rows
	})
	// Invalidate the list whenever todos changes so it rebuilds on next render.
	todos.AddListener(itemList.Invalidate)

	return layout.NewVBox(
		widget.NewTitle("Todo List  (ChangeNotifier)").WithSeparator(true),
		layout.NewVGap(1),
		inputRow,
		layout.NewVGap(1),
		itemList,
	)
}

// ── Main ─────────────────────────────────────────────────────────────────────

func main() {
	canvas := oat.NewCanvas()

	body := layout.NewVBox(
		layout.NewPaddingUniform(buildCounterSection(canvas), 1),
		layout.NewPaddingUniform(buildClockSection(canvas), 1),
		layout.NewPaddingUniform(buildTodoSection(canvas), 1),
	)

	canvas = oat.NewCanvas(
		oat.WithTheme(latte.ThemeDark),
		oat.WithBody(body),
		oat.WithAutoStatusBar(widget.NewStatusBar()),
	)

	if err := canvas.Run(); err != nil {
		log.Fatal(err)
	}
}
