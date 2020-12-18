package cli

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"manager"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"
	"types"
	"utils"

	"github.com/kardianos/osext"
)

//CLI - Command line interface.
type CLI struct {
	Manager manager.Manager
}

//Init - starts cli
func (cli CLI) Init(configLocation string, tasksLocation string) *CLI {

	cli.Manager = manager.Manager{
		ConfigLocation: configLocation,
		TasksLocation:  tasksLocation,
	}

	return &cli
}

//AddTask - Adds new task to the Task Manager
func (cli *CLI) AddTask() {
	task := types.Task{
		ID:       utils.GenerateUUID(),
		Tag:      "NOT SET",
		OpType:   "Folder",
		Created:  time.Now(),
		LastRun:  time.Now(),
		Interval: 999999 * time.Hour,
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Task Name: ")
	tag, _ := reader.ReadString('\n')
	tag = strings.ReplaceAll(tag, "\r", "")
	tag = strings.ReplaceAll(tag, "\n", "")
	task.Tag = tag

	fmt.Print("Task Run Interval(Seconds): ")
	interval, _ := reader.ReadString('\n')
	i, err := strconv.Atoi(strings.TrimSpace(interval))
	if err != nil {
		fmt.Println("Invalid Interval")
		return
	}
	task.Interval = time.Duration(i) * time.Second

	fmt.Print("Task Type (folder, database): ")
	opType, _ := reader.ReadString('\n')
	opType = strings.ReplaceAll(opType, "\r", "")
	opType = strings.ReplaceAll(opType, "\n", "")
	if opType != "database" && opType != "folder" {
		fmt.Println("Invalid Task Type")
		return
	}
	task.OpType = opType

	if opType == "folder" {
		task = cli.getFolderInfo(task)
	} else if opType == "database" {
		task = cli.getDatabaseInfo(task)
	}

	if !cli.Manager.AddTask(task) {
		return
	}

	fmt.Println("Task Added")
}

func (cli *CLI) getFolderInfo(task types.Task) types.Task {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Folder Path: ")
	path, _ := reader.ReadString('\n')
	path = strings.ReplaceAll(path, "\r", "")
	path = strings.ReplaceAll(path, "\n", "")
	if !utils.CheckFolderExists(path) {
		fmt.Println("Path does not exist for task: " + task.Tag)
		os.Exit(0)
	}

	task.Path = path
	return task
}

func (cli *CLI) getDatabaseInfo(task types.Task) types.Task {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Database Name: ")
	dbName, _ := reader.ReadString('\n')
	dbName = strings.ReplaceAll(dbName, "\r", "")
	dbName = strings.ReplaceAll(dbName, "\n", "")

	task.Database = dbName
	return task
}

//RemoveTask - Removes task from the Task Manager
func (cli *CLI) RemoveTask(tag string) {
	if cli.Manager.RemoveTask(tag) {
		fmt.Println("Task Removed")
	} else {
		fmt.Println("Task not found")
	}
}

//ListTasks - Displays all tasks in the manager
func (cli *CLI) ListTasks() {
	// initialize tabwriter
	w := new(tabwriter.Writer)

	// minwidth, tabwidth, padding, padchar, flags
	w.Init(os.Stdout, 8, 8, 0, '\t', 0)

	defer w.Flush()
	fmt.Fprintf(w, "\n %s\t%s\t%s\t%s\t", "Task Name", "Task Type", "Next Run", "Last Run")
	fmt.Fprintf(w, "\n %s\t%s\t%s\t%s\t", "----------", "----------", "----------", "----------")

	tasks := cli.Manager.LoadTasks()
	for _, t := range tasks {
		nextMins := int(t.LastRun.Add(t.Interval).Sub(time.Now()).Minutes())
		lastMins := int(time.Now().Sub(t.LastRun).Minutes())
		fmt.Fprintf(w, "\n %s\t%s\t%s\t%s\t", t.Tag, t.OpType, strconv.Itoa(nextMins)+" minutes", strconv.Itoa(lastMins)+" minutes ago")
	}
	fmt.Fprint(w, "\n", "")
	fmt.Fprint(w, "\n", "")
}

//CheckServiceStatus - Checks if the backup service is running.
func (cli *CLI) CheckServiceStatus() {
	cmd := exec.Command("systemctl", "status", "autobackup")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	t, err := cmd.Output()
	if err != nil {
		if stderr.String() == "" {
			fmt.Println(err)
		} else {
			fmt.Println(stderr.String())
		}
		return
	}
	fmt.Println(string(t))
}

//StopService - Stops backup service
func (cli *CLI) StopService() {
	cmd := exec.Command("systemctl", "stop", "autobackup")
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

	fmt.Println("Service stopped")
}

//StartService - Starts backup service
func (cli *CLI) StartService() {
	cmd := exec.Command("systemctl", "start", "autobackup")
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

	fmt.Println("Service started")
}

//RestartService - Restarts backup service
func (cli *CLI) RestartService() {
	cmd := exec.Command("systemctl", "restart", "autobackup")
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

	fmt.Println("Service restarted")
}

//SetupService - Downloads borgbackup && creates a service for systemd
func (cli *CLI) SetupService() bool {

	if !cli.Manager.CheckBorg() {
		fmt.Println("Starting Download...")
		fmt.Println("Updating apt repo...")
		_, err := exec.Command("apt", "update").Output()
		if err != nil {
			fmt.Println("Failed updating apt repo")
			return false
		}
		fmt.Println("Downloading borgbackup")
		_, err = exec.Command("apt", "-y", "install", "borgbackup").Output()
		if err != nil {
			fmt.Println("Failed installing borgbackup")
			return false
		}
	}

	path, err := osext.ExecutableFolder()
	if err != nil {
		fmt.Println(err)
		return false
	}

	content := "[Unit]\nDescription=AutoBackup\n\n[Service]\nType=simple\nRestart=on-failure\nRestartSec=5s\nUser=root\nExecStart=" + path + "/autobackup\n\n[Install]\nWantedBy=multi-user.target"
	if ioutil.WriteFile("/etc/systemd/system/autobackup.service", []byte(content), 0644) != nil {
		fmt.Println("Error saving service file")
		return false
	}
	cmd := exec.Command("systemctl", "enable", "autobackup.service")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	_, err = cmd.Output()
	if err != nil {
		fmt.Println(stderr.String())
		return false
	}
	cmd = exec.Command("systemctl", "start", "autobackup")
	cmd.Stderr = &stderr
	_, err = cmd.Output()
	if err != nil {
		fmt.Println(stderr.String())
		return false
	}

	fmt.Println("AutoBackup service was created")
	return true
}

//RemoveService - removes systemd service
func (cli *CLI) RemoveService() {
	cmd := exec.Command("systemctl", "stop", "autobackup.service")
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

	cmd = exec.Command("systemctl", "disable", "autobackup.service")
	cmd.Stderr = &stderr
	_, err = cmd.Output()
	if err != nil {
		if stderr.String() == "" {
			fmt.Println(err)
		} else {
			fmt.Println(stderr.String())
		}
		return
	}

	cmd = exec.Command("rm", "/etc/systemd/system/autobackup.service")
	cmd.Stderr = &stderr
	_, err = cmd.Output()
	if err != nil {
		if stderr.String() == "" {
			fmt.Println(err)
		} else {
			fmt.Println(stderr.String())
		}
		return
	}

	fmt.Println("AutoBackup service has been removed")
}
