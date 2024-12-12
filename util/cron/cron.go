package cron

import (
	"context"
	"github.com/robfig/cron"
	"mtt-indexer/logger"
)

type Handler func(ctx context.Context) error

type ErrHandler func(name string, err error)

func DefaultErrHandler(name string, err error) {
	if err != nil {
		logger.Logger.Error("run task: %s err: %v", name, err)
	}
}

type task struct {
	name       string
	spec       string
	handler    Handler
	errHandler ErrHandler
	cron       *cron.Cron
}

type Cron struct {
	tasks []task
}

func NewCron() *Cron {
	return &Cron{
		tasks: make([]task, 0),
	}
}

// Register 可以一次性注册多个任务
func (c *Cron) Register(name, spec string, handler Handler, errHandlers ...ErrHandler) {
	// 默认的error处理
	errHandler := DefaultErrHandler
	if len(errHandlers) > 0 {
		errHandler = errHandlers[0]
	}
	job := cron.New()
	err := job.AddFunc(spec, func() {
		// 每次执行时打印log
		logger.Logger.Info("[cron] run task: %s", name)
		err := handler(context.Background())
		if err != nil {
			errHandler(name, err)
		}
		logger.Logger.Info("[cron] run task end: %s", name)
	})
	if err != nil {
		logger.Logger.Fatal("[cron] job.AddFunc err, name: %s, err: %v", name, err)
	}
	c.tasks = append(c.tasks, task{
		name:       name,
		spec:       spec,
		handler:    handler,
		errHandler: errHandler,
		cron:       job,
	})
}

// Run 运行任务
func (c *Cron) Run() {
	for i := range c.tasks {
		c.tasks[i].cron.Run()
	}
}

// Stop 停止执行
func (c *Cron) Stop() {
	for i := range c.tasks {
		c.tasks[i].cron.Stop()
	}
}
