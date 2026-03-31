---
sidebar_position: 4
title: ScrollView
description: Clip a child to a viewport and let the user scroll vertically to reveal overflow content.
---

# ScrollView

`layout.ScrollView` clips a single child to a viewport and lets the user scroll vertically to reveal content that exceeds the available height.

## Constructors

```go
// Standalone constructor
sv := layout.NewScrollView(myVBox).WithScrollBar(true)

// Convenience builders — wrap a box directly
sv := layout.NewVBox(items...).AsScrollView().WithScrollBar(true)
sv := layout.NewHBox(cols...).AsScrollView()   // horizontal content, vertical scroll
```

## WithScrollBar

```go
func (sv *ScrollView) WithScrollBar(show bool, anchor ...oat.Anchor) *ScrollView
```

| Parameter | Default | Effect |
|---|---|---|
| `show` | `false` | `true` — display a single-column gutter bar |
| `anchor` (optional) | `oat.AnchorRight` | `oat.AnchorLeft` — move bar to the left edge |

The bar uses the theme's `Muted` token for the track (`│`) and `Accent` FG for the thumb (`█`). These colours are set by `ApplyTheme` and can be overridden with `WithTrackColor` / `WithThumbColor`.

## WithTrackColor / WithThumbColor

```go
func (sv *ScrollView) WithTrackColor(c latte.Color) *ScrollView
func (sv *ScrollView) WithThumbColor(c latte.Color) *ScrollView
```

Override the scroll bar colours for this specific `ScrollView`. Overrides survive `SetTheme` calls. Pass `latte.ColorDefault` to revert to theme-driven behaviour.

```go
sv := layout.NewVBox(items...).AsScrollView().
    WithScrollBar(true).
    WithTrackColor(latte.Hex("#444466")).
    WithThumbColor(latte.ColorBrightCyan)
```

## Nesting with Border

Prefer **`Border(ScrollView(VBox))`** over `ScrollView(Border(VBox))`. In the correct pattern the border chrome is fixed and only the VBox content scrolls. In the reverse pattern the entire `Border` — including its title row — scrolls off-screen as the user scrolls down.

```go
// CORRECT — fixed border, scrolling content:
panel := layout.NewBorder(
    layout.NewVBox(items...).AsScrollView().WithScrollBar(true),
).WithTitle("Items")

// VALID but unusual — entire border (including title) scrolls:
panel := layout.NewScrollView(
    layout.NewBorder(layout.NewVBox(items...)).WithTitle("Items"),
).WithScrollBar(true)
```

## VFill / FlexChild inside ScrollView

`VFill` and `FlexChild` behave differently depending on whether content overflows the viewport:

- **Content fits** (no scrolling) — the full viewport height is passed to the child, so flex children expand normally.
- **Content overflows** (scrolling active) — the child is measured unconstrained and flex children collapse to zero height.

Avoid `VFill` / `FlexChild` inside a `ScrollView` that is expected to scroll; use fixed-height children instead.

## Focus and keyboard

`ScrollView` is always present in the Tab cycle. `HandleKey` returns `false` for all scroll keys when content fits the viewport, so arrow keys fall through to inter-widget focus cycling as usual. When content overflows, `HandleKey` consumes the scroll keys:

| Key | Action |
|---|---|
| `↑` / `↓` | ±1 row |
| `PgUp` / `PgDn` | ±viewport height |
| `Home` / `End` | Jump to top / bottom |

:::warning Do not use FocusGuard on ScrollView
The focus tree is collected once at startup before any `Measure`/`Render` pass. A `FocusGuard` that checks `contentH > viewportH` would always return `false` at startup (both values are zero), permanently excluding the `ScrollView` from Tab cycling.
:::

## Programmatic scroll

```go
sv.ScrollOffset() int        // current row offset
sv.ContentHeight() int       // full unconstrained child height
sv.ScrollTo(off int)         // set offset; clamped to [0, contentH-viewportH]
```
