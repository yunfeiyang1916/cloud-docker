//go:build linux
// +build linux

package container

import (
	"github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"syscall"
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
