package oat

// App wraps a Canvas with a Router to provide multi-screen navigation.
//
// Usage:
//
//	router := oat.NewRouter().
//	    Handle("/", homeBuilder).
//	    Handle("/users/:id", userBuilder)
//
//	canvas := oat.NewCanvas(oat.WithAutoStatusBar())
//	app := oat.NewApp(canvas, router)
//	app.Navigate("/")    // load initial route
//	canvas.Run()
type App struct {
	canvas *Canvas
	router *Router
	stack  []navEntry
}

type navEntry struct {
	path      string
	component Component
}

// NewApp creates an App wrapping canvas with the given router.
// Call Navigate to push the initial route before canvas.Run().
func NewApp(canvas *Canvas, router *Router) *App {
	return &App{canvas: canvas, router: router}
}

// Navigate resolves path, builds the screen, pushes it onto the stack,
// sets it as the canvas body, and triggers a redraw.
// No-op if path does not match any registered route.
func (a *App) Navigate(path string) {
	builder, params, ok := a.router.Match(path)
	if !ok {
		return
	}
	c := builder(params, a)
	a.stack = append(a.stack, navEntry{path: path, component: c})
	a.canvas.SetBody(c)
	a.canvas.InvalidateLayout()
	a.canvas.Redraw()
}

// GoBack pops the current screen and restores the previous one.
// No-op if the stack has fewer than 2 entries.
func (a *App) GoBack() {
	if len(a.stack) < 2 {
		return
	}
	a.stack = a.stack[:len(a.stack)-1]
	prev := a.stack[len(a.stack)-1]
	a.canvas.SetBody(prev.component)
	a.canvas.InvalidateLayout()
	a.canvas.Redraw()
}

// CurrentPath returns the path of the currently displayed screen, or "".
func (a *App) CurrentPath() string {
	if len(a.stack) == 0 {
		return ""
	}
	return a.stack[len(a.stack)-1].path
}

// CanGoBack reports whether GoBack would do anything.
func (a *App) CanGoBack() bool { return len(a.stack) >= 2 }
