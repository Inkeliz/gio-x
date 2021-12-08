// SPDX-License-Identifier: Unlicense OR MIT

package explorer

import (
	"errors"
	"gioui.org/app"
	"gioui.org/io/event"
	"gioui.org/io/plugins"
	"gioui.org/op"
	"io"
	"sync"
)

var (
	// ErrUserDecline is returned when the user doesn't select the file.
	ErrUserDecline = errors.New("user exit the file selector without selecting a file")

	// ErrNotAvailable is return when the current OS isn't supported.
	ErrNotAvailable = errors.New("current OS not supported")
)

type ReadOp struct {
	Tag        event.Tag
	Extensions []string
}

func (op ReadOp) Add(o *op.Ops) {
	plugins.PluginOp{ID: explorerIdentifier, Content: op}.Add(o)
}

type WriteOp struct {
	Tag  event.Tag
	Name string
}

func (op WriteOp) Add(o *op.Ops) {
	plugins.PluginOp{ID: explorerIdentifier, Content: op}.Add(o)
}

type ExportOp struct {
	Tag    event.Tag
	Name   string
	Reader io.ReadCloser
}

func (op ExportOp) Add(o *op.Ops) {
	plugins.PluginOp{ID: explorerIdentifier, Content: op}.Add(o)
}

type ImportOp struct {
	Tag        event.Tag
	Extensions []string
	Writer     io.WriteCloser
}

func (op ImportOp) Add(o *op.Ops) {
	plugins.PluginOp{ID: explorerIdentifier, Content: op}.Add(o)
}

var explorerIdentifier int

func init() {
	plugins.Register(&explorerIdentifier, func(w interface{}) plugins.Plugin {
		winMutex.Lock()
		window = w.(*app.Window)
		winMutex.Unlock()
		return &explorer{window: w.(*app.Window)}
	})
}

type ReadEvent struct {
	File io.ReadCloser
}

type WriteEvent struct {
	File io.WriteCloser
}

type SelectedEvent struct{}
type CancelEvent struct{}

func (r ReadEvent) ImplementsEvent()     {}
func (r WriteEvent) ImplementsEvent()    {}
func (r SelectedEvent) ImplementsEvent() {}
func (r CancelEvent) ImplementsEvent()   {}

type explorer struct {
	window *app.Window
	queue  event.Tag
}

func (e *explorer) Process(op interface{}) {
	switch op := op.(type) {
	case ReadOp:
		e.queue = op.Tag
		go func() {
			if f, err := ReadFile(op.Extensions...); err == nil {
				e.window.SendEvent(SelectedEvent{})
				e.window.SendEvent(ReadEvent{File: f})
			}
		}()
	case WriteOp:
		e.queue = op.Tag
		go func() {
			if f, err := WriteFile(op.Name); err == nil {
				e.window.SendEvent(SelectedEvent{})
				e.window.SendEvent(WriteEvent{File: f})
			}
		}()
	case ImportOp:
		e.queue = op.Tag
		go func() {
			if f, err := ReadFile(op.Extensions...); err == nil {
				e.window.SendEvent(SelectedEvent{})
				e.window.SendEvent(ReadEvent{File: f})
			}
		}()
	case ExportOp:
		e.queue = op.Tag
		go func() {
			if f, err := WriteFile(op.Name); err == nil {
				e.window.SendEvent(SelectedEvent{})

				if _, err := io.Copy(f, op.Reader); err != nil {

				}
				e.window.SendEvent(WriteEvent{File: f})
			}
		}()
	}
}

func (e *explorer) Push(event event.Event) (tag event.Tag, evt event.Event, ok bool) {
	listenEvents(event)

	switch event.(type) {
	case CancelEvent:
	case SelectedEvent:
	case ReadEvent:
	case WriteEvent:
	default:
		return tag, evt, ok
	}

	tag, evt, ok = e.queue, event, true
	e.queue = nil
	return tag, evt, ok
}

// ListenEventsWindow must get all the events from Gio, in order to get the GioView. You must
// include that function where you listen for Gio events.
//
// Similar as:
//
// select {
// case e := <-window.Events():
// 		explorer.ListenEvents(e)
// 		switch e := e.(type) {
// 				(( ... your code ...  ))
// 		}
// }
func ListenEventsWindow(win *app.Window, event event.Event) {
	winMutex.Lock()
	window = win
	listenEvents(event)
	winMutex.Unlock()
}

var (
	winMutex sync.Mutex
	window   *app.Window
)

// ReadFile shows the file selector, allowing the user to select a single file.
// Optionally, it's possible to define which file extensions is supported to
// be selected (such as `.jpg`, `.png`).
//
// Example: ReadFile(".jpg", ".png") will only accept the selection of files with
// .jpg or .png extensions.
//
// In some platforms the resulting `io.ReadCloser` is a `os.File`, but it's not
// a guarantee.
func ReadFile(extensions ...string) (io.ReadCloser, error) {
	return openFile(extensions...)
}

// WriteFile opens the file selector, and writes the given content into
// some file, which the use can choose the location.
//
// It's important to close the `io.WriteCloser`. In some platforms the
// file will be saved only when the writer is closer.
//
// In some platforms the resulting `io.WriteCloser` is a `os.File`, but it's not
// a guarantee.
func WriteFile(name string) (io.WriteCloser, error) {
	return createFile(name)
}
