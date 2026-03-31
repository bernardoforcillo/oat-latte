---
sidebar_position: 4
title: Layout
description: Overview of oat-latte layout containers — VBox, HBox, Border, Padding, ScrollView, Grid, spacers, FlexChild, AlignChild.
---

# Layout

Layout containers hold children and handle all size negotiation between them. Every container implements `Component` (so it can be nested anywhere) and `Layout` (so the framework can walk its children for theme propagation and focus collection).

## Containers

| Container | Purpose |
|---|---|
| [VBox and HBox](./layout/vbox-hbox.md) | Stack children vertically or horizontally; the primary building blocks |
| [Border](./layout/border.md) | Wrap a child with a configurable box border and optional title |
| [Padding](./layout/padding.md) | Add blank space around a child |
| [ScrollView](./layout/scrollview.md) | Clip a child to a viewport with vertical scrolling |
| [Grid](./layout/grid.md) | Rows × columns grid with equal cell sizes |

## Spacers and wrappers

| Type | Purpose |
|---|---|
| [Spacers (VFill, HFill, VGap, HGap)](./layout/spacers.md) | Push, fill, or add fixed gaps between children |
| [FlexChild](./layout/flexchild.md) | Wrap any component as a flex slot for inline use in variadic constructors |
| [AlignChild](./layout/alignchild.md) | Per-child cross-axis alignment override |

## Two-pass render pipeline

Every container follows the same two-pass contract:

1. **Measure** — the parent calls `Measure(constraint)` on each child to learn its desired size. `Constraint.MaxWidth` / `MaxHeight` are the available cells; `-1` means unconstrained.
2. **Render** — the parent calls `Render(buf, region)` on each child with its allocated `Region`. The child draws into `buf` clipped to that region.

Never skip Measure before Render. Never store `buf` or `region` across frames.

## AddChild vs AddFlexChild

The core sizing decision in every box layout:

- **`AddChild(c)`** — natural size. The child takes exactly what `Measure` returns.
- **`AddFlexChild(c, weight)`** — flex distribution. Leftover space is divided among flex children proportionally by weight.

See [VBox and HBox](./layout/vbox-hbox.md) for a full explanation with diagrams.

## Import path

```go
import "github.com/antoniocali/oat-latte/layout"
```
