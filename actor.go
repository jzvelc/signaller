package main

type ActorGroup struct {
	actors []actor
}

func (g *ActorGroup) Add(execute func() error, interrupt func(error)) {
	g.actors = append(g.actors, actor{execute, interrupt})
}

func (g *ActorGroup) Reset() {
	g.actors = make([]actor, 0)
}

func (g *ActorGroup) RunUntilError(minSatisfy int) error {
	if len(g.actors) == 0 {
		return nil
	}

	if minSatisfy == 0 {
		minSatisfy = len(g.actors)
	}

	// Run each actor
	errors := make(chan error, len(g.actors))
	for _, a := range g.actors {
		go func(a actor) {
			errors <- a.execute()
		}(a)
	}

	// Wait for the first actor to return error
	var err error
	var j int
	for {
		err = <-errors
		j += 1
		if err != nil || j == minSatisfy {
			break
		}
	}

	// Signal all actors to stop
	for _, a := range g.actors {
		a.interrupt(err)
	}

	// Wait for all actors to stop
	for i := j; i < cap(errors); i++ {
		<-errors
	}

	g.Reset()

	// Return the original error
	return err
}

type actor struct {
	execute   func() error
	interrupt func(error)
}
