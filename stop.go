package main

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/yunfeiyang1916/cloud-docker/container"
	"io/ioutil"
	"os"
	"strconv"
	"syscall"
)

func stopContainer(containerName string) {
	info, err := getContainerInfo(containerName)
	if err != nil {
		logrus.Errorf("Get container info by name %s error %v", containerName, err)
		return
	}
	pid, err := strconv.Atoi(info.Pid)
	if err != nil {
		logrus.Errorf("Conver pid from string to int error %v", err)
		return
	}
	// 系统调用kill可以发送信号给进程，通过传递syscall.SIGTERM信号，kill掉容器住进程
	if err = syscall.Kill(pid, syscall.SIGTERM); err != nil {
		logrus.Errorf("Stop container %s error %v", containerName, err)
		return
	}
	// 至此，容器进程已经被kill，所以下面需要修改容器的状态,PID可以置为空
	info.Status = container.Stop
	info.Pid = ""
	buf, err := json.Marshal(info)
	if err != nil {
		logrus.Errorf("Json marshal %s error %v", containerName, err)
		return
	}
	filePath := fmt.Sprintf(container.DefaultInfoLocation, containerName) + container.ConfigName
	// 重新写入新的数据覆盖原来的信息
	if err = ioutil.WriteFile(filePath, buf, 0622); err != nil {
		logrus.Errorf("Write file %s error %v", filePath, err)
	}
}

func removeContainer(containerName string) {
	info, err := getContainerInfo(containerName)
	if err != nil {
		logrus.Errorf("Get container info by name %s error %v", containerName, err)
		return
	}
	// 只删除处于停止状态的容器
	if info.Status != container.Stop {
		logrus.Errorf("Couldn't remove running container")
		return
	}
	dirPath := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	// 将所有信息包括子目录都移除
	if err = os.RemoveAll(dirPath); err != nil {
		logrus.Errorf("remove file %s error %v", dirPath, err)
		return
	}
	container.DeleteWorkSpace(info.Volume, containerName)
}
