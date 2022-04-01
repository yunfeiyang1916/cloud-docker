//go:build linux
// +build linux

package container

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

// RunContainerInitProcess 是在容器内部执行的，也就是说代码执行到这里后,容器所在的进程其实就已经创建出来了，这是本容器执行的第1个进程。
// 使用 mount 先去挂载 proc 文件系统，以便后面通过 ps 等系统命令去查看当前进程资源使用情况。
func RunContainerInitProcess() error {
	cmdArray := readUserCommand()
	if cmdArray == nil || len(cmdArray) == 0 {
		return fmt.Errorf("Run container get user command error, cmdArray is nil")
	}
	// MS_NODEV linux2.4之后的默认参数
	// MS_NOEXEC 在本文件系统允许许运行其他程序
	// MS_NOSUID 在本系统中运行程序的时候， 允许 set-user-ID set-group-ID
	defaultMountFlags := syscall.MS_NODEV | syscall.MS_NOEXEC | syscall.MS_NOSUID
	// 使用mount挂载proc文件系统，以便后面通过ps命令查询当前进程使用资源情况
	syscall.Mount("proc", "/proc", "proc", uintptr(defaultMountFlags), "")
	// 调用exec.LookPath，可以在系统的PATH路面寻找命令的绝对路径
	path, err := exec.LookPath(cmdArray[0])
	if err != nil {
		logrus.Errorf("Exec loop path error %v", err)
		return err
	}
	logrus.Infof("Find path %s", path)
	// 如果使用下面这种调用的话，进程id为1的会是容器进程而不是用户进程
	//c := exec.Command(cmd)
	//c.Stdin = os.Stdin
	//c.Stdout = os.Stdout
	//c.Stderr = os.Stderr
	//if err := c.Run(); err != nil {
	//	logrus.Error(err.Error())
	//	return err
	//}
	// 使用下面的系统调用可以使用户进程覆盖掉容器进程，从而使得用户进程的id可以为1
	if err := syscall.Exec(path, cmdArray[0:], os.Environ()); err != nil {
		logrus.Error(err.Error())
		return err
	}
	return nil
}

// 读取用户的初始化命令及其参数
func readUserCommand() []string {
	// 第4个文件（下标从0开始）传过来的是匿名读管道文件
	pipe := os.NewFile(uintptr(3), "pipe")
	// 如果父进程还没写入文件，读操作会阻塞在这里
	msg, err := ioutil.ReadAll(pipe)
	if err != nil {
		logrus.Errorf("init read pipe error %v", err)
		return nil
	}
	msgStr := string(msg)
	return strings.Split(msgStr, " ")
}
