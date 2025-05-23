package config

import (
	"magento.GO/cron/jobs"
)

// Map of job names to job functions
type CronJob struct {
	Schedule string
	Job      func(...string)
}

var CronJobs = map[string]CronJob{
	"productjsonjob": {Schedule: "0 * * * *", Job: jobs.ProductJsonJob},
	"testjob":        {Schedule: "@every 10s", Job: jobs.TestJob},
	// Add more jobs here
}
