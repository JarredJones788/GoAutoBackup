package manager

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"
	"types"
	"utils"
)

//Manager - manages backup logic
type Manager struct {
	ConfigLocation string
	TasksLocation  string
	config         types.Config
}

//Init - starts the auto backup service
func (m Manager) Init(configLocation string, TasksLocation string) error {

	m.ConfigLocation = configLocation
	m.TasksLocation = TasksLocation

	//If a config file does not exists. Create a default one.
	if !utils.FileExists(configLocation) {
		if utils.CreateConfigFile(configLocation) {
			fmt.Println("Default config created. You can find it here: " + configLocation)
		} else {
			fmt.Println("Failed creating default configuration file..")
			return nil
		}
	}

	configFile, err := ioutil.ReadFile(configLocation)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	var config types.Config

	//Cast config into types.Config struct
	err = json.Unmarshal(configFile, &config)
	if err != nil {
		fmt.Println("Failed reading config file")
		fmt.Println(err)
		return nil
	}

	m.config = config

	if !m.CheckBorg() {
		return nil
	}

	go utils.Schedule(m.prune, time.Duration(config.PruneInterval)*time.Second)

	fmt.Println("autobackup service started")
	utils.Schedule(m.check, time.Duration(config.CheckInterval)*time.Second)

	return nil
}

//CheckBorg - Checks if borgbackup is installed
func (m *Manager) CheckBorg() bool {
	cmd := exec.Command("borg")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	_, err := cmd.Output()
	if err != nil {
		if stderr.String() == "" {
			fmt.Println(err)
		} else {
			fmt.Println(stderr.String())
		}
		return false
	}

	//Determine if local or remote REPO
	repo := m.config.BorgHost + ":" + m.config.Repo
	if m.config.LocalBackups {
		repo = m.config.Repo
	}

	//Make sure the repo is already created. If not create it.
	cmd = exec.Command("borg", "init", "-e", "none", repo)
	cmd.Stderr = &stderr
	_, err = cmd.Output()
	if err == nil {
		if stderr.String() == "" {
			fmt.Println(err)
		} else {
			fmt.Println(stderr.String())
		}
		return false
	}

	return true
}

func (m *Manager) check() {

	Tasks := m.LoadTasks()
	for i, task := range Tasks {

		if time.Now().Sub(task.LastRun) >= task.Interval {
			Tasks[i].LastRun = time.Now()
			switch task.OpType {
			case "folder":
				if m.backUpFolder(task) {
					fmt.Println("Saved backup for: " + task.Tag)
				}
				break
			case "database":
				if m.backUpMySQLDatabase(task) {
					fmt.Println("Saved backup for: " + task.Tag)
				}
				break
			default:
				fmt.Println("Invalid Task Type: " + task.Tag)
				break
			}
		}
	}

	m.save(Tasks)

	fmt.Println("Finished Checking...")
}

func (m *Manager) save(Tasks []types.Task) bool {
	data, err := json.Marshal(Tasks)
	if err != nil {
		fmt.Println("Error saving Tasks config")
		return false
	}
	if ioutil.WriteFile("/etc/autobackup/Tasks.json", data, 0644) != nil {
		fmt.Println("Error saving Tasks config")
		return false
	}
	return true
}

func (m *Manager) prune() {
	//Determine if local or remote REPO
	repo := m.config.BorgHost + ":" + m.config.Repo
	if m.config.LocalBackups {
		repo = m.config.Repo
	}
	cmd := exec.Command("borg", "prune", "-v", "--keep-within="+m.config.BackupDuration, repo)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	_, err := cmd.Output()
	if err != nil {
		if stderr.String() == "" {
			fmt.Println(err)
		} else {
			fmt.Println(stderr.String())
		}
		return
	}

	fmt.Println("Pruned Repo")
}

func (m *Manager) backUpFolder(task types.Task) bool {
	if !utils.CheckFolderExists(task.Path) {
		fmt.Println("Path does not exist for task: " + task.Tag)
		return false
	}
	//Determine if local or remote REPO
	repo := m.config.BorgHost + ":" + m.config.Repo
	if m.config.LocalBackups {
		repo = m.config.Repo
	}
	cmd := exec.Command("borg", "create", repo+"::"+task.Tag+"-"+task.OpType+"-"+strings.ReplaceAll(time.Now().Format("2006-01-02 15:04:05"), " ", "-"), task.Path)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	_, err := cmd.Output()
	if err != nil {
		if stderr.String() == "" {
			fmt.Println(err)
		} else {
			fmt.Println(stderr.String())
		}
		return false
	}
	return true
}

func (m *Manager) backUpMySQLDatabase(task types.Task) bool {
	tmpID := utils.GenerateUUID()
	task.Path = "/tmp/" + tmpID
	os.MkdirAll(task.Path, os.ModePerm)
	cmd := exec.Command("mysqldump", "--user="+m.config.MySQLUser, "--password="+m.config.MySQLPass, "--result-file="+task.Path+"/data.sql", task.Database)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	_, err := cmd.Output()
	if err != nil {
		if stderr.String() == "" {
			fmt.Println(err)
		} else {
			fmt.Println(stderr.String())
		}
		return false
	}

	if m.backUpFolder(task) {
		os.RemoveAll(task.Path)
		return true
	}

	os.RemoveAll(task.Path)
	return false
}

//RemoveTask - removes a task
func (m *Manager) RemoveTask(tag string) bool {

	Tasks := m.LoadTasks()
	for i, t := range Tasks {
		if t.Tag == tag {
			Tasks = append(Tasks[:i], Tasks[i+1:]...)
			m.save(Tasks)
			return true
		}
	}

	return false
}

//AddTask - creates a new task
func (m *Manager) AddTask(task types.Task) bool {

	Tasks := m.LoadTasks()
	for _, t := range Tasks {
		if task.Tag == t.Tag {
			fmt.Println("Task already exists")
			return false
		}
	}

	t := append(Tasks, task)

	if !m.save(t) {
		return false
	}

	return true
}

//LoadTasks - fetches live tasks
func (m *Manager) LoadTasks() []types.Task {

	//If a tasks file does not exists. Create a default one.
	if !utils.FileExists(m.TasksLocation) {
		if utils.CreateTasksFile(m.TasksLocation) {
			fmt.Println("Default tasks created. You can find it here: " + m.TasksLocation)
		} else {
			fmt.Println("Failed creating default tasks file..")
			return nil
		}
	}

	//Read config file
	tasksFile, err := ioutil.ReadFile(m.TasksLocation)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	var tasks []types.Task

	//Cast config into types.Config struct
	err = json.Unmarshal(tasksFile, &tasks)
	if err != nil {
		fmt.Println("Failed reading tasks file")
		fmt.Println(err)
		return nil
	}
	return tasks
}
