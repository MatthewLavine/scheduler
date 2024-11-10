package main

import (
	"fmt"
	"math"
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
	workFunc    func(int) int
	isCompleted bool
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

func (s *Scheduler) AddTask(id int, workLeft int, workFunc func(int) int) {
	task := &Task{id: id, workLeft: int32(workLeft), initialWork: workLeft, workFunc: workFunc}
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
			if atomic.LoadInt32(&task.workLeft) > 0 && !task.isCompleted {
				// Send task to taskQueue if not completed
				s.taskQueue <- task
				allCompleted = false
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
		// Check if the task still has work
		if atomic.LoadInt32(&task.workLeft) > 0 && !task.isCompleted {
			workDone := min(int(atomic.LoadInt32(&task.workLeft)), s.timeSlice)
			atomic.AddInt32(&task.workLeft, -int32(workDone))
			if atomic.LoadInt32(&task.workLeft) == 0 {
				task.isCompleted = true
			}

			// Set core status without blocking task progress
			s.coreStatus[coreID] = fmt.Sprintf("Task %d", task.id)

			if s.outputMode == LogMode {
				s.logChannel <- fmt.Sprintf("Core %d processing Task %d... work left: %d\n", coreID+1, task.id, task.workLeft)
			}
		} else {
			// Mark the core as idle if the task is completed
			s.coreStatus[coreID] = "Idle"
			if s.outputMode == LogMode {
				s.logChannel <- fmt.Sprintf("Core %d transitioning to idle", coreID+1)
			}
		}

		// Simulate the time it takes to do work
		time.Sleep(time.Duration(s.timeSlice) * time.Millisecond)

		// Log completion of the task
		if task.isCompleted && s.outputMode == LogMode {
			s.logChannel <- fmt.Sprintf("Core %d finished Task %d\n", coreID+1, task.id)
		}
	}
	s.coreStatus[coreID] = "Idle"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func SimulateHeavyComputation(units int) int {
	sum := 0
	for i := 0; i < units; i++ {
		sum += int(math.Sqrt(float64(i)))
	}
	return units - 50
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
	scheduler := NewScheduler(500, 4, ProgressMode)

	// Add tasks
	scheduler.AddTask(1, 5000, SimulateHeavyComputation)
	scheduler.AddTask(2, 3000, SimulateHeavyComputation)
	scheduler.AddTask(3, 2500, SimulateHeavyComputation)
	scheduler.AddTask(4, 4000, SimulateHeavyComputation)
	scheduler.AddTask(5, 3800, SimulateHeavyComputation)

	// Start the scheduler
	scheduler.Start()
}
