package main

import (
	"fmt"
	"os"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"github.com/yunfeiyang1916/cloud-docker/cgroups/subsystems"
	"github.com/yunfeiyang1916/cloud-docker/container"
)

// 内部初始化命令，不能从外部调用
var initCommand = cli.Command{
	Name:  "init",
	Usage: "Init container process run user's process in container. Do not call it outside",
	Action: func(ctx *cli.Context) error {
		// 获取传过来的命令参数，初始化容器
		logrus.Infof("执行初始化的进程id:%d", syscall.Getpid())
		return container.RunContainerInitProcess()
	},
}

// run命令执行函数,其作用类似于运行命令时使用--来指定参数
var runCommand = cli.Command{
	Name: "run",
	Usage: `Create a container with namespace and cgroups limit
			cloud-docker run -ti [command]`,
	Flags: []cli.Flag{
		cli.BoolFlag{Name: "ti", Usage: "enable tty"},
		cli.StringFlag{Name: "v", Usage: "volume"},
		cli.BoolFlag{Name: "d", Usage: "detach container"},
		cli.StringFlag{Name: "m", Usage: "memory limit"},
		cli.StringFlag{Name: "cpushare", Usage: "cpushare limit"},
		cli.StringFlag{Name: "cpuset", Usage: "cpuset limit"},
		cli.StringFlag{Name: "name", Usage: "container name"}, // 容器名字
		cli.StringSliceFlag{Name: "e", Usage: "set environment"},
	},
	Action: func(ctx *cli.Context) error {
		// 判断参数是否包含command
		if len(ctx.Args()) < 1 {
			return fmt.Errorf("缺少运行命令")
		}
		// 获取命令
		var cmdArray []string
		for _, arg := range ctx.Args() {
			cmdArray = append(cmdArray, arg)
		}
		imageName := cmdArray[0]
		cmdArray = cmdArray[1:]
		cmd := ctx.Args().Get(0)
		logrus.Infof("run中要执行的命令：%s  进程: %d", cmd, syscall.Getpid())
		// 是否包含ti参数
		tty := ctx.Bool("ti")
		detach := ctx.Bool("d")
		// tty与后台运行不能共存
		if tty && detach {
			return fmt.Errorf("ti and d paramter can not both provided")
		}
		// 资源限制
		resConf := &subsystems.ResourceConfig{
			MemoryLimit: ctx.String("m"),
			CpuSet:      ctx.String("cpuset"),
			CpuShare:    ctx.String("cpushare"),
		}
		volume := ctx.String("v")
		containerName := ctx.String("name")
		envSlice := ctx.StringSlice("e")
		Run(tty, cmdArray, resConf, containerName, volume, imageName, envSlice)
		return nil
	},
}

// 镜像打包命令
var commitCommand = cli.Command{
	Name:  "commit",
	Usage: "commit a container into image",
	Action: func(ctx *cli.Context) error {
		if len(ctx.Args()) < 1 {
			return fmt.Errorf("missing container name")
		}
		containerName := ctx.Args().Get(0)
		imageName := ctx.Args().Get(1)
		commitContainer(containerName, imageName)
		return nil
	},
}

var listCommand = cli.Command{
	Name:  "ps",
	Usage: "list all the containers",
	Action: func(ctx *cli.Context) error {
		ListContainers()
		return nil
	},
}

var logCommand = cli.Command{
	Name:  "logs",
	Usage: "print logs of a container",
	Action: func(ctx *cli.Context) error {
		if len(ctx.Args()) < 1 {
			return fmt.Errorf("Please input your container name")
		}
		containerName := ctx.Args().Get(0)
		logContainer(containerName)
		return nil
	},
}

var execCommand = cli.Command{
	Name:  "exec",
	Usage: "exec a command into container",
	Action: func(ctx *cli.Context) error {
		// 判断是否是执行exec fork回调回来的
		if os.Getenv(EnvExecPID) != "" {
			logrus.Infof("pid callback pid %v", os.Getgid())
			return nil
		}
		// 我们希望命令格式是cloud_docker exec 容器名 命令
		if len(ctx.Args()) < 2 {
			return fmt.Errorf("missing container name or command")
		}
		containerName := ctx.Args().Get(0)
		var cmdArray []string
		// 将除了容器名之外的参数当作需要执行的命令处理
		for _, arg := range ctx.Args().Tail() {
			cmdArray = append(cmdArray, arg)
		}
		// 执行命令
		ExecContainer(containerName, cmdArray)
		return nil
	},
}

var stopCommand = cli.Command{
	Name:  "stop",
	Usage: "stop a container",
	Action: func(ctx *cli.Context) error {
		if len(ctx.Args()) < 1 {
			return fmt.Errorf("missing container name")
		}
		containerName := ctx.Args().Get(0)
		stopContainer(containerName)
		return nil
	},
}

var removeCommand = cli.Command{
	Name:  "rm",
	Usage: "remove unused containers",
	Action: func(ctx *cli.Context) error {
		if len(ctx.Args()) < 1 {
			return fmt.Errorf("missing container name")
		}
		containerName := ctx.Args().Get(0)
		removeContainer(containerName)
		return nil
	},
}
