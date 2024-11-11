package main

import (
	"fmt"
	"time"
)

const (
	Idle = iota
	Running
	Stopped
)

type Task struct {
	id          int
	initialWork int
	work        int
	dispatched  bool
	completed   bool
}

type Core struct {
	id        int
	state     int
	timeSlice time.Duration
	runqueue  chan *Task
	current   *Task
}

type Scheduler struct {
	timeSlice time.Duration
	numCores  int

	cores []*Core
	tasks []*Task
}

func NewTask(id int, work int) *Task {
	return &Task{
		id:          id,
		initialWork: work,
		work:        work,
		dispatched:  false,
		completed:   false,
	}
}

func NewCore(id int, timeSlice time.Duration) *Core {
	return &Core{
		id:        id,
		state:     Idle,
		timeSlice: timeSlice,
		runqueue:  make(chan *Task),
	}
}

func (c *Core) Run() {
	for task := range c.runqueue {
		c.current = task
		c.state = Running

		var work int
		if task.work < int(c.timeSlice.Milliseconds()) {
			work = task.work
		} else {
			work = min(int(c.timeSlice.Milliseconds()), task.work)
		}
		time.Sleep(time.Duration(work) * time.Millisecond)

		task.work -= work
		if task.work == 0 {
			task.completed = true
		}

		c.state = Idle
		c.current = nil
		task.dispatched = false
	}
	c.state = Stopped
}

func (c *Core) Shutdown() {
	close(c.runqueue)
}

func NewScheduler(timeSlice time.Duration, numCores int) *Scheduler {
	return &Scheduler{
		timeSlice: timeSlice,
		numCores:  numCores,
		cores:     make([]*Core, numCores),
		tasks:     make([]*Task, 0),
	}
}

func (s *Scheduler) AddTask(task *Task) {
	s.tasks = append(s.tasks, task)
}

func (s *Scheduler) Run() {
	fmt.Println("Scheduler starting up")

	fmt.Println("Starting cores")
	for i := 0; i < s.numCores; i++ {
		core := NewCore(i, s.timeSlice)
		s.cores[i] = core
		go core.Run()
	}

	fmt.Println("Dispatching tasks")

	for {
		allCompleted := true
		for _, task := range s.tasks {
			if !task.completed {
				allCompleted = false
				if !task.dispatched {
					for _, core := range s.cores {
						if core.state == Idle {
							task.dispatched = true
							core.runqueue <- task
							break
						}
					}
				}
			}
		}
		s.LogProgress()
		if allCompleted {
			fmt.Println("All tasks completed")
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	s.Shutdown()
}

func clearScreen() {
	fmt.Print("\033[H\033[2J")
}

func (s *Scheduler) LogProgress() {
	clearScreen()
	for _, task := range s.tasks {
		task := *task
		progress := float64(task.initialWork-task.work) / float64(task.initialWork) * 100
		fmt.Printf("Task %d: %05.2f%% complete, progress %d/%d\n", task.id, progress, task.initialWork-task.work, task.initialWork)
	}
	for _, core := range s.cores {
		core := *core
		switch core.state {
		case Idle:
			fmt.Printf("Core %d: Idle\n", core.id)
		case Running:
			fmt.Printf("Core %d: Running task %d\n", core.id, core.current.id)
		case Stopped:
			fmt.Printf("Core %d: Stopped\n", core.id)
		}
	}
}

func (s *Scheduler) Shutdown() {
	for _, core := range s.cores {
		core.Shutdown()
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func main() {
	// Start the scheduler
	scheduler := NewScheduler(
		/* timeSlice= */ 10*time.Millisecond,
		/* numCores= */ 2,
	)

	for i := 0; i < 6; i++ {
		scheduler.AddTask(NewTask(i /* work= */, 1000))
	}

	scheduler.Run()
}
