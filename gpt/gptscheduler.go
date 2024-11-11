package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

const (
	LogMode      = iota // Logs events when cores switch tasks
	ProgressMode        // Displays relative progress of tasks and core busyness
)

type Task struct {
	id          int
	workLeft    int32 // Use int32 for atomic operations
	initialWork int
	isCompleted bool
	assigned    int32 // Atomic flag to indicate if the task is assigned (1 if assigned, 0 if not)
}

type Scheduler struct {
	tasks       []*Task
	taskQueue   chan *Task
	timeSlice   int
	numCores    int
	outputMode  int
	coreStatus  []string
	wg          sync.WaitGroup
	outputMutex sync.Mutex
	logChannel  chan string
	logWG       sync.WaitGroup
	doneChannel chan struct{}
	dispatchWG  sync.WaitGroup // WaitGroup for dispatching tasks
	progressWG  sync.WaitGroup // WaitGroup for progress checking
	once        sync.Once      // Ensures that logChannel is closed only once
}

func NewScheduler(timeSlice int, numCores int, outputMode int) *Scheduler {
	return &Scheduler{
		timeSlice:   timeSlice,
		numCores:    numCores,
		outputMode:  outputMode,
		taskQueue:   make(chan *Task), // unbuffered channel
		coreStatus:  make([]string, numCores),
		logChannel:  make(chan string, 100), // buffered channel for logs
		doneChannel: make(chan struct{}),
	}
}

func (s *Scheduler) AddTask(id int, workLeft int) {
	task := &Task{id: id, workLeft: int32(workLeft), initialWork: workLeft}
	s.tasks = append(s.tasks, task)
}

func (s *Scheduler) Start() {
	// Start worker goroutines for each core
	for i := 0; i < s.numCores; i++ {
		s.wg.Add(1)
		go s.runCore(i)
	}

	// Start logging goroutine
	s.logWG.Add(1)
	go s.handleLogging()

	// Start task dispatching
	s.dispatchWG.Add(1)
	go s.dispatchTasks()

	// Start progress checking
	if s.outputMode == ProgressMode {
		s.progressWG.Add(1)
		go s.checkProgress()
	}

	// Wait for task dispatch to finish
	s.dispatchWG.Wait()

	// Close task queue after all tasks are dispatched
	close(s.taskQueue)

	// Wait for all cores to finish
	s.wg.Wait()

	// Wait for logging to finish
	s.logWG.Wait()

	// Signal completion
	s.once.Do(func() {
		close(s.logChannel)  // Safely close logChannel
		close(s.doneChannel) // Signal progress checking to stop
	})

	// Final log before exit.
	clearScreen()
	s.printProgress()

	fmt.Println("All tasks completed")
}

func (s *Scheduler) dispatchTasks() {
	defer s.dispatchWG.Done()

	// Keep dispatching tasks until all tasks are dispatched
	for {
		allCompleted := true
		for _, task := range s.tasks {
			// Try to assign the task to a core (if not already assigned)
			if atomic.LoadInt32(&task.assigned) == 0 && atomic.LoadInt32(&task.workLeft) > 0 && !task.isCompleted {
				// Use atomic CompareAndSwap to ensure no other core picks this task
				if atomic.CompareAndSwapInt32(&task.assigned, 0, 1) {
					s.taskQueue <- task
					allCompleted = false
				}
			}
		}

		// Exit if all tasks are completed
		if allCompleted {
			// Ensure logChannel is closed only once
			s.once.Do(func() {
				close(s.logChannel)  // Safely close logChannel
				close(s.doneChannel) // Signal progress checking to stop
			})
			break
		}

		time.Sleep(time.Duration(s.timeSlice) * time.Millisecond)
	}
}

func (s *Scheduler) runCore(coreID int) {
	defer s.wg.Done()

	for task := range s.taskQueue {
		// Check if the task is assigned and still needs work
		if atomic.LoadInt32(&task.assigned) == 1 && atomic.LoadInt32(&task.workLeft) > 0 && !task.isCompleted {
			// Do the work
			workDone := min(int(atomic.LoadInt32(&task.workLeft)), s.timeSlice)
			atomic.AddInt32(&task.workLeft, -int32(workDone))
			if atomic.LoadInt32(&task.workLeft) == 0 {
				// Mark task as completed atomically
				task.isCompleted = true
			}

			// Set core status
			s.coreStatus[coreID] = fmt.Sprintf("Task %d", task.id)

			if s.outputMode == LogMode {
				s.logChannel <- fmt.Sprintf("Core %d processing Task %d... work left: %d\n", coreID+1, task.id, task.workLeft)
			}

			// Task completion logic
			if task.isCompleted {
				atomic.StoreInt32(&task.assigned, 0) // Mark task as unassigned when completed
				if s.outputMode == LogMode {
					s.logChannel <- fmt.Sprintf("Core %d finished Task %d\n", coreID+1, task.id)
				}
			}
		}

		// Simulate the time it takes to do work
		time.Sleep(time.Duration(s.timeSlice) * time.Millisecond)
	}

	s.coreStatus[coreID] = "Shutdown"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func clearScreen() {
	fmt.Print("\033[H\033[2J")
}

func (s *Scheduler) logToOutput(message string) {
	s.outputMutex.Lock()
	defer s.outputMutex.Unlock()
	fmt.Print(message)
}

func (s *Scheduler) checkProgress() {
	defer s.progressWG.Done()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Display progress without holding critical locks
			clearScreen()
			s.printProgress()
		case <-s.doneChannel:
			return
		}
	}
}

func (s *Scheduler) printProgress() {
	fmt.Println("---- Progress Report ----")
	for _, task := range s.tasks {
		progress := 100 * (float64(task.initialWork-int(atomic.LoadInt32(&task.workLeft))) / float64(task.initialWork))
		fmt.Printf("Task %d: %.2f%% complete, work left: %d\n", task.id, progress, task.workLeft)
	}
	for i, status := range s.coreStatus {
		fmt.Printf("Core %d: %s\n", i+1, status)
	}
	fmt.Println("------------------------")
}

func (s *Scheduler) handleLogging() {
	defer s.logWG.Done()

	for message := range s.logChannel {
		s.logToOutput(message)
	}
}

func main() {
	scheduler := NewScheduler(100, 4, ProgressMode)

	// Add tasks
	scheduler.AddTask(1, 5000)
	scheduler.AddTask(2, 3000)
	scheduler.AddTask(3, 2500)
	scheduler.AddTask(4, 4000)
	scheduler.AddTask(5, 3800)

	// Start the scheduler
	scheduler.Start()
}
