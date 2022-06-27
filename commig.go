package main

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"os/exec"
)

func commitContainer(imageName string) {
	mntUrl := "/root/mnt"
	imagTar := "/root/" + imageName + ".tar"
	fmt.Printf("%s \n", imagTar)
	if _, err := exec.Command("tar", "-czf", imagTar, "-C", mntUrl, ".").CombinedOutput(); err != nil {
		logrus.Errorf("Tar folder %s error %v", mntUrl, err)
	}
}
