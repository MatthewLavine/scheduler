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
	id       int
	workLeft int           // Units of work left for this task
	workFunc func(int) int // Function to perform work, returns remaining work
}

// Scheduler simulates a CPU scheduler with multiple cores
type Scheduler struct {
	tasks      []*Task
	taskQueue  chan *Task
	timeSlice  int            // Time slice for each task in ms
	numCores   int            // Number of logical CPU cores
	outputMode int            // Mode of output: LogMode or ProgressMode
	coreStatus []string       // Status of each core (busy or idle)
	wg         sync.WaitGroup // Wait group to wait for all cores to finish
}

// NewScheduler creates a new scheduler with the given time slice, number of cores, and output mode
func NewScheduler(timeSlice int, numCores int, outputMode int) *Scheduler {
	return &Scheduler{
		timeSlice:  timeSlice,
		numCores:   numCores,
		outputMode: outputMode,
		taskQueue:  make(chan *Task, numCores), // Buffered to avoid blocking
		coreStatus: make([]string, numCores),
	}
}

// AddTask adds a new task to the scheduler
func (s *Scheduler) AddTask(id int, workLeft int, workFunc func(int) int) {
	task := &Task{id: id, workLeft: workLeft, workFunc: workFunc}
	s.tasks = append(s.tasks, task)
}

// Start begins the scheduling simulation, creating worker cores
func (s *Scheduler) Start() {
	// Start worker goroutines for each core
	for i := 0; i < s.numCores; i++ {
		s.wg.Add(1)
		go s.runCore(i) // Start each core as a goroutine
	}

	// Dispatch tasks to the queue
	go s.dispatchTasks()

	// Wait for all cores to finish processing tasks
	s.wg.Wait()
	fmt.Println("All tasks completed")
}

// dispatchTasks distributes tasks to the taskQueue until all tasks are processed
func (s *Scheduler) dispatchTasks() {
	// Dispatch tasks to the taskQueue
	for len(s.tasks) > 0 {
		// Only dispatch tasks that still have work left
		for i := 0; i < len(s.tasks); i++ {
			task := s.tasks[i]
			if task.workLeft > 0 {
				// Send the task to the task queue for cores to work on
				s.taskQueue <- task
			}
			// If task is completed, remove it from the list
			if task.workLeft <= 0 {
				fmt.Printf("Task %d finished\n", task.id)
				s.tasks = append(s.tasks[:i], s.tasks[i+1:]...)
				i-- // Adjust index after removal
			}
		}

		// In Progress Mode, print the progress display
		if s.outputMode == ProgressMode {
			s.printProgress()
		}

		time.Sleep(time.Duration(s.timeSlice) * time.Millisecond) // Simulate scheduler tick
	}

	// Close the task queue once all tasks have been dispatched
	close(s.taskQueue)
}

// runCore represents a CPU core that processes tasks
func (s *Scheduler) runCore(coreID int) {
	defer s.wg.Done()
	for task := range s.taskQueue {
		// Process the task
		if task.workLeft > 0 {
			// Calculate the amount of work done in this time slice
			workDone := min(task.workLeft, s.timeSlice)
			task.workLeft -= workDone
			task.workLeft = max(task.workLeft, 0) // Ensure no negative workLeft

			// Update core status for Progress Mode
			s.coreStatus[coreID] = fmt.Sprintf("Core %d: Task %d", coreID+1, task.id)

			if s.outputMode == LogMode {
				fmt.Printf("Core %d started processing Task %d... work left: %d\n", coreID+1, task.id, task.workLeft)
			}
		} else {
			s.coreStatus[coreID] = fmt.Sprintf("Core %d: Idle", coreID+1)
		}

		// Simulate the core's time slice duration without blocking other cores
		time.Sleep(time.Duration(s.timeSlice) * time.Millisecond)

		s.coreStatus[coreID] = fmt.Sprintf("Core %d: Idle", coreID+1) // Mark core as idle when done
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

// printProgress displays the current status of all tasks and cores
func (s *Scheduler) printProgress() {
	fmt.Println("---- Progress Report ----")
	for _, task := range s.tasks {
		progress := 100 * (1 - float64(task.workLeft)/float64(task.workFunc(0))) // Show progress as a percentage
		fmt.Printf("Task %d: %.2f%% complete, work left: %d\n", task.id, progress, task.workLeft)
	}
	for _, status := range s.coreStatus {
		fmt.Println(status)
	}
	fmt.Println("------------------------")
}

func main() {
	numCores := 4              // Number of logical CPU cores
	timeSlice := 100           // 100ms time slice
	outputMode := ProgressMode // Change to LogMode or ProgressMode

	scheduler := NewScheduler(timeSlice, numCores, outputMode)

	// Add tasks with heavy computation functions
	scheduler.AddTask(1, 1500, SimulateHeavyComputation)
	scheduler.AddTask(2, 2500, SimulateHeavyComputation)
	scheduler.AddTask(3, 1000, SimulateHeavyComputation)
	scheduler.AddTask(4, 3000, SimulateHeavyComputation)
	scheduler.AddTask(5, 2000, SimulateHeavyComputation)

	// Start the scheduler
	scheduler.Start()
}
