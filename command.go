package main

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"github.com/yunfeiyang1916/cloud-docker/cgroups/subsystems"
	"github.com/yunfeiyang1916/cloud-docker/container"
	"syscall"
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
		cli.StringFlag{Name: "m", Usage: "memory limit"},
		cli.StringFlag{Name: "cpushare", Usage: "cpushare limit"},
		cli.StringFlag{Name: "cpuset", Usage: "cpuset limit"},
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
		cmd := ctx.Args().Get(0)
		logrus.Infof("run中要执行的命令：%s  进程：%d", cmd, syscall.Getpid())
		// 是否包含ti参数
		tty := ctx.Bool("ti")
		// 资源限制
		resConf := &subsystems.ResourceConfig{
			MemoryLimit: ctx.String("m"),
			CpuSet:      ctx.String("cpuset"),
			CpuShare:    ctx.String("cpushare"),
		}
		Run(tty, cmdArray, resConf)
		return nil
	},
}
