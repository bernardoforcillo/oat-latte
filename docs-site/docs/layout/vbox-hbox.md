---
sidebar_position: 1
title: VBox and HBox
description: Vertical and horizontal box containers — the primary layout building blocks in oat-latte.
---

# VBox and HBox

`VBox` stacks children **vertically**; `HBox` places them **horizontally**. They are the primary layout building blocks — almost every screen is composed from nested VBox and HBox containers.

## Constructors

```go
// Variadic constructor — all children are fixed-size (AddChild semantics).
vbox := layout.NewVBox(titleText, bodyText, btnRow)
hbox := layout.NewHBox(leftPanel, rightPanel)

// Empty box — children added later.
vbox := layout.NewVBox()
hbox := layout.NewHBox()
```

## AddChild vs AddFlexChild

This is the core sizing decision for every child.

**`AddChild(c)`** — the child takes its **natural size** and no more. The box measures the child and allocates exactly that. Use this for elements whose size is inherently fixed: labels, buttons, a single-line input, a status row.

**`AddFlexChild(c, weight)`** — the child participates in **flex distribution**. After all fixed children have claimed their space, whatever is left is divided among flex children proportionally by weight. A child with weight `2` gets twice the space of one with weight `1`. Use this for the main content area, multi-line editors, lists — anything that should grow to fill available space.

```
┌─────────────────────────────┐
│  AddChild  → natural size   │  ← e.g. a title row: always 1 row tall
│  AddChild  → natural size   │  ← e.g. a button row: always 1 row tall
│                             │
│  AddFlexChild weight 1      │  ← fills all remaining space
│                             │
│  AddChild  → natural size   │  ← e.g. a status bar: always 1 row tall
└─────────────────────────────┘
```

When **multiple** flex children are present, space is split by the ratio of their weights:

```
remaining space = 30 rows
  flex child A  weight 1  → 10 rows  (1/3)
  flex child B  weight 2  → 20 rows  (2/3)
```

```go
vbox := layout.NewVBox()
vbox.AddChild(widget.NewText("Label"))        // fixed: takes its natural height
vbox.AddFlexChild(myEditText, 1)              // flex weight 1
vbox.AddFlexChild(anotherField, 2)            // flex weight 2 → twice as tall

// VFill shorthand — equivalent to AddFlexChild(layout.NewVFill(), 1)
vbox.AddChild(layout.NewVFill())

// HBox variadic constructor, then a flex child
hbox := layout.NewHBox(leftLabel, rightLabel)
hbox.AddFlexChild(progressBar, 1)
```

:::tip Most common pattern
One `AddFlexChild(mainContent, 1)` for the central area, `AddChild` for everything else (header, footer, button rows). Everything snaps to its natural size; the main area fills the rest.
:::

## FlexChild wrapper

`layout.NewFlexChild` lets you pass flex children directly to the variadic `NewVBox` / `NewHBox` constructors without a separate `AddFlexChild` call:

```go
// Two-step pattern:
vbox := layout.NewVBox(titleText)
vbox.AddFlexChild(bodyEditor, 1)
vbox.AddChild(btnRow)

// One-liner with NewFlexChild:
vbox := layout.NewVBox(
    titleText,
    layout.NewFlexChild(bodyEditor),   // weight defaults to 1
    btnRow,
)
```

See [FlexChild](./flexchild.md) for the full reference.

## AsScrollView

Wrap a box in a `ScrollView` with a single call:

```go
sv := layout.NewVBox(items...).AsScrollView().WithScrollBar(true)
```

See [ScrollView](./scrollview.md) for details.

## Cross-axis alignment

By default every child in a VBox fills the full allocated **width** and every child in an HBox fills the full allocated **height**. Cross-axis alignment lets you shrink a child to its natural size and pin it within its slot — without spacer widgets.

### Box-wide default — WithHAlign / WithVAlign

`VBox.WithHAlign` sets a default horizontal alignment for all children that do not declare their own. `HBox.WithVAlign` sets the default vertical alignment.

```go
// Every child right-aligned inside this VBox
vbox := layout.NewVBox(titleText, bodyText, footerText).
    WithHAlign(oat.HAlignRight)

// Every child pinned to the bottom inside this HBox
hbox := layout.NewHBox(iconText, nameText, statusText).
    WithVAlign(oat.VAlignBottom)
```

The zero value (`HAlignFill` / `VAlignFill`) keeps the existing full-stretch behaviour.

**HAlign values** (used by VBox):

| Value | Effect |
|---|---|
| `oat.HAlignFill` | Fill the full allocated width (default) |
| `oat.HAlignLeft` | Shrink to natural width, pin left |
| `oat.HAlignCenter` | Shrink to natural width, centre horizontally |
| `oat.HAlignRight` | Shrink to natural width, pin right |

**VAlign values** (used by HBox):

| Value | Effect |
|---|---|
| `oat.VAlignFill` | Fill the full allocated height (default) |
| `oat.VAlignTop` | Shrink to natural height, pin top |
| `oat.VAlignMiddle` | Shrink to natural height, centre vertically |
| `oat.VAlignBottom` | Shrink to natural height, pin bottom |

### Per-widget self-alignment

Every built-in widget exposes fluent `WithHAlign` / `WithVAlign` methods:

```go
saveBtn   := widget.NewButton("Save",   fn).WithHAlign(oat.HAlignRight)
cancelBtn := widget.NewButton("Cancel", fn).WithHAlign(oat.HAlignLeft)

vbox := layout.NewVBox(saveBtn, cancelBtn)
```

For custom widgets that embed `BaseComponent` but do not yet have their own builders, set the field directly:

```go
myWidget.BaseComponent.HAlign = oat.HAlignRight
```

### Per-child override — AlignChild

`layout.NewAlignChild` is the cleanest inline override, with the highest priority:

```go
vbox := layout.NewVBox(
    layout.NewAlignChild(saveBtn,   oat.HAlignRight, oat.VAlignFill),
    layout.NewAlignChild(cancelBtn, oat.HAlignLeft,  oat.VAlignFill),
)
```

See [AlignChild](./alignchild.md) for full details.

### Resolution order

For each child the effective alignment is resolved as:

1. `AlignChild` wrapper — if the child is an `AlignChild`, its value wins.
2. Child's own `AlignProvider` — `BaseComponent.HAlign` / `VAlign` if non-fill.
3. Box-wide default — the value passed to `WithHAlign` / `WithVAlign`.
4. `HAlignFill` / `VAlignFill` — full-stretch fallback.

### Alignment scope — direct children only

Alignment is only honoured on the **direct children** of the distributing box. Setting `VAlign` on a widget nested inside a `Border` that is itself a child of an `HBox` has no effect — the `HBox` reads the `Border`'s alignment, not the inner widget's.

```go
// WRONG — VAlign on the Text, buried inside Border.
inner  := widget.NewText("mid").WithVAlign(oat.VAlignMiddle) // ignored
border := layout.NewBorder(inner)
hbox.AddFlexChild(border, 1)

// CORRECT — set VAlign on the Border (direct child of HBox).
inner  := widget.NewText("mid")
border := layout.NewBorder(inner)
border.BaseComponent.VAlign = oat.VAlignMiddle
hbox.AddFlexChild(border, 1)
```

## Common patterns

### Centred button row

```go
btnRow := layout.NewHBox(cancelBtn, layout.NewHGap(2), okBtn)

vbox := layout.NewVBox(
    layout.NewFlexChild(contentArea),
    layout.NewAlignChild(btnRow, oat.HAlignCenter, oat.VAlignFill),
)
```

### Bottom-aligned status chip

```go
row := layout.NewHBox(
    layout.NewFlexChild(nameText, 1),
    layout.NewAlignChild(statusChip, oat.HAlignFill, oat.VAlignBottom),
)
```

### Mixed alignment

```go
vbox := layout.NewVBox(
    layout.NewAlignChild(widget.NewText("Welcome"), oat.HAlignCenter, oat.VAlignFill),
    layout.NewFlexChild(bodyContent),
    layout.NewAlignChild(widget.NewText("Sign in →"), oat.HAlignRight, oat.VAlignFill),
)
```
