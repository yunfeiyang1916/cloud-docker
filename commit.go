package main

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/yunfeiyang1916/cloud-docker/container"
	"os/exec"
)

func commitContainer(containerName, imageName string) {
	mntUrl := fmt.Sprintf(container.MntUrl, containerName) + "/"
	imagTar := container.RootUrl + "/" + imageName + ".tar"
	fmt.Printf("%s \n", imagTar)
	if _, err := exec.Command("tar", "-czf", imagTar, "-C", mntUrl, ".").CombinedOutput(); err != nil {
		logrus.Errorf("Tar folder %s error %v", mntUrl, err)
	}
}
