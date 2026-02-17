# Cron Jobs

Scheduled and on-demand jobs via [robfig/cron](https://github.com/robfig/cron).

## Run Scheduler

```bash
go run cli.go cron:start
```

Background:
```bash
nohup go run cli.go cron:start > cron.log 2>&1 &
```

## Run Single Job

```bash
go run cli.go cron:start --job ProductJsonJob
go run cli.go cron:start --job ProductJsonJob param1 param2
```

Job name is case-insensitive. Extra args passed to job.
