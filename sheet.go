package materials

import (
	"image"
	"image/color"
	"time"

	"gioui.org/f32"
	"gioui.org/gesture"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
)

// Sheet implements the standard side sheet described here:
// https://material.io/components/sheets-side#usage
type Sheet struct {
	Background color.RGBA
}

// NewSheet returns a sheet with its background color initialized to white.
func NewSheet() Sheet {
	return Sheet{
		Background: color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff},
	}
}

// Layout renders the provided widget on a background. The background will use
// the maximum space available.
func (s Sheet) Layout(gtx layout.Context, anim *VisibilityAnimation, widget layout.Widget) layout.Dimensions {
	defer op.Push(gtx.Ops).Pop()

	revealed := -1 + anim.Revealed(gtx)
	finalOffset := revealed * (float32(gtx.Constraints.Max.X))
	revealedWidth := finalOffset + float32(gtx.Constraints.Max.X)
	op.Offset(f32.Point{X: finalOffset}).Add(gtx.Ops)
	// lay out background
	paintRect(gtx, gtx.Constraints.Max, s.Background)

	// lay out sheet contents
	dims := widget(gtx)

	return layout.Dimensions{
		Size: image.Point{
			X: int(revealedWidth),
			Y: gtx.Constraints.Max.Y,
		},
		Baseline: dims.Baseline,
	}
}

// ModalSheet implements the Modal Side Sheet component
// specified at https://material.io/components/sheets-side#modal-side-sheet
type ModalSheet struct {
	// MaxWidth constrains the maximum amount of horizontal screen real-estate
	// covered by the drawer. If the screen is narrower than this value, the
	// width will be inferred by reserving space for the scrim and using the
	// leftover area for the drawer. Values between 200 and 400 Dp are recommended.
	//
	// The default value used by NewModalNav is 400 Dp.
	MaxWidth unit.Value

	Modal *ModalLayer

	drag gesture.Drag

	// animation state
	dragging    bool
	dragStarted f32.Point
	dragOffset  float32
}

// NewModalSheet creates a modal sheet that can render a widget on the modal layer.
func NewModalSheet(m *ModalLayer) *ModalSheet {
	s := &ModalSheet{
		MaxWidth: unit.Dp(400),
		Modal:    m,
	}
	return s
}

// updateDragState ensures that a partially-dragged sheet
// snaps back into place when released and otherwise chooses
// when the sheet has been dragged far enough to close.
func (s *ModalSheet) updateDragState(gtx layout.Context, anim *VisibilityAnimation) {
	if s.dragOffset != 0 && !s.dragging && !anim.Animating() {
		if s.dragOffset < 2 {
			s.dragOffset = 0
		} else {
			s.dragOffset /= 2
		}
	} else if s.dragging && int(s.dragOffset) > gtx.Constraints.Max.X/10 {
		anim.Disappear(gtx.Now)
	}
}

// ConfigureModal requests that the sheet prepare the associated
// ModalLayer to render itself (rather than another modal widget).
func (s *ModalSheet) LayoutModal(contents func(gtx layout.Context, anim *VisibilityAnimation) layout.Dimensions) {
	s.Modal.Widget = func(gtx C, anim *VisibilityAnimation) D {
		s.updateDragState(gtx, anim)
		if !anim.Visible() {
			return D{}
		}
		for _, event := range s.drag.Events(gtx.Metric, gtx.Queue, gesture.Horizontal) {
			switch event.Type {
			case pointer.Press:
				s.dragStarted = event.Position
				s.dragOffset = 0
				s.dragging = true
			case pointer.Drag:
				newOffset := s.dragStarted.X - event.Position.X
				if newOffset > s.dragOffset {
					s.dragOffset = newOffset
				}
			case pointer.Release:
				fallthrough
			case pointer.Cancel:
				s.dragging = false
			}
		}
		if s.dragOffset != 0 || anim.Animating() {
			defer op.Push(gtx.Ops).Pop()
			s.drawerTransform(gtx, anim).Add(gtx.Ops)
			op.InvalidateOp{}.Add(gtx.Ops)
		}
		gtx.Constraints.Max.X = s.sheetWidth(gtx)

		// Beneath sheet content, listen for tap events. This prevents taps in the
		// empty sheet area from passing downward to the scrim underneath it.
		pointer.PassOp{Pass: false}.Add(gtx.Ops)
		pointer.Rect(image.Rectangle{Max: gtx.Constraints.Max}).Add(gtx.Ops)
		pointer.InputOp{
			Tag:   s,
			Types: pointer.Press | pointer.Release,
		}.Add(gtx.Ops)

		// lay out widget
		dims := NewSheet().Layout(gtx, anim, func(gtx C) D {
			return contents(gtx, anim)
		})

		// On top of sheet content, listen for drag events to close the sheet.
		pointer.PassOp{Pass: true}.Add(gtx.Ops)
		pointer.Rect(image.Rectangle{Max: gtx.Constraints.Max}).Add(gtx.Ops)
		s.drag.Add(gtx.Ops)

		return dims
	}
}

// drawerTransform returns the current offset transformation
// of the sheet taking both drag and animation progress
// into account.
func (s ModalSheet) drawerTransform(gtx C, anim *VisibilityAnimation) op.TransformOp {
	finalOffset := -s.dragOffset
	return op.Offset(f32.Point{X: finalOffset})
}

// sheetWidth returns the width of the sheet taking both the dimensions
// of the modal layer and the MaxWidth field into account.
func (s ModalSheet) sheetWidth(gtx layout.Context) int {
	scrimWidth := gtx.Px(unit.Dp(56))
	withScrim := gtx.Constraints.Max.X - scrimWidth
	max := gtx.Px(s.MaxWidth)
	return min(withScrim, max)
}

// ToggleVisibility triggers the appearance or disappearance of the
// ModalSheet.
func (s *ModalSheet) ToggleVisibility(when time.Time) {
	s.Modal.ToggleVisibility(when)
}

// Appear triggers the appearance of the ModalSheet.
func (s *ModalSheet) Appear(when time.Time) {
	s.Modal.Appear(when)
}

// Disappear triggers the appearance of the ModalSheet.
func (s *ModalSheet) Disappear(when time.Time) {
	s.Modal.Disappear(when)
}
