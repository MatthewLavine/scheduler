# GPT CPU Scheduler

Playing around comparing a ChatGPT generated CPU scheduler versus a manually coded version.

Output looks like this:

```
---- Task Progress ----
Task 0: 100% complete, runtime 9.35s
Task 1: 100% complete, runtime 4.192s
Task 2: 100% complete, runtime 275ms
Task 3: 100% complete, runtime 9.757s
Task 4: 100% complete, runtime 12.399s
Task 5: 100% complete, runtime 1.208s
Task 6: 100% complete, runtime 14.992s
Task 7: 100% complete, runtime 14.19s
Task 8: 100% complete, runtime 13.497s
Task 9: 100% complete, runtime 8.128s
---- Core Progress ----
Core 0: Stopped
Core 1: Stopped
---- Scheduler Stats ----
Tasks waiting to be scheduled: 0
------------------------
------ Task Stats ------
Average task runtime: 8.799s
------------------------
```

You can use this to demonstrate the responsiveness difference between a preemptive scheduler and a (non)cooperative scheduler:

Preemptive:

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

Cooperative (using a `999m` timeSlice to emulate non-preemption):

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
