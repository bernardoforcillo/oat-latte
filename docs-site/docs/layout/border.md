---
sidebar_position: 2
title: Border
description: Wrap any component in a configurable box border with an optional title.
---

# Border

`layout.Border` wraps a single child component with a configurable box border and an optional title stamped into the top rule.

## Basic usage

```go
panel := layout.NewBorder(innerComponent).
    WithTitle("My Panel")
```

## Builder options

| Method | Description |
|---|---|
| `WithTitle(title string, anchor ...oat.Anchor)` | Set a title and optionally its horizontal position |
| `WithTitleStyle(s latte.Style)` | Style applied to the title text only |
| `WithStyle(s latte.Style)` | Style applied to the border itself (colour, border type, padding) |
| `WithRoundedCorner(bool)` | Explicitly enable or disable arc corners (`╭╮╰╯`) |

## WithTitle

`WithTitle` accepts an optional `oat.Anchor` that controls where the title is placed in the top rule. The anchor is optional; omitting it defaults to `oat.AnchorLeft`.

```go
// Left-aligned (default)
layout.NewBorder(inner).WithTitle("My Panel")

// Centred
layout.NewBorder(inner).WithTitle("My Panel", oat.AnchorCenter)

// Right-aligned
layout.NewBorder(inner).WithTitle("My Panel", oat.AnchorRight)
```

| Anchor | Result |
|---|---|
| `oat.AnchorLeft` (default) | `╭─ My Panel ──────────╮` |
| `oat.AnchorCenter` | `╭──── My Panel ────────╮` |
| `oat.AnchorRight` | `╭────────── My Panel ──╮` |

## WithRoundedCorner

```go
func (b *Border) WithRoundedCorner(rounded bool) *Border
```

Stores the rounded-corner intent in an internal field; does **not** mutate `Style.Border`. The effective corner shape is resolved at render time: `BorderSingle` ↔ `BorderRounded` is toggled based on this field.

```go
// Arc corners: ╭─ My Panel ──╮ / ╰──────────╯
panel := layout.NewBorder(inner).
    WithTitle("My Panel").
    WithRoundedCorner(true)
```

| Border style | `WithRoundedCorner(true)` |
|---|---|
| `BorderSingle` (default) | Renders with arc corners `╭╮╰╯` |
| `BorderRounded` | Already rounded — no change |
| `BorderDouble` | Silently ignored |
| `BorderThick` | Silently ignored |
| `BorderDashed` | Silently ignored |

Once called, this explicit choice overrides the theme's `RoundedCorner` setting for this `Border`. Calling `WithRoundedCorner(false)` explicitly opts out of rounded corners even when `theme.RoundedCorner` is `true`.

:::tip
All built-in themes have `RoundedCorner: true`, so `Border` automatically uses arc corners when a theme is applied. Only call `WithRoundedCorner` explicitly when you want to override the theme's default for a specific panel.
:::

## Focus highlight

When any descendant of the border is focused, `Border` automatically promotes its border colour to the theme's `FocusBorder` token. No extra code is needed.

## Custom style

```go
// Explicit padding inside the border
panel := layout.NewBorder(inner).
    WithStyle(latte.Style{Padding: latte.Insets{Bottom: 1}}).
    WithTitle("Editor")

// Custom border colour
panel := layout.NewBorder(inner).
    WithStyle(latte.Style{BorderFG: latte.Hex("#ff6600")})
```

## Nesting with ScrollView

When combining `Border` with `ScrollView`, prefer **`Border(ScrollView(VBox))`** over `ScrollView(Border(VBox))`. In the correct pattern the border chrome is fixed and only the content scrolls. In the reverse pattern the entire `Border` (including its title row) scrolls off-screen.

```go
// CORRECT — fixed border, scrolling content:
panel := layout.NewBorder(
    layout.NewVBox(items...).AsScrollView().WithScrollBar(true),
).WithTitle("Items")
```

See [ScrollView](./scrollview.md) for details.
