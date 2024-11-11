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
	fmt.Printf("Core %d starting up\n", c.id)
	for task := range c.runqueue {
		c.state = Running

		var work int
		if task.work < int(c.timeSlice.Milliseconds()) {
			work = task.work
		} else {
			work = min(int(c.timeSlice.Milliseconds()), task.work)
		}
		time.Sleep(time.Duration(work))

		task.work -= work
		if task.work == 0 {
			task.completed = true
		}

		c.state = Idle
		task.dispatched = false
	}
	fmt.Printf("Core %d shutting down\n", c.id)
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
		time.Sleep(s.timeSlice)
	}

	s.Shutdown()
}

func clearScreen() {
	fmt.Print("\033[H\033[2J")
}

func (s *Scheduler) LogProgress() {
	clearScreen()
	for _, task := range s.tasks {
		fmt.Printf("Task %d: %d/%d\n", task.id, task.initialWork-task.work, task.initialWork)
	}
	for _, core := range s.cores {
		fmt.Printf("Core %d: %d\n", core.id, core.state)
	}
}

func (s *Scheduler) Shutdown() {
	fmt.Println("Scheduler shutting down")
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
		/* timeSlice= */ 1*time.Second,
		/* numCores= */ 4,
	)

	for i := 0; i < 6; i++ {
		scheduler.AddTask(NewTask(i /* work= */, 10000))
	}

	scheduler.Run()
	time.Sleep(1 * time.Second)
}
