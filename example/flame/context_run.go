package flame

import (
	"sync"
)

/*
There are 3 running mode:
1. as normal program
	If not in distributed mode, it should not be intercepted.
2. "-driver" mode to drive in distributed mode
	context runner will register
3. "-task.[context|step|task].id" mode to run task in distributed mode
*/

var contextRunner ContextRunner
var taskRunner TaskRunner

// Invoked by driver task runner
func RegisterContextRunner(r ContextRunner) {
	contextRunner = r
}
func RegisterTaskRunner(r TaskRunner) {
	taskRunner = r
}

type ContextRunner interface {
	Run(fc *FlowContext)
	ShouldRun(fc *FlowContext) bool
	IsDriverMode() bool
}

type TaskRunner interface {
	Run(fc *FlowContext, step *Step, task *Task)
	ShouldRun(fc *FlowContext, step *Step, task *Task) bool
	IsTaskMode() bool
}

func (fc *FlowContext) Run() {

	if taskRunner.IsTaskMode() {
		fc.run_taskrunner(taskRunner)
	} else if contextRunner.IsDriverMode() {
		fc.run_driver(contextRunner)
	} else {
		fc.run_standalone()
	}
}

func (fc *FlowContext) run_driver(contextRunner ContextRunner) {
	if contextRunner.ShouldRun(fc) {
		contextRunner.Run(fc)
	}
}

func (fc *FlowContext) run_taskrunner(tr TaskRunner) {
	var wg sync.WaitGroup
	// start all task edges
	for _, step := range fc.Steps {
		for _, t := range step.Tasks {
			if tr.ShouldRun(fc, step, t) {
				wg.Add(1)
				go func(t *Task) {
					defer wg.Done()
					tr.Run(fc, step, t)
				}(t)
			}
		}
	}
	wg.Wait()
}

func (fc *FlowContext) run_standalone() {

	var wg sync.WaitGroup

	// start all task edges
	for i, step := range fc.Steps {
		if i == 0 {
			wg.Add(1)
			go func(step *Step) {
				defer wg.Done()
				// println("start dataset", step.Id)
				if step.Input != nil {
					step.Input.RunSelf(step.Id)
				}
			}(step)
		}
		wg.Add(1)
		go func(step *Step) {
			defer wg.Done()
			step.Run()
		}(step)
		wg.Add(1)
		go func(step *Step) {
			defer wg.Done()
			// println("start dataset", step.Id+1)
			if step.Output != nil {
				step.Output.RunSelf(step.Id + 1)
			}
		}(step)
	}
	wg.Wait()
}
