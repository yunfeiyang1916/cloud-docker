//go:build linux
// +build linux

package container

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
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
}

// NewParentProcess 构建父进程，实际上是克隆了一个当前进程处理做环境隔离，执行init命令
func NewParentProcess(tty bool, volume string, containerName string) (*exec.Cmd, *os.File) {
	readPipe, writePipe, err := NewPipe()
	if err != nil {
		logrus.Errorf("New pipe error %v", err)
		return nil, nil
	}
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
	//mntUrl := "/root/mnt/"
	//rootUrl := "/root/"
	//NewWorkSpace(rootUrl, mntUrl, volume)
	//cmd.Dir = mntUrl
	cmd.Dir = "/root/busybox"
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

// NewWorkSpace Create a AUFS filesystem as container root workspace
func NewWorkSpace(rootUrl, mntUrl, volume string) {
	CreateReadOnlyLayer(rootUrl)
	CreateWriteLayer(rootUrl)
	CreateMountPoint(rootUrl, mntUrl)
	// 根据volume判断是否执行挂载数据卷操作
	if volume != "" {
		volumeUrls := volumeUrlExtract(volume)
		length := len(volumeUrls)
		if length == 2 && volumeUrls[0] != "" && volumeUrls[1] != "" {
			MountVolume(rootUrl, mntUrl, volumeUrls)
			logrus.Infof("%q", volumeUrls)
		} else {
			logrus.Infof("数据卷参数不正确")
		}
	}
}

// MountVolume 挂载数据卷
func MountVolume(rootUrl, mntUrl string, volumeUrls []string) {
	// 创建宿主机目录
	parentUrl := volumeUrls[0]
	if err := os.Mkdir(parentUrl, 0777); err != nil {
		logrus.Infof("Mkdir parent dir %s error.%v", parentUrl, err)
	}
	// 在容器文件系统里创建挂载点
	containerVolumeUrl := mntUrl + volumeUrls[1]
	if err := os.Mkdir(containerVolumeUrl, 0777); err != nil {
		logrus.Infof("Mkdir container dir %s error.%v", containerVolumeUrl, err)
	}
	// 把宿主机文件目录挂载到容器挂载点
	dirs := "dirs=" + parentUrl
	cmd := exec.Command("mount", "-t", "aufs", "-o", dirs, "none", containerVolumeUrl)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logrus.Errorf("Mount volume failed.%v", err)
	}
}

// 解析volume字符串
func volumeUrlExtract(volume string) []string {
	return strings.Split(volume, ":")
}

// CreateReadOnlyLayer 将busybox.tar解压到busybox目录下,作为容器的只读层
func CreateReadOnlyLayer(rootUrl string) {
	busyboxUrl := rootUrl + "busybox/"
	busyboxTarUrl := rootUrl + "busybox.tar"
	exist, err := PathExists(busyboxUrl)
	if err != nil {
		logrus.Infof("Fail to judge whether dir %s exists. %v", busyboxUrl, err)
	}
	if !exist {
		if err = os.Mkdir(busyboxUrl, 0777); err != nil {
			logrus.Errorf("Mkdir dir %s error. %v", busyboxUrl, err)
		}
		if _, err = exec.Command("tar", "-xvf", busyboxTarUrl, "-C", busyboxUrl).CombinedOutput(); err != nil {
			logrus.Errorf("Untar dir %s error %v", busyboxUrl, err)
		}
	}
}

// CreateWriteLayer 创建一个名为writeLayer的文件夹作为容器唯一的可写层
func CreateWriteLayer(rootUrl string) {
	writeURL := rootUrl + "writeLayer/"
	if err := os.Mkdir(writeURL, 0777); err != nil {
		logrus.Errorf("Mkdir dir %s error. %v", writeURL, err)
	}
}

func CreateMountPoint(rootUrl, mntUrl string) {
	// 创建mnt文件夹作为挂载点
	if err := os.Mkdir(mntUrl, 0777); err != nil {
		logrus.Errorf("Mkdir dir %s error. %v", mntUrl, err)
	}
	// 把writeLayer目录和busybox目录mount到mnt目录下
	dirs := "dirs=" + rootUrl + "writeLayer:" + rootUrl + "busybox"
	cmd := exec.Command("mount", "-t", "aufs", "-o", dirs, "none", mntUrl)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logrus.Errorf("%v", err)
	}
}

// DeleteWorkSpace Delete the AUFS filesystem while container exit
func DeleteWorkSpace(rootUrl, mntUrl, volume string) {
	if volume != "" {
		volumeUrls := volumeUrlExtract(volume)
		length := len(volumeUrls)
		if length == 2 && volumeUrls[0] != "" && volumeUrls[1] != "" {
			DeleteMountPointWithVolume(rootUrl, mntUrl, volumeUrls)
		} else {
			DeleteMountPoint(rootUrl, mntUrl)
		}
	} else {
		DeleteMountPoint(rootUrl, mntUrl)
	}
	DeleteWriteLayer(rootUrl)
}

func DeleteMountPointWithVolume(rootUrl, mntUrl string, volumeUrls []string) {
	// 卸载容器里volume挂载点的文件系统
	containerUrl := mntUrl + volumeUrls[1]
	cmd := exec.Command("umount", containerUrl)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logrus.Errorf("umount volume failed.%v", err)
	}
	// 卸载整个容器文件系统的挂载点
	DeleteMountPoint(rootUrl, mntUrl)
}

func DeleteMountPoint(rootUrl, mntUrl string) {
	cmd := exec.Command("umount", mntUrl)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logrus.Errorf("%v", err)
	}
	if err := os.RemoveAll(mntUrl); err != nil {
		logrus.Errorf("Remove dir %s error %v", mntUrl, err)
	}
}

func DeleteWriteLayer(rootUrl string) {
	writeURL := rootUrl + "writeLayer/"
	if err := os.RemoveAll(writeURL); err != nil {
		logrus.Errorf("Remove dir %s error %v", writeURL, err)
	}
}

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
