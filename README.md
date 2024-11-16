# (GPT) CPU Scheduler

Playing around comparing a ChatGPT generated CPU scheduler versus a manually coded version.

---

Output looks like this:

```
$ go run scheduler.go --cores=4 --timeSlice=10ms --cheapTasksCount=10 --expensiveTasksCount=10
---- Task Progress ----
Task  1: 100.00% complete, runtime 339ms, longest wait time: 92ms
Task  2: 100.00% complete, runtime 145ms, longest wait time: 92ms
Task  3: 100.00% complete, runtime 432ms, longest wait time: 92ms
Task  4: 100.00% complete, runtime 515ms, longest wait time: 92ms
Task  5: 100.00% complete, runtime 83ms, longest wait time: 61ms
Task  6: 100.00% complete, runtime 457ms, longest wait time: 82ms
Task  7: 100.00% complete, runtime 495ms, longest wait time: 72ms
Task  8: 100.00% complete, runtime 98ms, longest wait time: 61ms
Task  9: 100.00% complete, runtime 205ms, longest wait time: 103ms
Task 10: 100.00% complete, runtime 524ms, longest wait time: 102ms
Task 11: 100.00% complete, runtime 4.228s, longest wait time: 103ms
Task 12: 100.00% complete, runtime 24.57s, longest wait time: 71ms
Task 13: 100.00% complete, runtime 11.943s, longest wait time: 81ms
Task 14: 100.00% complete, runtime 18.401s, longest wait time: 82ms
Task 15: 100.00% complete, runtime 17.293s, longest wait time: 92ms
Task 16: 100.00% complete, runtime 25.508s, longest wait time: 92ms
Task 17: 100.00% complete, runtime 22.701s, longest wait time: 92ms
Task 18: 100.00% complete, runtime 11.807s, longest wait time: 92ms
Task 19: 100.00% complete, runtime 23.812s, longest wait time: 82ms
Task 20: 100.00% complete, runtime 22.21s, longest wait time: 72ms
---- Core Progress ----
Core 0: Stopped
Core 1: Stopped
Core 2: Stopped
Core 3: Stopped
---- Scheduler Stats ----
Tasks waiting to be scheduled: 0
------ Task Stats ------
Min task runtime: 83ms
Avg task runtime: 9.288s
Max task runtime: 25.508s
Min task wait time: 61ms
Avg task wait time: 85ms
Max task wait time: 103ms
------------------------
```

---

You can use this to demonstrate the responsiveness difference between a preemptive scheduler and a (non)cooperative scheduler:

#### Preemptive:

```
$ go run scheduler.go --cores=6 --timeSlice=10ms --cheapTasksCount=0 --expensiveTasksCount=20
...
------ Task Stats ------
Min task runtime: 818ms
Avg task runtime: 21.864s
Max task runtime: 31.566s
Min task wait time: 72ms
Avg task wait time: 83ms
Max task wait time: 103ms
------------------------
```

#### Cooperative (using a `999m` timeSlice to emulate non-preemption):

```
$ go run scheduler.go --cores=6 --timeSlice=999m --cheapTasksCount=0 --expensiveTasksCount=20
------ Task Stats ------
Min task runtime: 178ms
Avg task runtime: 13.05s
Max task runtime: 28.427s
Min task wait time: 0s
Avg task wait time: 6.726s
Max task wait time: 20.542s
------------------------
```

In the preemptive scheduler, max task wait time is proportial to the timeSlice, in the non-preemptive scheduler the wait time is proportional to the largest task / process.
