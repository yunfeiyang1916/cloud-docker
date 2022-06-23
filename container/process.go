//go:build linux
// +build linux

package container

import (
	"os"
	"os/exec"
	"syscall"

	"github.com/sirupsen/logrus"
)

// NewParentProcess 构建父进程，实际上是克隆了一个当前进程处理做环境隔离，执行init命令
func NewParentProcess(tty bool) (*exec.Cmd, *os.File) {
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
	}
	// 将读管道文件附带给子进程，子进程的第4个文件描述符就是该管道文件
	cmd.ExtraFiles = []*os.File{readPipe}
	mntUrl := "/root/mnt/"
	rootUrl := "/root/"
	NewOldWorkSpace(rootUrl, mntUrl)
	cmd.Dir = mntUrl
	//imageURL := "/tmp/image"
	// a index thing that is only needed for overlayfs do not totally understand yet
	//NewWorkSpace(imageURL)
	//cmd.Dir = "/tmp/merged"
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

func NewWorkSpace(imageURL string) {
	mergedURL := "/tmp/merged"
	indexURL := "/tmp/index"
	writeLayerURL := "/tmp/container_layer"
	// for easy coding did not check whether certain folders exists before
	// ideally should do it
	if err := os.Mkdir(writeLayerURL, 0777); err != nil {
		logrus.Errorf("Mkdir dir %s error. %v", writeLayerURL, err)
	}
	if err := os.Mkdir(mergedURL, 0777); err != nil {
		logrus.Errorf("Mkdir dir %s error. %v", mergedURL, err)
	}
	if err := os.Mkdir(indexURL, 0777); err != nil {
		logrus.Errorf("Mkdir dir %s error. %v", indexURL, err)
	}

	dirs := "lowerdir=" + imageURL + ",upperdir=" + writeLayerURL + ",workdir=" + indexURL
	logrus.Infof("overlayfs union parameters: %s", dirs)
	cmd := exec.Command("mount", "-t", "overlay", "overlay", "-o", dirs, mergedURL)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logrus.Errorf("%v", err)
	}
}

// the overlayfs created content for new created container
func DeleteWorkSpace() {
	mergedURL := "./merged"
	writeLayerURL := "./container_layer"
	indexURL := "./index"

	cmd := exec.Command("umount", mergedURL)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logrus.Errorf("%v", err)
	}
	// remove merged, index and container write layer
	if err := os.RemoveAll(mergedURL); err != nil {
		logrus.Errorf("Remove dir %s error %v", mergedURL, err)
	}
	if err := os.RemoveAll(writeLayerURL); err != nil {
		logrus.Errorf("Remove dir %s error %v", writeLayerURL, err)
	}
	if err := os.RemoveAll(indexURL); err != nil {
		logrus.Errorf("Remove dir %s error %v", indexURL, err)
	}
}

// NewWorkSpace Create a AUFS filesystem as container root workspace
func NewOldWorkSpace(rootUrl, mntUrl string) {
	CreateReadOnlyLayer(rootUrl)
	CreateWriteLayer(rootUrl)
	CreateMountPoint(rootUrl, mntUrl)
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
func DeleteOldWorkSpace(rootURL string, mntURL string) {
	DeleteMountPoint(rootURL, mntURL)
	DeleteWriteLayer(rootURL)
}

func DeleteMountPoint(rootURL string, mntURL string) {
	cmd := exec.Command("umount", mntURL)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logrus.Errorf("%v", err)
	}
	if err := os.RemoveAll(mntURL); err != nil {
		logrus.Errorf("Remove dir %s error %v", mntURL, err)
	}
}

func DeleteWriteLayer(rootURL string) {
	writeURL := rootURL + "writeLayer/"
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
