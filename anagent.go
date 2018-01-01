// Copyright 2017-2018 Ettore Di Giacinto
//
// Permission is hereby granted, free of charge, to any person obtaining
// a copy of this software and associated documentation files (the "Software"),
// to deal in the Software without restriction, including without limitation
// the rights to use, copy, modify, merge, publish, distribute, sublicense,
// and/or sell copies of the Software, and to permit persons to whom the
// Software is furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included
// in all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
// EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES
// OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
// IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM,
// DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
// TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE
// OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package anagent

import (
	"io"
	"log"
	"os"
	"reflect"
	"time"

	"github.com/chuckpreslar/emission"
	"github.com/go-macaron/inject"
)

const _VERSION = "0.1"

// Handler can be any callable function.
// Anagent attempts to inject services into the handler's argument list,
// and panics if an argument could not be fullfilled via dependency injection.
type Handler interface{}
type TimerID string

type Timer struct {
	time      time.Time
	after     time.Duration
	handler   Handler
	recurring bool
}

func Version() string {
	return _VERSION
}

// Anagent represents the top level web application.
// inject.Injector methods can be invoked to map services on a global level.
type Anagent struct {
	inject.Injector

	handlers          []Handler
	parallel_handlers []Handler
	timers            map[TimerID]*Timer

	logger   *log.Logger
	ee       *emission.Emitter
	Parallel bool
	Fatal    bool
	Started  bool
}

// Handlers sets the entire middleware stack with the given Handlers.
// This will clear any current middleware handlers,
// and panics if any of the handlers is not a callable function
func (a *Anagent) Handlers(handlers ...Handler) {
	a.handlers = make([]Handler, 0)
	for _, handler := range handlers {
		a.Use(handler)
	}
}

func (a *Anagent) ParallelHandlers(handlers ...Handler) {
	a.parallel_handlers = make([]Handler, 0)
	for _, handler := range handlers {
		a.UseParallel(handler)
	}
}

// validateAndWrapHandler makes sure a handler is a callable function, it panics if not.
// When the handler is also potential to be any built-in inject.FastInvoker,
// it wraps the handler automatically to have some performance gain.
func validateAndWrapHandler(h Handler) Handler {
	if reflect.TypeOf(h).Kind() != reflect.Func {
		panic("Anagent handler must be a callable function")
	}
	return h
}

// Use adds a middleware Handler to the stack,
// and panics if the handler is not a callable func.
// Middleware Handlers are invoked in the order that they are added.
func (a *Anagent) Use(handler Handler) {
	handler = validateAndWrapHandler(handler)
	a.handlers = append(a.handlers, handler)
}

func (a *Anagent) TimerSeconds(seconds int64, recurring bool, handler Handler) TimerID {
	id := TimerID(GetMD5Hash(time.Now().String()))
	handler = validateAndWrapHandler(handler)
	dt := time.Duration(seconds) * time.Second
	t := &Timer{handler: handler, time: time.Now().Add(dt), after: dt, recurring: recurring}
	a.timers[id] = t
	return id
}

func (a *Anagent) Timer(tid TimerID, ti time.Time, after time.Duration, recurring bool, handler Handler) TimerID {

	var id TimerID
	if tid != "" {
		id = tid
	} else {
		id = TimerID(GetMD5Hash(time.Now().String()))
	}

	handler = validateAndWrapHandler(handler)
	t := &Timer{handler: handler, time: ti, after: after, recurring: recurring}
	a.timers[id] = t
	return id
}

func (a *Anagent) SetDuration(id TimerID, after time.Duration) TimerID {
	a.timers[id].after = after
	return id
}

func (a *Anagent) AddTimerSeconds(seconds int64, handler Handler) TimerID {
	return a.TimerSeconds(seconds, false, handler)
}

func (a *Anagent) AddRecurringTimerSeconds(seconds int64, handler Handler) TimerID {
	return a.TimerSeconds(seconds, true, handler)
}

func (a *Anagent) UseParallel(handler Handler) {
	handler = validateAndWrapHandler(handler)
	a.parallel_handlers = append(a.parallel_handlers, handler)
}

// NewWithLogger creates a bare bones Anagent instance.
// Use this method if you want to have full control over the middleware that is used.
// You can specify logger output writer with this function.
func NewWithLogger(out io.Writer) *Anagent {
	ts := make(map[TimerID]*Timer)
	a := &Anagent{
		Injector: inject.New(),
		logger:   log.New(out, "[Anagent] ", log.Ldate|log.Ltime),
		ee:       emission.NewEmitter(),
		Parallel: false,
		timers:   ts,
	}
	a.Map(a.logger)
	a.Map(a.ee)

	return a
}

// New creates a bare bones Anagent instance.
// Use this method if you want to have full control over the middleware that is used.
func New() *Anagent {
	return NewWithLogger(os.Stdout)
}

func (a *Anagent) runAll(h *[]Handler, parallel bool) {
	var i = 0
	for i < len(*h) {
		var err error
		if parallel {

			go func(a *Anagent, hs *[]Handler, n int) {
				// return and error
				_, err = a.Invoke((*hs)[n])
			}(a, h, i)

		} else {
			_, err = a.Invoke((*h)[i]) // was vals
		}

		if err != nil && a.Fatal {
			panic(err)
		}

		i++
	}
}

func (a *Anagent) RunLoop() {
	for {
		a.Step()
	}
}

func (a *Anagent) Start() {
	if a.Started == true {
		return
	}
	a.Started = true
	for a.Started {
		a.Step()
	}
}

func (a *Anagent) Stop() {
	a.Started = false
}

func (a *Anagent) Step() {
	a.runAll(&a.handlers, a.Parallel)
	a.runAll(&a.parallel_handlers, true)

	if len(a.timers) == 0 {
		return
	}

	mintime := time.Now()
	var mintimeid TimerID
	for timerid, t := range a.timers {
		if t.time.Before(mintime) {
			mintime = t.time
			mintimeid = timerid
		}
	}
	if mintimeid == "" {
		return
	}

	now := time.Now()

	if mintime.After(now) {
		time.Sleep(mintime.Sub(now))
	}
	a.Invoke(a.timers[mintimeid].handler)

	if a.timers[mintimeid].recurring == true {
		a.timers[mintimeid].time = time.Now().Add(a.timers[mintimeid].after)
	} else {
		delete(a.timers, mintimeid)
	}
}
