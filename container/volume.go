package container

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"strings"
)

// NewWorkSpace Create a AUFS filesystem as container root workspace
func NewWorkSpace(volume, imageName, containerName string) {
	CreateReadOnlyLayer(imageName)
	CreateWriteLayer(containerName)
	CreateMountPoint(containerName, imageName)
	// 根据volume判断是否执行挂载数据卷操作
	if volume != "" {
		volumeUrls := volumeUrlExtract(volume)
		length := len(volumeUrls)
		if length == 2 && volumeUrls[0] != "" && volumeUrls[1] != "" {
			MountVolume(volumeUrls, containerName)
			logrus.Infof("%q", volumeUrls)
		} else {
			logrus.Infof("数据卷参数不正确")
		}
	}
}

// MountVolume 挂载数据卷
func MountVolume(volumeUrls []string, containerName string) error {
	// 创建宿主机目录
	parentUrl := volumeUrls[0]
	if err := os.Mkdir(parentUrl, 0777); err != nil {
		logrus.Infof("Mkdir parent dir %s error.%v", parentUrl, err)
	}
	// 在容器文件系统里创建挂载点
	mntUrl := fmt.Sprintf(MntUrl, containerName)
	containerVolumeUrl := mntUrl + "/" + volumeUrls[1]
	if err := os.Mkdir(containerVolumeUrl, 0777); err != nil {
		logrus.Infof("Mkdir container dir %s error.%v", containerVolumeUrl, err)
	}
	// 把宿主机文件目录挂载到容器挂载点
	dirs := "dirs=" + parentUrl
	if _, err := exec.Command("mount", "-t", "aufs", "-o", dirs, "none", containerVolumeUrl).CombinedOutput(); err != nil {
		logrus.Errorf("Mount volume failed.%v", err)
		return err
	}
	return nil
}

// 解析volume字符串
func volumeUrlExtract(volume string) []string {
	return strings.Split(volume, ":")
}

// CreateReadOnlyLayer 将busybox.tar解压到busybox目录下,作为容器的只读层
func CreateReadOnlyLayer(imageName string) error {
	unTarFolderUrl := RootUrl + "/" + imageName + "/"
	imageUrl := RootUrl + "/" + imageName + ".tar"
	exist, err := PathExists(unTarFolderUrl)
	if err != nil {
		logrus.Infof("Fail to judge whether dir %s exists. %v", unTarFolderUrl, err)
		return err
	}
	if !exist {
		if err = os.Mkdir(unTarFolderUrl, 0777); err != nil {
			logrus.Errorf("Mkdir dir %s error. %v", unTarFolderUrl, err)
			return err
		}
		if _, err = exec.Command("tar", "-xvf", imageUrl, "-C", unTarFolderUrl).CombinedOutput(); err != nil {
			logrus.Errorf("Untar dir %s error %v", unTarFolderUrl, err)
			return err
		}
	}
	return nil
}

// CreateWriteLayer 创建一个名为writeLayer的文件夹作为容器唯一的可写层
func CreateWriteLayer(containerName string) {
	writeURL := fmt.Sprintf(WriteLayerUrl, containerName)
	if err := os.MkdirAll(writeURL, 0777); err != nil {
		logrus.Errorf("Mkdir dir %s error. %v", writeURL, err)
	}
}

func CreateMountPoint(containerName, imageName string) error {
	// 创建mnt文件夹作为挂载点
	mntUrl := fmt.Sprintf(MntUrl, containerName)
	if err := os.MkdirAll(mntUrl, 0777); err != nil {
		logrus.Errorf("Mkdir dir %s error. %v", mntUrl, err)
	}
	tmpWriteLayer := fmt.Sprintf(WriteLayerUrl, containerName)
	tmpImageLocation := RootUrl + "/" + imageName
	dirs := "dirs=" + tmpWriteLayer + ":" + tmpImageLocation
	if _, err := exec.Command("mount", "-t", "aufs", "-o", dirs, "none", mntUrl).CombinedOutput(); err != nil {
		logrus.Errorf("run command for creating mount point failed %v", err)
		return err
	}
	return nil
}

// DeleteWorkSpace Delete the AUFS filesystem while container exit
func DeleteWorkSpace(volume, containerName string) {
	if volume != "" {
		volumeUrls := volumeUrlExtract(volume)
		length := len(volumeUrls)
		if length == 2 && volumeUrls[0] != "" && volumeUrls[1] != "" {
			DeleteMountPointWithVolume(volumeUrls, containerName)
		} else {
			DeleteMountPoint(containerName)
		}
	} else {
		DeleteMountPoint(containerName)
	}
	DeleteWriteLayer(containerName)
}

func DeleteMountPointWithVolume(volumeUrls []string, containerName string) {
	mntUrl := fmt.Sprintf(MntUrl, containerName)
	// 卸载容器里volume挂载点的文件系统
	containerUrl := mntUrl + "/" + volumeUrls[1]
	if _, err := exec.Command("umount", containerUrl).CombinedOutput(); err != nil {
		logrus.Errorf("umount volume failed.%v", err)
	}
	// 卸载整个容器文件系统的挂载点
	DeleteMountPoint(containerName)
}

func DeleteMountPoint(containerName string) error {
	mntUrl := fmt.Sprintf(MntUrl, containerName)
	cmd := exec.Command("umount", mntUrl)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logrus.Errorf("%v", err)
		return err
	}
	if err := os.RemoveAll(mntUrl); err != nil {
		logrus.Errorf("Remove dir %s error %v", mntUrl, err)
		return err
	}
	return nil
}

func DeleteWriteLayer(containerName string) {
	writeURL := fmt.Sprintf(WriteLayerUrl, containerName)
	if err := os.RemoveAll(writeURL); err != nil {
		logrus.Errorf("Remove dir %s error %v", writeURL, err)
	}
}

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
