---
sidebar_position: 5
title: Grid
description: Fixed rows × columns grid layout with equal cell sizes and optional gaps.
---

# Grid

`layout.Grid` positions children in a fixed-size rows × columns grid. All cells have equal width and height. Children can span multiple rows or columns.

## Constructor

```go
g := layout.NewGrid(rows, cols int) *Grid
```

## Adding children

```go
g.AddChildAt(child oat.Component, row, col, rowSpan, colSpan int)
```

- `row`, `col` — zero-indexed position of the top-left corner of the child.
- `rowSpan`, `colSpan` — how many cells the child occupies.

```go
g := layout.NewGrid(2, 3)              // 2 rows, 3 cols

g.AddChildAt(titleWidget, 0, 0, 1, 3) // spans all 3 columns in row 0
g.AddChildAt(leftWidget,  1, 0, 1, 1) // row 1, col 0, 1×1
g.AddChildAt(midWidget,   1, 1, 1, 1) // row 1, col 1, 1×1
g.AddChildAt(rightWidget, 1, 2, 1, 1) // row 1, col 2, 1×1
```

## Gap between cells

```go
g.WithGap(rowGap, colGap int)
```

```go
g := layout.NewGrid(2, 3)
g.WithGap(0, 1)   // no row gap, 1-column gap between columns
```

## When to use Grid vs HBox/VBox nesting

`Grid` is the right choice when you need children to align across both axes simultaneously — for example, a form where labels in one column need to line up with inputs in another, and both rows and columns must stay aligned.

For most layouts, nested `VBox` and `HBox` containers are sufficient and more flexible. Use `Grid` only when the cross-axis alignment requirement makes box nesting awkward.
