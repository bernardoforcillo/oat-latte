---
sidebar_position: 3
title: Padding
description: Add blank space around any component.
---

# Padding

`layout.Padding` adds blank space around a single child component. It is the explicit alternative to setting `Padding` on a `latte.Style` — useful when you want to pad a component that does not expose a `WithStyle` builder, or when you want to keep style and spacing concerns separate.

## Constructors

```go
// Uniform padding — the same number of cells on all four sides.
padded := layout.NewPaddingUniform(child, 1)

// Asymmetric padding — control each side independently.
padded := layout.NewPadding(child, latte.Insets{
    Top:    1,
    Right:  2,
    Bottom: 1,
    Left:   2,
})
```

## When to use Padding vs Style.Padding

`latte.Style` has a `Padding` field that most widgets and `Border` will apply during render. Use that when the component already supports `WithStyle`.

Use `layout.Padding` when:
- The component does not expose a `WithStyle` builder.
- You want to pad a container (`VBox`, `HBox`, `ScrollView`) as a whole without adding a `Border`.
- You want to compose padding onto a component you did not write.

## Example

```go
// 1 cell of padding on all sides around a VBox.
content := layout.NewPaddingUniform(
    layout.NewVBox(
        widget.NewText("Line one"),
        widget.NewText("Line two"),
    ),
    1,
)

// 2 columns of left/right padding, 1 row top/bottom.
content := layout.NewPadding(myWidget, latte.Insets{
    Top: 1, Right: 2, Bottom: 1, Left: 2,
})
```
