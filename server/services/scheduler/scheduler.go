// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package scheduler

import (
	"fmt"
	"time"
)

type TaskFunc func()

type ScheduledTask struct {
	Name      string        `json:"name"`
	Interval  time.Duration `json:"interval"`
	Recurring bool          `json:"recurring"`
	function  func()
	cancel    chan struct{}
	canceled  chan struct{}
}

func CreateTask(name string, function TaskFunc, timeToExecution time.Duration) *ScheduledTask {
	return createTask(name, function, timeToExecution, false)
}

func CreateRecurringTask(name string, function TaskFunc, interval time.Duration) *ScheduledTask {
	return createTask(name, function, interval, true)
}

func createTask(name string, function TaskFunc, interval time.Duration, recurring bool) *ScheduledTask {
	task := &ScheduledTask{
		Name:      name,
		Interval:  interval,
		Recurring: recurring,
		function:  function,
		cancel:    make(chan struct{}),
		canceled:  make(chan struct{}),
	}

	go func() {
		defer close(task.canceled)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				function()
			case <-task.cancel:
				return
			}

			if !task.Recurring {
				break
			}
		}
	}()

	return task
}

func (task *ScheduledTask) Cancel() {
	close(task.cancel)
	<-task.canceled
}

func (task *ScheduledTask) String() string {
	return fmt.Sprintf(
		"%s\nInterval: %s\nRecurring: %t\n",
		task.Name,
		task.Interval.String(),
		task.Recurring,
	)
}
