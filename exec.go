package main

import (
	"fmt"
	"github.com/sirupsen/logrus"
	_ "github.com/yunfeiyang1916/cloud-docker/nsenter"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
)

const (
	// EnvExecPID pid环境变量
	EnvExecPID = "cloud_docker_pid"
	// 	EnvExecCmd exec命令环境变量
	EnvExecCmd = "cloud_docker_cmd"
)

func ExecContainer(containerName string, cmdArray []string) {
	info, err := getContainerInfo(containerName)
	if err != nil {
		logrus.Errorf("ExecContainer getContainerInfoByName %s error %v", containerName, err)
		return
	}
	// 把命令以空格为分隔符拼接成一个字符串，便于传递
	cmdStr := strings.Join(cmdArray, " ")
	logrus.Infof("container pid %s", info.Pid)
	logrus.Infof("command %s", cmdStr)
	// 克隆自己，执行exec命令
	cmd := exec.Command("/proc/self/exe", "exec")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	os.Setenv(EnvExecPID, info.Pid)
	os.Setenv(EnvExecCmd, cmdStr)
	envs := getEnvsByPid(info.Pid)
	cmd.Env = append(os.Environ(), envs...)
	if err = cmd.Run(); err != nil {
		logrus.Errorf("exec container %s error %v", containerName, err)
	}
}

// 获取指定进程的环境变量
func getEnvsByPid(pid string) []string {
	path := fmt.Sprintf("/proc/%s/environ", pid)
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		logrus.Errorf("read file %s error %v", path, err)
		return nil
	}
	// 多个环境变量中的分隔符是\u0000
	envs := strings.Split(string(buf), "\u0000")
	return envs
}
