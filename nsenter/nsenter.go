package nsenter

/*
#include <errno.h>
#include <sched.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <fcntl.h>

// __attribute__((constructor)) 类似构造函数，这个包一旦被引用，这个函数会自动执行。也就是会在程序一启动的时候运行
__attribute__((constructor)) void enter_namespace(void) {
	char *cloud_docker_pid;
	// 从环境变量中获取需要进入的PID
    cloud_docker_pid=getenv("cloud_docker_pid");
    if(cloud_docker_pid){
		fprintf(stdout, "got cloud_docker_pid=%s\n", cloud_docker_pid);
	}else{
		fprintf(stdout, "missing cloud_docker_pid env skip nsenter");
		// 未设置pid环境变量说明不是执行exec命令，返回
		return;
	}
	char *cloud_docker_cmd;
	cloud_docker_cmd = getenv("cloud_docker_cmd");
	if (cloud_docker_cmd) {
		fprintf(stdout, "got cloud_docker_cmd=%s\n", cloud_docker_cmd);
	} else {
		fprintf(stdout, "missing cloud_docker_cmd env skip nsenter");
		return;
	}
	int i;
	char nspath[1024];
	// 需要进入的五种Namespace
	char *namespaces[] = { "ipc", "uts", "net", "pid", "mnt" };
	for (i=0; i<5; i++) {
		// 拼接对应路径，比如/proc/pid/ns/ipc
		sprintf(nspath, "/proc/%s/ns/%s", cloud_docker_pid, namespaces[i]);
		int fd = open(nspath, O_RDONLY);
		// 这里才真正调用setns系统调用进入对应的Namespace
		if (setns(fd, 0) == -1) {
			fprintf(stderr, "setns on %s namespace failed: %s\n", namespaces[i], strerror(errno));
		} else {
			fprintf(stdout, "setns on %s namespace succeeded\n", namespaces[i]);
		}
		close(fd);
	}
	// 在进入的Namespace中执行指定的命令
	int res = system(cloud_docker_cmd);
	// 退出
	exit(0);
	return;
}
*/
import "C"
