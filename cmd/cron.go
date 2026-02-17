package cmd

import (
	"fmt"
	"os"
	"strings"
	"magento.GO/cron"
	//"magento.GO/cron/jobs"
	"magento.GO/config"
	"github.com/spf13/cobra"
)

var jobName string

var cronStartCmd = &cobra.Command{
	Use:   "cron:start",
	Short: "Start the cron scheduler or run a single job by name",
	Run: func(cmd *cobra.Command, args []string) {
		if jobName != "" {
			name := strings.ToLower(jobName)
			if cronJob, ok := config.CronJobs[name]; ok {
				fmt.Printf("Running cron job: %s\n", jobName)
				cronJob.Job(args...)
				return
			}
			if j, ok := cron.Jobs()[name]; ok {
				fmt.Printf("Running cron job: %s\n", jobName)
				j.Run(args...)
				return
			}
			fmt.Printf("Unknown job: %s\n", jobName)
			os.Exit(1)
		}
		fmt.Println("Starting cron scheduler...")
		c := cron.StartCron()
		defer c.Stop()
		fmt.Println("Cron scheduler started. Press Ctrl+C to exit.")
		select {} // Block forever
	},
}

func init() {
	cronStartCmd.Flags().StringVarP(&jobName, "job", "j", "", "Run a single cron job by name and exit")
	rootCmd.AddCommand(cronStartCmd)
} 