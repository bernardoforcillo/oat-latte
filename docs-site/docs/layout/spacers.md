---
sidebar_position: 6
title: Spacers
description: VFill, HFill, VGap, HGap — flex and fixed spacers for pushing, separating, and distributing space in VBox and HBox.
---

# Spacers

oat-latte has two families of spacers:

| Type | Behaviour | Use when |
|---|---|---|
| **`VFill` / `HFill`** | Flex spacer — claims remaining space via flex distribution | You want to push content to one side, or fill variable gaps |
| **`VGap` / `HGap`** | Fixed spacer — always claims exactly `n` cells | You need a guaranteed fixed gap between two widgets |

## VFill and HFill

`VFill` is a flex spacer for `VBox`; `HFill` is a flex spacer for `HBox`. When added with `AddChild`, a `VFill` / `HFill` auto-promotes itself to a flex slot with weight `1` — it is shorthand for `AddFlexChild(layout.NewVFill(), 1)`.

```go
// Push content to either end of an HBox.
row := layout.NewHBox()
row.AddChild(leftLabel)
row.AddChild(layout.NewHFill())   // fills the gap between labels
row.AddChild(rightLabel)

// Fill remaining vertical space in a VBox.
vbox := layout.NewVBox()
vbox.AddChild(topContent)
vbox.AddChild(layout.NewVFill())  // absorbs all remaining rows
vbox.AddChild(bottomContent)
```

### WithMaxSize — capped flex spacer

`WithMaxSize(n)` caps the spacer's growth at `n` cells. The spacer still participates in flex distribution but yields any allocation beyond `n`.

```go
// Gap up to 2 columns — shrinks if the container is tight.
row.AddChild(layout.NewHFill().WithMaxSize(2))
```

Use `WithMaxSize` when you want a gap that can gracefully shrink (e.g. inside a resizable dialog). For a gap that must always be a fixed size, use `HGap` / `VGap` instead.

## VGap and HGap

`VGap(n)` is a fixed-height inert spacer for use in `VBox`. `HGap(n)` is a fixed-width inert spacer for use in `HBox`. They do **not** participate in flex distribution — `Measure` returns a fixed `Size` and `Render` is a no-op.

```go
// Exactly 1 row of vertical space between two sections.
vbox := layout.NewVBox(
    widget.NewText("Section A"),
    layout.NewVGap(1),
    widget.NewText("Section B"),
)

// Exactly 2 columns between two buttons.
btnRow := layout.NewHBox()
btnRow.AddChild(layout.NewHFill())   // flex push to the right
btnRow.AddChild(cancelBtn)
btnRow.AddChild(layout.NewHGap(2))   // fixed 2-column gap
btnRow.AddChild(okBtn)
```

| Type | Constructor | Axis | Measure result |
|---|---|---|---|
| `VGap` | `layout.NewVGap(n int) *VGap` | Vertical (VBox) | `Size{Width: 0, Height: n}` |
| `HGap` | `layout.NewHGap(n int) *HGap` | Horizontal (HBox) | `Size{Width: n, Height: 0}` |

`n < 0` is clamped to `0`. `Render` is a no-op — both types are pure spacers with no visual output.

## VFill/HFill vs VGap/HGap — which to use

| Scenario | Use |
|---|---|
| Push buttons to the right edge of an HBox | `HFill` |
| Fixed 2-column gap between two buttons | `HGap(2)` |
| Fill all remaining rows in a VBox | `VFill` |
| Fixed 1-row gap between two sections | `VGap(1)` |
| Gap that should shrink in a tight dialog | `HFill.WithMaxSize(2)` or `VFill.WithMaxSize(1)` |
