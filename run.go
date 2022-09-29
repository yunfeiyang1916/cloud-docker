package main

import (
	"encoding/json"
	"fmt"
	"github.com/yunfeiyang1916/cloud-docker/network"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/yunfeiyang1916/cloud-docker/cgroups/subsystems"
	"github.com/yunfeiyang1916/cloud-docker/container"
)

// Run 执行run命令
func Run(tty bool, cmdArray []string, res *subsystems.ResourceConfig, containerName, volume, imageName string, envSlice []string, nw string, portmapping []string) {
	containerID := randStringBytes(10)
	if containerName == "" {
		containerName = containerID
	}
	parent, writePipe := container.NewParentProcess(tty, containerName, volume, imageName, envSlice)
	if parent == nil {
		logrus.Errorf("New parent process error")
		return
	}
	// 这里的 Start 方法是真正执行前面创建好的 command 的调用，它首先会克隆出来 namespace 隔离的进程，
	// 然后在子进程中，调用/proc/self/exe ，也就是调用自己，发送 init 参数，调用我们写的init方法，去初始化容器的一些资源。
	if err := parent.Start(); err != nil {
		logrus.Errorf("parent.Run() error,err=%s", err)
		return
	}
	// 记录容器信息
	containerName, err := recordContainerInfo(parent.Process.Pid, cmdArray, containerName, containerID, volume)
	if err != nil {
		logrus.Errorf("record container info error %s", err)
		return
	}
	// 使用cloud-docker-cgroup 作为cgroup名称
	// 创建cgroup manager,并通过调用set和apply设置资源限制并限制在容器生效
	//cgroupManager := cgroups.NewCgroupManager("cloud-docker-cgroup")
	//defer cgroupManager.Destroy()
	//// 设置资源限制
	//cgroupManager.Set(res)
	//// 将容器进程加入到各个subsystem挂载对应的cgroup中
	//cgroupManager.Apply(parent.Process.Pid)

	if nw != "" {
		network.Init()
		containerInfo := &container.ContainerInfo{
			Id:          containerID,
			Pid:         strconv.Itoa(parent.Process.Pid),
			Name:        containerName,
			PortMapping: portmapping,
		}
		if err := network.Connect(nw, containerInfo); err != nil {
			logrus.Errorf("Error Connect Network %v", err)
			return
		}
	}

	// 对容器设置完限制后，初始化容器
	sendInitCommand(cmdArray, writePipe)
	if tty {
		// 如果是交互式的，父进程需要等待子进程结束
		parent.Wait()
		deleteContainerInfo(containerName)
		container.DeleteWorkSpace(volume, containerName)
	}
}

// 通过匿名管道向初始化进程发送命令
func sendInitCommand(cmdArray []string, writePipe *os.File) {
	command := strings.Join(cmdArray, " ")
	logrus.Infof("command all is %s", command)
	writePipe.WriteString(command)
	writePipe.Close()
}

// 记录容器信息
func recordContainerInfo(containerPID int, cmdArray []string, containerName, id, volume string) (string, error) {
	now := time.Now().Format("2006-01-02 15:04:05")
	command := strings.Join(cmdArray, "")
	info := &container.ContainerInfo{
		Pid:         strconv.Itoa(containerPID),
		Id:          id,
		Name:        containerName,
		Command:     command,
		CreatedTime: now,
		Status:      container.Running,
		Volume:      volume,
	}
	buf, err := json.Marshal(info)
	if err != nil {
		logrus.Errorf("json.Marshal error,%s", err)
		return "", err
	}
	dirPath := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	if err = os.MkdirAll(dirPath, 0622); err != nil {
		logrus.Errorf("MkdirAll %s error %s", dirPath, err)
		return "", err
	}
	fileName := dirPath + "/" + container.ConfigName
	// 创建配置文件
	file, err := os.Create(fileName)
	if err != nil {
		logrus.Errorf("Create file %s error %s", fileName, err)
		return "", err
	}
	defer file.Close()
	if _, err = file.Write(buf); err != nil {
		logrus.Errorf("file write error %s", err)
		return "", err
	}
	return containerName, nil
}

func deleteContainerInfo(containerName string) {
	dirPath := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	if err := os.RemoveAll(dirPath); err != nil {
		logrus.Errorf("remove all dir %s error %s", dirPath, err)
	}
}

// 生成随机id
func randStringBytes(n int) string {
	letterBytes := "1234567890"
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}
