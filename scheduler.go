package main

import (
	"fmt"
	"math"
	"sync"
	"time"
)

const (
	LogMode      = iota // Logs events when cores switch tasks
	ProgressMode        // Displays relative progress of tasks and core busyness
)

type Task struct {
	id          int
	workLeft    int
	initialWork int
	workFunc    func(int) int
	isCompleted bool
	mu          sync.Mutex
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
	task := &Task{id: id, workLeft: workLeft, initialWork: workLeft, workFunc: workFunc}
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
	s.progressWG.Add(1)
	go s.checkProgress()

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
	fmt.Println("All tasks completed")
}

func (s *Scheduler) dispatchTasks() {
	defer s.dispatchWG.Done()

	// Keep dispatching tasks until all tasks are dispatched
	for {
		allCompleted := true
		for _, task := range s.tasks {
			task.mu.Lock()
			if !task.isCompleted && task.workLeft > 0 {
				// Send task to taskQueue if not completed
				s.taskQueue <- task
				allCompleted = false
			}
			task.mu.Unlock()
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
		task.mu.Lock()
		if task.workLeft > 0 && !task.isCompleted {
			workDone := min(task.workLeft, s.timeSlice)
			task.workLeft -= workDone
			if task.workLeft == 0 {
				task.isCompleted = true
			}

			// Set core status without blocking task progress
			s.coreStatus[coreID] = fmt.Sprintf("Core %d: Task %d", coreID+1, task.id)

			if s.outputMode == LogMode {
				s.logChannel <- fmt.Sprintf("Core %d processing Task %d... work left: %d\n", coreID+1, task.id, task.workLeft)
			}
		} else {
			// Mark the core as idle if the task is completed
			s.coreStatus[coreID] = fmt.Sprintf("Core %d: Idle", coreID+1)
		}

		// Simulate the time it takes to do work
		time.Sleep(time.Duration(s.timeSlice) * time.Millisecond)

		// Log completion of the task
		if task.isCompleted && s.outputMode == LogMode {
			s.logChannel <- fmt.Sprintf("Core %d finished Task %d\n", coreID+1, task.id)
		}

		task.mu.Unlock()
	}
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

	ticker := time.NewTicker(1 * time.Second)
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
		task.mu.Lock()
		progress := 100 * (float64(task.initialWork-task.workLeft) / float64(task.initialWork))
		fmt.Printf("Task %d: %.2f%% complete, work left: %d\n", task.id, progress, task.workLeft)
		task.mu.Unlock()
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
	scheduler.AddTask(1, 500, SimulateHeavyComputation)
	scheduler.AddTask(2, 1000, SimulateHeavyComputation)
	scheduler.AddTask(3, 1500, SimulateHeavyComputation)
	scheduler.AddTask(4, 1000, SimulateHeavyComputation)
	scheduler.AddTask(5, 1800, SimulateHeavyComputation)

	// Start the scheduler
	scheduler.Start()
}
