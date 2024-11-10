package main

import (
	"fmt"
	"math"
	"sync"
	"time"
)

// Output modes for the scheduler
const (
	LogMode      = iota // Logs events when cores switch tasks
	ProgressMode        // Displays relative progress of tasks and core busyness
)

// Task represents a simulated process or task
type Task struct {
	id          int           // Task ID
	workLeft    int           // Units of work left for this task
	initialWork int           // Initial units of work for this task
	workFunc    func(int) int // Function to perform work, returns remaining work
	isCompleted bool          // Flag to check if task is completed
	mu          sync.Mutex    // Mutex to protect access to task data
}

// Scheduler simulates a CPU scheduler with multiple cores
type Scheduler struct {
	tasks       []*Task
	taskQueue   chan *Task
	timeSlice   int            // Time slice for each task in ms
	numCores    int            // Number of logical CPU cores
	outputMode  int            // Mode of output: LogMode or ProgressMode
	coreStatus  []string       // Status of each core (busy or idle)
	wg          sync.WaitGroup // Wait group to wait for all cores to finish
	outputMutex sync.Mutex     // Mutex to ensure synchronized logging
	logChannel  chan string    // Channel for log messages
	logWG       sync.WaitGroup // Wait group to ensure logging is complete
	doneChannel chan struct{}  // Channel to signal completion of all tasks and logging
}

// NewScheduler creates a new scheduler with the given time slice, number of cores, and output mode
func NewScheduler(timeSlice int, numCores int, outputMode int) *Scheduler {
	return &Scheduler{
		timeSlice:   timeSlice,
		numCores:    numCores,
		outputMode:  outputMode,
		taskQueue:   make(chan *Task, numCores), // Buffered to avoid blocking
		coreStatus:  make([]string, numCores),
		logChannel:  make(chan string, 100), // Channel for logs, buffered
		doneChannel: make(chan struct{}),    // Channel to signal when everything is done
	}
}

// AddTask adds a new task to the scheduler
func (s *Scheduler) AddTask(id int, workLeft int, workFunc func(int) int) {
	task := &Task{id: id, workLeft: workLeft, initialWork: workLeft, workFunc: workFunc}
	s.tasks = append(s.tasks, task)
}

// Start begins the scheduling simulation, creating worker cores
func (s *Scheduler) Start() {
	// Start worker goroutines for each core
	for i := 0; i < s.numCores; i++ {
		s.wg.Add(1)
		go s.runCore(i) // Start each core as a goroutine
	}

	// Start a goroutine to handle logging
	s.logWG.Add(1) // Add 1 to wait for the logging goroutine to finish
	go s.handleLogging()

	// Dispatch tasks to the task queue
	go s.dispatchTasks()

	// Wait for all cores to finish processing tasks
	s.wg.Wait()

	// Close taskQueue after all tasks are dispatched
	close(s.taskQueue)

	// Wait for the logging goroutine to finish
	s.logWG.Wait()

	// Indicate that all tasks have been completed
	fmt.Println("All tasks completed")
	close(s.doneChannel) // Signal completion
}

// dispatchTasks distributes tasks to the taskQueue until all tasks are processed
func (s *Scheduler) dispatchTasks() {
	// Dispatch tasks to the taskQueue
	for {
		// Check if any tasks are pending
		allCompleted := true
		for _, task := range s.tasks {
			task.mu.Lock()
			if !task.isCompleted {
				allCompleted = false
				// Only dispatch tasks that still have work left
				if task.workLeft > 0 {
					s.taskQueue <- task
				}
			}
			task.mu.Unlock()
		}

		// Exit if all tasks are completed
		if allCompleted {
			break
		}

		// In Progress Mode, print the progress display at reduced frequency
		if s.outputMode == ProgressMode {
			s.checkProgress()
		}

		time.Sleep(time.Duration(s.timeSlice) * time.Millisecond) // Simulate scheduler tick
	}
}

// runCore represents a CPU core that processes tasks
func (s *Scheduler) runCore(coreID int) {
	defer s.wg.Done() // Signal when the core is done

	for task := range s.taskQueue {
		// Process the task
		task.mu.Lock()
		if task.workLeft > 0 && !task.isCompleted {
			// Calculate the amount of work done in this time slice
			workDone := min(task.workLeft, s.timeSlice)
			task.workLeft -= workDone
			task.workLeft = max(task.workLeft, 0) // Ensure no negative workLeft

			// Mark task as completed if no work is left
			if task.workLeft == 0 {
				task.isCompleted = true
			}

			// Update core status for Progress Mode
			s.coreStatus[coreID] = fmt.Sprintf("Core %d: Task %d", coreID+1, task.id)

			// Log task progress or completion in LogMode
			if s.outputMode == LogMode {
				s.logChannel <- fmt.Sprintf("Core %d started processing Task %d... work left: %d\n", coreID+1, task.id, task.workLeft)
			}
		} else {
			s.coreStatus[coreID] = fmt.Sprintf("Core %d: Idle", coreID+1)
		}

		// Simulate the core's time slice duration without blocking other cores
		time.Sleep(time.Duration(s.timeSlice) * time.Millisecond)

		// Log task completion in LogMode
		if task.isCompleted && s.outputMode == LogMode {
			s.logChannel <- fmt.Sprintf("Core %d finished processing Task %d\n", coreID+1, task.id)
		}

		// Mark core as idle when done
		s.coreStatus[coreID] = fmt.Sprintf("Core %d: Idle", coreID+1)
		task.mu.Unlock()
	}
}

// Utility functions to get the minimum and maximum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// SimulateHeavyComputation performs a CPU-intensive calculation
func SimulateHeavyComputation(units int) int {
	sum := 0
	for i := 0; i < units; i++ {
		sum += int(math.Sqrt(float64(i))) // Heavy computation
	}
	return units - 50 // Reduce work units by 50 for each call
}

// clearScreen clears the terminal screen
func clearScreen() {
	fmt.Print("\033[H\033[2J") // Escape sequence to clear the screen
}

// logToOutput safely writes to the terminal with synchronized access
func (s *Scheduler) logToOutput(message string) {
	s.outputMutex.Lock() // Ensure only one goroutine writes at a time
	defer s.outputMutex.Unlock()
	fmt.Print(message) // Write log to terminal
}

// checkProgress checks if enough time has passed to print a progress report
func (s *Scheduler) checkProgress() {
	// Reduce logging frequency by checking every second
	select {
	case <-time.After(1 * time.Second):
		clearScreen()
		s.printProgress()
	default:
		// Continue without blocking other tasks
	}
}

// printProgress displays the current status of all tasks and cores
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

// handleLogging processes log messages asynchronously
func (s *Scheduler) handleLogging() {
	defer s.logWG.Done()

	for message := range s.logChannel {
		s.logToOutput(message) // Safely log the message
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
