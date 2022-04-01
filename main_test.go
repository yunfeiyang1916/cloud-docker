package main

import (
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"os"
	"os/exec"
	"syscall"
	"testing"
)

func TestRun(t *testing.T) {
	logrus.Infof("当前进程id:%d", syscall.Getpid())
	app := cli.NewApp()
	app.Name = "docker"
	app.Usage = `这是说明啊，哈哈
				 好的`
	app.Commands = []cli.Command{
		test_initCommand,
		test_runCommand,
	}
	app.Before = func(ctx *cli.Context) error {
		logrus.SetFormatter(&logrus.JSONFormatter{})
		logrus.SetOutput(os.Stdout)
		return nil
	}
	if err := app.Run(os.Args); err != nil {
		logrus.Fatal(err)
	}
}

var test_initCommand = cli.Command{
	Name:  "init",
	Usage: "初始化命令",
	Action: func(ctx *cli.Context) error {
		logrus.Infof("init 命令参数 %v,ti", ctx.Args())
		logrus.Infof("init命令中的进程id:%d", syscall.Getpid())
		return nil
	},
}

var test_runCommand = cli.Command{
	Name:  "run",
	Usage: "run命令",
	Flags: []cli.Flag{
		cli.BoolFlag{Name: "ti", Usage: "使用终端交互"},
	},
	Action: func(ctx *cli.Context) error {
		logrus.Infof("run 命令参数 %v,ti", ctx.Args())
		// 执行run命令
		logrus.Infof("run命令中的进程id:%d", syscall.Getpid())
		argv := []string{"init", ctx.Args().Get(0)}
		cmd := exec.Command("/proc/self/exe", argv...)
		if err := cmd.Run(); err != nil {
			logrus.Error(err)
		}
		return nil
	},
}
