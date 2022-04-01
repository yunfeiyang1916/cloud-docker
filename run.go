package main

import (
	"github.com/sirupsen/logrus"
	"github.com/yunfeiyang1916/cloud-docker/cgroups"
	"github.com/yunfeiyang1916/cloud-docker/cgroups/subsystems"
	"github.com/yunfeiyang1916/cloud-docker/container"
	"os"
	"strings"
)

// Run 执行run命令
func Run(tty bool, cmdArray []string, res *subsystems.ResourceConfig) {
	parent, writePipe := container.NewParentProcess(tty)
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

	// 使用cloud-docker-cgroup 作为cgroup名称
	// 创建cgroup manager,并通过调用set和apply设置资源限制并限制在容器生效
	cgroupManager := cgroups.NewCgroupManager("cloud-docker-cgroup")
	defer cgroupManager.Destroy()
	// 设置资源限制
	cgroupManager.Set(res)
	// 将容器进程加入到各个subsystem挂载对应的cgroup中
	cgroupManager.Apply(parent.Process.Pid)
	// 对容器设置完限制后，初始化容器
	sendInitCommand(cmdArray, writePipe)
	parent.Wait()
}

// 通过匿名管道向初始化进程发送命令
func sendInitCommand(cmdArray []string, writePipe *os.File) {
	command := strings.Join(cmdArray, " ")
	logrus.Infof("command all is %s", command)
	writePipe.WriteString(command)
	writePipe.Close()
}
