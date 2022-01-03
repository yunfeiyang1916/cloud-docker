//go:build linux
// +build linux

package container

import (
	"github.com/sirupsen/logrus"
	"os"
	"syscall"
)

// Init 是在容器内部执行的，也就是说代码执行到这里后,容器所在的进程其实就已经创建出来了，
// 这是本容器执行的第1个进程。使用 mount 先去挂载 proc 文件系统，以便后面通过 ps 等系统命令去查看当前进程资源使用情况。
func Init(cmd string, args []string) error {
	logrus.Infof("容器初始化执行命令:%s,进程id:%d", cmd, syscall.Getpid())
	// MS_NODEV linux2.4之后的默认参数
	// MS_NOEXEC 在本文件系统允许许运行其他程序
	// MS_NOSUID 在本系统中运行程序的时候， 允许 set-user-ID set-group-ID
	defaultMountFlags := syscall.MS_NODEV | syscall.MS_NOEXEC | syscall.MS_NOSUID
	// 使用mount挂载proc文件系统，以便后面通过ps命令查询当前进程使用资源情况
	syscall.Mount("proc", "/proc", "proc", uintptr(defaultMountFlags), "")
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
	argv := []string{cmd}
	if err := syscall.Exec(cmd, argv, os.Environ()); err != nil {
		logrus.Error(err.Error())
		return err
	}
	return nil
}
