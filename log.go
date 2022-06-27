package main

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/yunfeiyang1916/cloud-docker/container"
	"io/ioutil"
	"os"
)

func logContainer(containerName string) {
	logFilePath := fmt.Sprintf(container.DefaultInfoLocation, containerName) + container.ContainerLogFile
	// 打开日志文件
	file, err := os.Open(logFilePath)
	if err != nil {
		logrus.Errorf("log container open file %s error %v", logFilePath, err)
		return
	}
	// 将文件的内容都读取出来
	content, err := ioutil.ReadAll(file)
	if err != nil {
		logrus.Errorf("log container read file %s error %v", logFilePath, err)
		return
	}
	// 输出到控制台
	fmt.Fprint(os.Stdout, string(content))
}
