# Anagent [![Build Status](https://travis-ci.org/mudler/anagent.svg?branch=master)](https://travis-ci.org/mudler/anagent) [![codecov](https://codecov.io/gh/mudler/anagent/branch/master/graph/badge.svg)](https://codecov.io/gh/mudler/anagent) [![Go Report Card](https://goreportcard.com/badge/github.com/mudler/anagent)](https://goreportcard.com/report/github.com/mudler/anagent) [![godoc](https://godoc.org/github.com/mudler/anagent?status.svg)](http://godoc.org/github.com/mudler/anagent)

Minimalistic, pluggable Golang evloop/timer handler with dependency-injection - based on [codegangsta/inject](github.com/codegangsta/inject) - [go-macaron/inject](github.com/go-macaron/inject) and [chuckpreslar/emission](https://github.com/chuckpreslar/emission).

*Anagent* is a lightweight library that allows you to plug inside to other event loops, or allows you to handle and create your own within your application - leaving the control to you.

It comes with dependency-injection from codegangsta/inject, and it's also a soft-wrapper to chuckpreslar/emission, adding to it dependency injection capabilities and timer handlers.

## Usage

### Event Emitter with Dependency injection

```go
    package main

    import (
    	"log"
    	"github.com/mudler/anagent"
    )

    type TestTest struct {
    	Test string
    }

    func main() {
    	agent := anagent.New()
    	mytest := &TestTest{Test: "PONG!"}
    	agent.Map(mytest)

    	agent.Once("test", func(te *TestTest, l *log.Logger) {
    		if te.Test == "PONG!" {
    			l.Println("It just works!")
    		}
    	})

    	agent.Emit("test")
    }
```

What happened here? we mapped our structure instance (```TestTest```) inside the agent with (```agent.Map()```), and all fired events can access to them.

### Timer / Reactor

```go
    package main

    import "github.com/mudler/anagent"
    import "fmt"

    type TestTest struct {
            Test string
    }

    func main() {
            agent := anagent.New()
            mytest := &TestTest{Test: "PONG!"}
            agent.Map(mytest)

            agent.Emitter().On("test", func(s string) { fmt.Println("Received: " + s) })

            // Not recurring timer
            agent.TimerSeconds(int64(3), false, func(a *anagent.Anagent, te *TestTest) {
                    a.Emitter().Emit("test", te.Test)
                    go a.Stop()
            })

            agent.Start() // Loops here and never returns
    }
 ```

The code portion will start and wait for 3 seconds, then it will execute the callback (not recurring, that's why the ```false```) that will fire a custom event defined before (note, it's not using the dependency-injection capabilities, thus it's accessing the emitter handler directly with ```agent.Emitter()```).

The difference is that when we access to ```On()``` provided by ```agent.On()```, we access to the agent dependencies, that have been mapped with ```agent.Map()``` - otherwise, with ```agent.Emitter().On()``` we are free to bind any arguments to the event callback.


After the event is fired, the timer stops the eventloop (```a.Stop()```), so the program returns.

### Hook into other loops

It is often in other framework to use loop patterns, as example in framework for game development, network agents, and such.
We can hook into other loops, and run the agent Step function, so we can still leverage the evloop functionalities.

```go
    package main

    import "github.com/mudler/anagent"
    import "fmt"

    type TestTest struct {
    	Test string
    }

    func main() {
    	agent := anagent.New()
    	mytest := &TestTest{Test: "PONG!"}
    	agent.Map(mytest)

        // Reply with a delay of 2s
    	agent.Emitter().On("test", func(s string) {
    		agent.TimerSeconds(int64(2), false, func() {
    			fmt.Println("Received: " + s)
    		})
    	})

    	// Recurring timer
    	agent.TimerSeconds(int64(3), true, func(a *anagent.Anagent, te *TestTest) {
    		fmt.Println("PING!")
    		a.Emitter().Emit("test", te.Test)
    	})

    	for { // Infinite loop
    		agent.Step()
    	}
    }
 ```
