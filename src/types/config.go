package types

import "time"

//Config - type
type Config struct {
	BorgHost       string
	Repo           string
	LocalBackups   bool
	CheckInterval  int32
	PruneInterval  int32
	BackupDuration string
	MySQLUser      string
	MySQLPass      string
}

//Task - type
type Task struct {
	ID       string        `json:"id"`
	Tag      string        `json:"tag"`
	OpType   string        `json:"opType"`
	Path     string        `json:"path"`
	Database string        `json:"database"`
	Created  time.Time     `json:"created"`
	LastRun  time.Time     `json:"lastRun"`
	Interval time.Duration `json:"interval"`
}
