package cron

import (
	"github.com/robfig/cron/v3"
	"magento.GO/config"
	"log"
)

func StartCron() *cron.Cron {
	c := cron.New()
	addJobs := func(jobMap map[string]config.CronJob) {
		for name, cronJob := range jobMap {
			jobFunc := cronJob.Job
			_, err := c.AddFunc(cronJob.Schedule, func() { jobFunc() })
			if err != nil {
				log.Fatalf("Failed to register job %s: %v", name, err)
			}
		}
	}
	addJobs(config.CronJobs)
	for name, j := range Jobs() {
		run := j.Run
		sched := j.Schedule
		_, err := c.AddFunc(sched, func() { run() })
		if err != nil {
			log.Fatalf("Failed to register job %s: %v", name, err)
		}
	}
	c.Start()
	return c
}