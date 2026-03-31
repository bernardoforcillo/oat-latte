---
sidebar_position: 8
title: AlignChild
description: Per-child cross-axis alignment override for VBox and HBox children.
---

# AlignChild

`layout.AlignChild` wraps a single component with explicit cross-axis alignment — the per-child counterpart to `VBox.WithHAlign` / `HBox.WithVAlign`. Use it when one child in a box needs different alignment from the rest, or when the child component does not expose its own `WithHAlign`/`WithVAlign` builder.

## Constructor

```go
func NewAlignChild(child oat.Component, h oat.HAlign, v oat.VAlign) *AlignChild
```

- `h` — `oat.HAlignFill`, `HAlignLeft`, `HAlignCenter`, or `HAlignRight`
- `v` — `oat.VAlignFill`, `VAlignTop`, `VAlignMiddle`, or `VAlignBottom`

Pass `oat.HAlignFill` or `oat.VAlignFill` for the axis you do not want to constrain.

## Basic usage

```go
// Save pinned right, Cancel pinned left — no spacers needed.
vbox := layout.NewVBox(
    layout.NewAlignChild(saveBtn,   oat.HAlignRight, oat.VAlignFill),
    layout.NewAlignChild(cancelBtn, oat.HAlignLeft,  oat.VAlignFill),
)
```

## Priority

`AlignChild` takes **highest priority** in the alignment resolution order:

1. `AlignChild` wrapper — wins unconditionally.
2. Child's own `BaseComponent.HAlign` / `VAlign` (if non-fill).
3. Box-wide default (`WithHAlign` / `WithVAlign` on the box).
4. `HAlignFill` / `VAlignFill` fallback.

## Scope — direct children only

Like all alignment, `AlignChild` is only honoured on the **direct child** of the distributing box. If you wrap a `Border` in an `AlignChild` and place it in an `HBox`, the `HBox` reads the `AlignChild`'s `VAlign`. The inner widget's alignment is irrelevant to the `HBox`.

## Combining with flex

`AlignChild` is **not** a `FlexSpacer` — it does not claim flex space on its own. Wrap it with `AddFlexChild` or `NewFlexChild` when you also need it to participate in flex distribution:

```go
// Flex child that is also right-aligned in its slot:
vbox.AddFlexChild(
    layout.NewAlignChild(headerWidget, oat.HAlignRight, oat.VAlignFill),
    1,
)

// Inline using NewFlexChild:
vbox := layout.NewVBox(
    layout.NewFlexChild(
        layout.NewAlignChild(headerWidget, oat.HAlignRight, oat.VAlignFill),
    ),
    btnRow,
)
```

## Theme and focus propagation

`AlignChild` implements `oat.Layout` via `Children()`, so theme propagation and focus collection recurse into the wrapped component automatically — exactly like `FlexChild`.

## Examples

### Bottom-aligned status chip in an HBox

```go
row := layout.NewHBox(
    layout.NewFlexChild(nameText, 1),
    layout.NewAlignChild(statusChip, oat.HAlignFill, oat.VAlignBottom),
)
```

### Centred button row in a VBox

```go
btnRow := layout.NewHBox(cancelBtn, layout.NewHGap(2), okBtn)

vbox := layout.NewVBox(
    layout.NewFlexChild(contentArea),
    layout.NewAlignChild(btnRow, oat.HAlignCenter, oat.VAlignFill),
)
```

### Mixed alignment in one VBox

```go
vbox := layout.NewVBox(
    layout.NewAlignChild(widget.NewText("Welcome"), oat.HAlignCenter, oat.VAlignFill),
    layout.NewFlexChild(bodyContent),
    layout.NewAlignChild(widget.NewText("Sign in →"), oat.HAlignRight, oat.VAlignFill),
)
```
