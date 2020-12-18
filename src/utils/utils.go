package utils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"
	"types"

	"github.com/google/uuid"
)

//FileExists - checks if current file exists
func FileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

//CheckFolderExists - check if path exists
func CheckFolderExists(path string) bool {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

//GenerateUUID - creates a UUID
func GenerateUUID() string {
	return uuid.New().String()
}

//CreateConfigFile - creates a default config file
func CreateConfigFile(configLocation string) bool {

	config := types.Config{
		LocalBackups:   true,
		BorgHost:       "",
		Repo:           "/BorgBackups",
		BackupDuration: "30D",
		CheckInterval:  60,
		PruneInterval:  3600,
		MySQLUser:      "root",
		MySQLPass:      "",
	}

	//Make default config a json object
	data, err := json.MarshalIndent(config, "", " ")
	if err != nil {
		fmt.Println(err)
		return false
	}

	//Create all parent folder if they do not exist.
	pathLocation := strings.Replace(configLocation, "/config.json", "", 1)
	err = os.MkdirAll(pathLocation, os.ModePerm)
	if err != nil {
		fmt.Println(err)
		return false
	}

	//Create the tasks file.
	err = ioutil.WriteFile(configLocation, data, 0644)
	if err != nil {
		fmt.Println(err)
		return false
	}

	return true
}

//CreateTasksFile - creates a default tasks file
func CreateTasksFile(tasksLocation string) bool {

	//Create all parent folder if they do not exist.
	pathLocation := strings.Replace(tasksLocation, "/tasks.json", "", 1)
	err := os.MkdirAll(pathLocation, os.ModePerm)
	if err != nil {
		fmt.Println(err)
		return false
	}

	//Create the tasks file.
	err = ioutil.WriteFile(tasksLocation, []byte("[]"), 0644)
	if err != nil {
		fmt.Println(err)
		return false
	}

	return true
}

//Schedule - creates a interval function
func Schedule(what func(), delay time.Duration) {
	stop := make(chan bool)

	func() {
		for {
			select {
			case <-time.After(delay):
				what()
			case <-stop:
				return
			}
		}
	}()

}
