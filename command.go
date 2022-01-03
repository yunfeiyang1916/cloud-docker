package main

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"github.com/yunfeiyang1916/cloud-docker/container"
	"syscall"
)

// 内部初始化命令，不能从外部调用
var initCommand = cli.Command{
	Name:  "init",
	Usage: "Init container process run user's process in container. Do not call it outside",
	Action: func(ctx *cli.Context) error {
		// 获取传过来的命令参数，初始化容器
		logrus.Infof("初始化进程id:%d", syscall.Getpid())
		cmd := ctx.Args().Get(0)
		logrus.Infof("初始化进程中要执行的命令：%s", cmd)
		return container.Init(cmd, nil)
	},
}

// run命令执行函数
var runCommand = cli.Command{
	Name: "run",
	Usage: `Create a container with namespace and cgroups limit
			cloud-docker run -ti [command]`,
	Flags: []cli.Flag{
		cli.BoolFlag{Name: "ti", Usage: "enable tty"},
	},
	Action: func(ctx *cli.Context) error {
		if len(ctx.Args()) < 1 {
			return fmt.Errorf("缺少运行命令")
		}
		cmd := ctx.Args().Get(0)
		logrus.Infof("run中要执行的命令：%s", cmd)
		tty := ctx.Bool("ti")
		Run(tty, cmd)
		return nil
	},
}
