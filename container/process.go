//go:build linux
// +build linux

package container

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/sirupsen/logrus"
)

const (
	Running             = "running"
	Stop                = "stopped"
	Exit                = "exited"
	DefaultInfoLocation = "/var/run/cloud-docker/%s/"
	ConfigName          = "config.json"
	ContainerLogFile    = "container.log"
	RootUrl             = "/root"
	MntUrl              = "/root/mnt/%s"
	WriteLayerUrl       = "/root/writeLayer/%s"
)

// ContainerInfo 容器信息
type ContainerInfo struct {
	// 容器的init进程在宿主机上的 PID
	Pid string `json:"pid"`
	// 容器Id
	Id string `json:"id"`
	// 容器名
	Name string `json:"name"`
	// 容器内init运行命令
	Command string `json:"command"`
	// 创建时间
	CreatedTime string `json:"createTime"`
	// 容器的状态
	Status string `json:"status"`
	// 容器的数据卷
	Volume string `json:"volume"`
	// 端口映射
	PortMapping []string `json:"portmapping"`
}

// NewParentProcess 构建父进程，实际上是克隆了一个当前进程处理做环境隔离，执行init命令
func NewParentProcess(tty bool, containerName, volume, imageName string, envSlice []string) (*exec.Cmd, *os.File) {
	readPipe, writePipe, err := NewPipe()
	if err != nil {
		logrus.Errorf("New pipe error %v", err)
		return nil, nil
	}
	// 克隆自己，执行init命令
	cmd := exec.Command("/proc/self/exe", "init")
	// 下面的 clone 参数就是去 fork 出来一个新进程，并且使用了 namespace 隔离新创建的进程和外部环境
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS |
			syscall.CLONE_NEWNET | syscall.CLONE_NEWIPC,
	}
	// 是否需要设置交互式终端，将当前进程的输入输出导入到标准输入输出上
	if tty {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	} else {
		// 生成容器日志
		dirPath := fmt.Sprintf(DefaultInfoLocation, containerName)
		if err = os.MkdirAll(dirPath, 0622); err != nil {
			logrus.Errorf("NewParentProcess mkdir %s error %v", dirPath, err)
			return nil, nil
		}
		stdLogFilePath := dirPath + ContainerLogFile
		stdLogFile, err := os.Create(stdLogFilePath)
		if err != nil {
			logrus.Errorf("NewParentProcess create file %s error %v", stdLogFilePath, err)
			return nil, nil
		}
		cmd.Stdout = stdLogFile
	}
	// 将读管道文件附带给子进程，子进程的第4个文件描述符就是该管道文件
	cmd.ExtraFiles = []*os.File{readPipe}
	cmd.Env = append(os.Environ(), envSlice...)
	NewWorkSpace(volume, imageName, containerName)
	cmd.Dir = fmt.Sprintf(MntUrl, containerName)
	// cmd.Dir = "/root/busybox"
	return cmd, writePipe
}

// NewPipe 创建匿名管道，供init进程与run进程通信
func NewPipe() (*os.File, *os.File, error) {
	read, write, err := os.Pipe()
	if err != nil {
		return nil, nil, err
	}
	return read, write, nil
}
