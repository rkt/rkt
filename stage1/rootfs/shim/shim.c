// Copyright 2014 The rkt Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

#define _GNU_SOURCE
#include <dlfcn.h>
#include <fcntl.h>
#include <stdarg.h>
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <sys/stat.h>
#include <sys/syscall.h>
#include <sys/types.h>
#include <unistd.h>


/* hack to make systemd-nspawn execute on non-sysd systems:
 * - intercept lstat() so lstat of /run/systemd/system always succeeds and returns a directory
 * - intercept close() to prevent nspawn closing the rkt lock, set it to CLOEXEC instead
 */

#define ENV_LOCKFD	"RKT_LOCK_FD"

static int (*libc_lxstat)(int, const char *, struct stat *);
static int (*libc_close)(int);
static int lock_fd = -1;

static __attribute__((constructor)) void wrapper_init(void)
{
	char *env;
	if((env = getenv(ENV_LOCKFD)))
		lock_fd = atoi(env);
	libc_lxstat = dlsym(RTLD_NEXT, "__lxstat");
	libc_close = dlsym(RTLD_NEXT, "close");
}

int __lxstat(int ver, const char *path, struct stat *stat)
{
	int ret = libc_lxstat(ver, path, stat);

	if(ret == -1 && !strcmp(path, "/run/systemd/system/")) {
		stat->st_mode = S_IFDIR;
		ret = 0;
	}

	return ret;
}

int close(int fd)
{
	if(lock_fd != -1 && fd == lock_fd)
		return fcntl(fd, F_SETFD, FD_CLOEXEC);

	return libc_close(fd);
}
