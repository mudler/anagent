package anagent

import (
	"fmt"
	"reflect"
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
	agent.Emitter().On("test", func() { fired = true })
	agent.Emitter().Emit("test")
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
