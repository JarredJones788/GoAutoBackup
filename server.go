package main

import (
	"cli"
	"fmt"
	"manager"
	"os"
	"runtime"
)

func main() {

	configLocation := "/etc/autobackup/config.json"
	tasksLocation := "/etc/autobackup/tasks.json"

	if len(os.Args) <= 1 {
		startBackup(configLocation, tasksLocation)
	} else {
		cli := cli.CLI{}.Init(configLocation, tasksLocation)
		switch os.Args[1] {
		case "add":
			cli.AddTask()
			break
		case "remove":
			cli.RemoveTask(os.Args[2])
			break
		case "list":
			cli.ListTasks()
			break
		case "status":
			cli.CheckServiceStatus()
			break
		case "stop":
			cli.StopService()
			break
		case "start":
			cli.StartService()
			break
		case "restart":
			cli.RestartService()
			break
		case "disable":
			cli.RemoveService()
			break
		case "init":
			cli.SetupService()
			break
		default:
			fmt.Println("Invalid base command")
			break
		}
	}

}

func startBackup(configLocation string, tasksLocation string) {

	if runtime.GOOS != "linux" {
		fmt.Println("Only works on linux... :(")
	}

	//Start AutoBackup service
	err := manager.Manager{}.Init(configLocation, tasksLocation)
	if err != nil {
		fmt.Println(err.Error())
	}
}
