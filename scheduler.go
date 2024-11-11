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
	startTime   time.Time
	endTime     time.Time
	runTime     time.Duration
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
	runqueue  int

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
		startTime:   time.Now(),
	}
}

func NewCore(id int, timeSlice time.Duration) *Core {
	return &Core{
		id:        id,
		state:     Idle,
		timeSlice: timeSlice,
		runqueue:  make(chan *Task, 10),
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
			task.endTime = time.Now()
			task.runTime = task.endTime.Sub(task.startTime)
		}

		c.state = Idle
		c.current = nil
		task.dispatched = false
	}
	c.state = Stopped
}

func (c *Core) Shutdown() {
	close(c.runqueue)
	c.state = Stopped
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

func (s *Scheduler) PickCore() *Core {
	for _, core := range s.cores {
		if core.state == Idle {
			return core
		}
	}
	return nil
}

func (s *Scheduler) Run() {
	for i := 0; i < s.numCores; i++ {
		core := NewCore(i, s.timeSlice)
		s.cores[i] = core
		go core.Run()
	}

	for {
		allCompleted := true
		s.runqueue = 0
		for _, task := range s.tasks {
			if !task.completed {
				allCompleted = false
				if !task.dispatched {
					core := s.PickCore()
					if core == nil {
						s.runqueue++
					} else {
						task.dispatched = true
						core.runqueue <- task
					}
				}
			}
		}
		s.LogProgress()
		if allCompleted {
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
	fmt.Println("---- Task Progress ----")
	for _, task := range s.tasks {
		task := *task
		progress := float64(task.initialWork-task.work) / float64(task.initialWork) * 100
		if task.completed {
			fmt.Printf("Task %d: 100%% complete, runtime %v\n", task.id, task.runTime.Round(time.Millisecond))
		} else {
			fmt.Printf("Task %d: %06.2f%% complete, remaining %d\n", task.id, progress, task.work)
		}
	}
	fmt.Println("---- Core Progress ----")
	for _, core := range s.cores {
		core := *core
		switch core.state {
		case Idle:
			fmt.Printf("Core %d: Idle\n", core.id)
		case Running:
			fmt.Printf("Core %d: Running task %d, queue size %d\n", core.id, core.current.id, len(core.runqueue))
		case Stopped:
			fmt.Printf("Core %d: Stopped\n", core.id)
		}
	}
	fmt.Println("------------------------")
	fmt.Println("Tasks waiting to be scheduled:", s.runqueue)
	fmt.Println("------------------------")
}

func (s *Scheduler) LogStats() {
	fmt.Println("------ Task Stats ------")
	totalRuntime := time.Duration(0)
	for _, task := range s.tasks {
		totalRuntime += task.runTime
	}
	averageRuntime := totalRuntime / time.Duration(len(s.tasks))
	fmt.Printf("Average task runtime: %v\n", averageRuntime.Round(time.Millisecond))
	fmt.Println("------------------------")
}

func (s *Scheduler) Shutdown() {
	for _, core := range s.cores {
		core.Shutdown()
	}
	s.LogProgress()
	s.LogStats()
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
		/* timeSlice= */ 100*time.Millisecond,
		/* numCores= */ 2,
	)

	for i := 0; i < 10; i++ {
		scheduler.AddTask(NewTask(i /* work= */, 5000))
		// scheduler.AddTask(NewTask(i, /* work= */ rand.Intn(5000)+1))
	}

	scheduler.Run()
}
