package main

import (
	"encoding/json"
	"fmt"
	"github.com/yunfeiyang1916/cloud-docker/container"
	"io/ioutil"
)

const (
	// EnvExecPID pid环境变量
	EnvExecPID = "cloud_docker_pid"
	// 	EnvExecCmd exec命令环境变量
	EnvExecCmd = "cloud_docker_cmd"
)

func ExecContainer(containerName string, cmdArray []string) {

}

func getContainerInfoByName(containerName string) (*container.ContainerInfo, error) {
	dirPath := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	configFilePath := dirPath + container.ConfigName
	// 读取该对应路径下的文件内容
	buf, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return nil, err
	}
	var info container.ContainerInfo
	if err = json.Unmarshal(buf, &info); err != nil {
		return nil, err
	}
	return &info, nil
}
