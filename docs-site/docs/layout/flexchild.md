---
sidebar_position: 7
title: FlexChild
description: Wrap any component as a flex slot for inline use in variadic VBox / HBox constructors.
---

# FlexChild

`layout.FlexChild` wraps any `Component` as a flex slot. It is a convenience type that lets you pass flex children directly to the variadic `NewVBox` / `NewHBox` constructors without needing a separate `AddFlexChild` call after construction.

## Constructor

```go
layout.NewFlexChild(child oat.Component, weight ...int) *FlexChild
```

- `weight` is variadic and defaults to `1`. The minimum effective weight is `1`.

## Basic usage

```go
// Two-step pattern (without FlexChild):
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

## Explicit weight

```go
hbox := layout.NewHBox(
    layout.NewFlexChild(leftPanel, 1),
    layout.NewFlexChild(rightPanel, 3),  // right panel gets 3× the space
)
```

## When to use FlexChild vs AddFlexChild

They are equivalent in effect.

| Prefer | When |
|---|---|
| `NewFlexChild` | Building the child list inline (passing to a variadic constructor) |
| `AddFlexChild` | Adding a flex child to an already-constructed box |

## Theme and focus propagation

`FlexChild` implements `oat.Layout` via `Children()`, so theme propagation and focus collection recurse into the wrapped component automatically.

## Combining with AlignChild

`FlexChild` and `AlignChild` are composable. To have a flex child that is also cross-axis-aligned:

```go
// Flex child that is also right-aligned in its slot:
vbox.AddFlexChild(
    layout.NewAlignChild(myWidget, oat.HAlignRight, oat.VAlignFill),
    1,
)

// Or inline:
vbox := layout.NewVBox(
    layout.NewFlexChild(
        layout.NewAlignChild(myWidget, oat.HAlignRight, oat.VAlignFill),
    ),
    btnRow,
)
```

See [AlignChild](./alignchild.md) for details.
