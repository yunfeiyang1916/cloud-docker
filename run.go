package main

import (
	"github.com/sirupsen/logrus"
	"github.com/yunfeiyang1916/cloud-docker/container"
	"os"
	"syscall"
)

// Run 执行run命令
func Run(tty bool, cmd string) {
	logrus.Infof("run进程id:%d", syscall.Getpid())
	parent := container.NewParentProcess(tty, cmd)
	// 这里的 Start 方法是真正执行前面创建好的 command 的调用，它首先会克隆出来 namespace 隔离的进程，
	// 然后在子进程中，调用/proc/self/exe ，也就是调用自己，发送 init 参数，调用我们写的init方法，去初始化容器的一些资源。
	if err := parent.Run(); err != nil {
		logrus.Errorf("parent.Run() error,err=%s", err)
		return
	}
	logrus.Infof("run进程id:%d", syscall.Getpid())
	parent.Wait()
	os.Exit(-1)
}
