package main

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/yunfeiyang1916/cloud-docker/container"
	"io/ioutil"
	"os"
	"text/tabwriter"
)

func ListContainers() {
	dirPath := fmt.Sprintf(container.DefaultInfoLocation, "")
	// 去除最后一个字符
	dirPath = dirPath[:len(dirPath)-1]
	// 读取该文件夹下的所有文件
	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		logrus.Errorf("read dir %s error %v", dirPath, err)
		return
	}
	var infoList []*container.ContainerInfo
	// 遍历所有文件
	for _, file := range files {
		// 根据容器配置文件获取对应信息
		info, err := getContainerInfo(file.Name())
		if err != nil {
			logrus.Errorf("getContainerInfo error %s", err)
			continue
		}
		infoList = append(infoList, info)
	}
	// 使用tabwriter打印容器信息
	w := tabwriter.NewWriter(os.Stdout, 12, 1, 3, ' ', 0)
	fmt.Fprint(w, "ID\tNAME\tPID\tSTATUS\tCOMMAND\tCREATED\n")
	for _, item := range infoList {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			item.Id,
			item.Name,
			item.Pid,
			item.Status,
			item.Command,
			item.CreatedTime)
	}
	// 刷新标准输出流到缓存区
	if err = w.Flush(); err != nil {
		logrus.Errorf("Flush error %s", err)
		return
	}
}

func getContainerInfo(containerName string) (*container.ContainerInfo, error) {
	configFilePath := fmt.Sprintf(container.DefaultInfoLocation, containerName) + container.ConfigName
	content, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		logrus.Errorf("read file %s error %v", configFilePath, err)
		return nil, err
	}
	var info container.ContainerInfo
	if err = json.Unmarshal(content, &info); err != nil {
		logrus.Errorf("json.Unmarshal error %s", err)
		return nil, err
	}
	return &info, nil
}
