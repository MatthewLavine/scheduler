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

		fmt.Printf("Core %d running task %d\n", c.id, task.id)

		var work int
		if task.work < int(c.timeSlice.Milliseconds()) {
			work = task.work
		} else {
			work = min(int(c.timeSlice.Milliseconds()), task.work)
		}
		time.Sleep(time.Duration(work))

		fmt.Printf("Core %d did %d work for task %d\n", c.id, work, task.id)
		fmt.Printf("Task %d has %d work left\n", task.id, task.work-work)
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
		fmt.Println("Scheduler loop start")
		for _, core := range s.cores {
			fmt.Printf("Core: %#v\n", core)
			fmt.Printf("Core runqueue: %#v\n", core.runqueue)
		}
		allCompleted := true
		for _, task := range s.tasks {
			fmt.Printf("Task: %#v\n", task)
			if !task.completed {
				allCompleted = false
				if !task.dispatched {
					task.dispatched = true
					for _, core := range s.cores {
						if core.state == Idle {
							fmt.Printf("Dispatching task %d to core %d\n", task.id, core.id)
							core.runqueue <- task
							break
						}
					}
				}
			}
		}
		if allCompleted {
			fmt.Println("All tasks completed")
			break
		}
		time.Sleep(s.timeSlice)
	}

	s.Shutdown()
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
		/* numCores= */ 2,
	)

	for i := 0; i < 5; i++ {
		scheduler.AddTask(NewTask(i /* work= */, 10000))
	}

	scheduler.Run()
	time.Sleep(1 * time.Second)
}
