package anagent

import (
	"fmt"
	"reflect"
	"strconv"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	agent := New()
	if reflect.TypeOf(agent).String() != "*anagent.Anagent" {
		t.Errorf("New() should return an anangent pointer: %v returned", agent)
	}
}

func TestEmitter(t *testing.T) {
	agent := New()

	fired := false
	agent.Emitter().On("test", func(v bool) { fired = v })
	agent.Emitter().Emit("test", true)
	if fired == false {
		t.Errorf("Expected event not fired")
	}
}

func TestUse(t *testing.T) {
	agent := New()

	fired := false
	agent.Use(func(a *Anagent) {
		fired = true
		a.Start()
		a.Stop()
	})

	agent.Start()
	if fired == false {
		t.Errorf("Agent middlewares are working and can stop the loop")
	}

	agent.Handlers()
	agent.Use(func(a *Anagent) {
		panic("OMG")
	})

	assertPanic(t, func() {
		agent.Start()
	})

	assertPanic(t, func() {
		agent.RunLoop()
	})

	fired = false
	agent.Handlers(func(a *Anagent) {
		fired = true
	})

	agent.Step()
	if fired == false {
		t.Errorf("Agent middlewares are working and can stop the loop")
	}

	assertPanic(t, func() {
		agent.Use("test")
	})
}

func TestAfter(t *testing.T) {
	agent := New()
	triggered := 0

	tid := agent.AddRecurringTimerSeconds(int64(1), func() {
		triggered++
	})

	agent.AddRecurringTimerSeconds(int64(3), func(a *Anagent) {
		timer := a.GetTimer(tid)
		timer.After(time.Duration(2 * time.Second))
	})

	agent.AddRecurringTimerSeconds(int64(6), func(a *Anagent) {
		timer := a.GetTimer(tid)
		if timer.after != time.Duration(2*time.Second) {
			t.Errorf("Timer was not set by the previous timer")
		}
		if triggered != 4 {
			t.Errorf("Timer was fired in not expected order! " + strconv.Itoa(triggered))
		}
		a.Stop()
	})

	agent.Start()
}

func TestTimerSeconds(t *testing.T) {
	agent := New()

	fired := false
	agent.AddTimerSeconds(int64(1), func(a *Anagent) {
		fired = true
		a.Stop()
	})

	agent.Start()
	if fired == false {
		t.Errorf("Agent middlewares are working and can stop the loop")
	}

	fired = false
	agent.TimerSeconds(int64(3), false, func(a *Anagent) {
		fired = true
		go a.Stop()
	})

	agent.Start()
	if fired == false {
		t.Errorf("Agent middlewares are working and can stop the loop")
	}
}

func TestRecurringTimer(t *testing.T) {
	agent := New()
	fired := 0
	agent.Emitter().On("Ping", func() { fmt.Println("PING") })
	tid := agent.AddRecurringTimerSeconds(int64(1), func(a *Anagent) {
		fired++
		go func() {
			a.Lock()
			defer a.Unlock()
			a.Emitter().Emit("Ping")
		}()
		if fired > 4 {
			a.Stop()
		}
	})

	agent.SetDuration(tid, time.Second)
	assertSleep(5.0, t, func() {
		agent.Start()
	})

	if fired != 5 {
		t.Errorf("Agent middlewares are working and can stop the loop")
	}

	agent.RemoveTimer(tid)
	fired = 0
	agent.AddRecurringTimerSeconds(int64(1), func(a *Anagent) {
		fired++
		if fired > 4 {
			a.Stop()
		}
	})

	assertSleep(5.0, t, func() {
		agent.Start()
	})
	if fired != 5 {
		t.Errorf("Agent middlewares are working and can stop the loop")
	}

}

type TestTest struct {
	Test string
}

func TestOn(t *testing.T) {
	agent := New()
	varr := &TestTest{Test: "W0h00"}
	fired := 0
	agent.Map(varr)
	agent.On("test", func(te *TestTest) {
		if te.Test != "W0h00" {
			t.Errorf("Cannot access to injections :(")
		}
		fired++
	})

	agent.Emit("test")
	agent.Emit("test")
	agent.Emit("test")
	agent.Emit("test")

	if fired != 4 {
		t.Errorf("Event not fired :(")
	}
}

func TestOnce(t *testing.T) {
	agent := New()
	varr := &TestTest{Test: "Just Once!"}
	fired := 0
	agent.Map(varr)
	agent.Once("test", func(te *TestTest) {
		if te.Test != "Just Once!" {
			t.Errorf("Cannot access to injections :(")
		}
		fired++
	})

	agent.Emit("test")
	agent.Emit("test")
	agent.Emit("test")
	agent.Emit("test")

	if fired != 1 {
		t.Errorf("Event not fired once :(")
	}
}

func TestEmitSync(t *testing.T) {
	agent := New()
	varr := &TestTest{Test: "Just Once?"}
	fired := 0
	agent.Map(varr)
	agent.Once("test", func(te *TestTest) {
		if te.Test != "Just Once?" {
			t.Errorf("Cannot access to injections :(")
		}
		fired++
	})
	agent.Once("test", func(te *TestTest) {
		if te.Test != "Just Once?" {
			t.Errorf("Cannot access to injections :(")
		}
		fired++
	})
	agent.Once("test", func(te *TestTest) {
		if te.Test != "Just Once?" {
			t.Errorf("Cannot access to injections :(")
		}
		fired++
	})
	agent.Once("test", func(te *TestTest) {
		if te.Test != "Just Once?" {
			t.Errorf("Cannot access to injections :(")
		}
		fired++
	})

	agent.EmitSync("test")

	if fired != 4 {
		t.Errorf("Event not fired once :(")
	}
}

func TestTimer(t *testing.T) {
	agent := New()
	var tid TimerID = "test"
	fired := 0
	agent.Timer(tid, time.Now(), time.Duration(5), true, func(a *Anagent) {
		fired++
		if fired > 4 {
			a.Stop()
		}
	})

	timer := agent.GetTimer(tid)

	if timer.recurring == false {
		t.Errorf("Timer should be recurring")
	}

	agent.Start()
	if fired != 5 {
		t.Errorf("Agent middlewares are working and can stop the loop")
	}
}

func TestNext(t *testing.T) {

	agent := New()
	agent.BusyLoop = true
	fired := 0
	loop := 0

	agent.Use(func(a *Anagent) {
		loop++
	})

	agent.Next(func(a *Anagent) {
		fired++
	})

	agent.AddTimerSeconds(int64(3), func(a *Anagent) {
		a.Stop()
	})

	agent.Start()

	if fired != 1 {
		t.Error("Next is removed from the loop")
	}

	if fired >= loop {
		t.Error("Loop had to run for long ", loop)
	}
}

func assertPanic(t *testing.T, f func()) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()
	f()
}

func assertSleep(secSleep float64, t *testing.T, f func()) {
	start := time.Now()
	f()
	sec := time.Since(start).Seconds()

	if sec < secSleep || sec > secSleep*1.05 {
		t.Error("Timer wasn't fired in the specified time")
	}
}
