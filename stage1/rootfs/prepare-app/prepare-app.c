// Copyright 2015 CoreOS, Inc.
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

#include <errno.h>
#include <string.h>
#include <stdio.h>
#include <stdlib.h>
#include <sys/mount.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <unistd.h>
#include <fcntl.h>

static int exit_err;
#define exit_if(_cond, _fmt, _args...)				\
	exit_err++;						\
	if(_cond) {						\
		fprintf(stderr, "Error: " _fmt "\n", ##_args);	\
		exit(exit_err);					\
	}
#define pexit_if(_cond, _fmt, _args...)				\
	exit_if(_cond, _fmt ": %s", ##_args, strerror(errno))

int main(int argc, char *argv[])
{
	static const char *dirs[] = {
		"dev",
		"dev/net",
		"proc",
		NULL
	};
	static const char *devnodes[] = {
		"/dev/null",
		"/dev/zero",
		"/dev/full",
		"/dev/random",
		"/dev/urandom",
		"/dev/tty",
		"/dev/net/tun",
		"/dev/console",
		NULL
	};
	const char *root;
	int rootfd;
	char to[4096];
	int i;

	exit_if(argc < 2,
		"Usage: %s /path/to/root", argv[0]);

	root = argv[1];

	rootfd = open(root, O_DIRECTORY | O_CLOEXEC);
	pexit_if(rootfd < 0,
		"Failed to open directory \"%s\"", root);

	/* First, create the directories */
	for (i = 0; dirs[i]; i++) {
		pexit_if(mkdirat(rootfd, dirs[i], 0755) == -1 && errno != EEXIST,
			"Failed to create directory \"%s/%s\"", root, dirs[i]);
	}
	close(rootfd);

	/* systemd-nspawn already creates few /dev entries in the container
	 * namespace: copy_devnodes()
	 * http://cgit.freedesktop.org/systemd/systemd/tree/src/nspawn/nspawn.c?h=v219#n1345
	 *
	 * But they are not visible by the apps because they are "protected" by
	 * the chroot.
	 *
	 * Bind mount them individually over the chroot border.
	 *
	 * Do NOT bind mount the whole directory /dev because it would shadow
	 * potential individual bind mount by stage0 ("rkt run --volume...").
	 *
	 * Do NOT use mknod, it would not work for /dev/console because it is
	 * a bind mount to a pts and pts device nodes only work when they live
	 * on a devpts filesystem.
	 */
	for (i = 0; devnodes[i]; i++) {
		const char *from = devnodes[i];
		int fd;

		/* If the file does not exist, skip it. It might be because
		 * the kernel does not provide it (e.g. kernel compiled without
		 * CONFIG_TUN) or because systemd-nspawn does not provide it
		 * (/dev/net/tun is not available with systemd-nspawn < v217
		 */
		if (access(from, F_OK) != 0)
			continue;

		exit_if(snprintf(to, sizeof(to), "%s%s", root, from) >= sizeof(to),
			"Path too long: \"%s\"", to);

		/* The mode does not matter: it will be bind-mounted over.
		 */
		fd = open(to, O_WRONLY|O_CREAT|O_CLOEXEC|O_NOCTTY, 0644);
		if (fd != -1)
			close(fd);

		pexit_if(mount(from, to, "bind", MS_BIND, NULL) == -1,
				"Mounting \"%s\" on \"%s\" failed", from, to);
	}

	exit_if(snprintf(to, sizeof(to), "%s/proc", root) >= sizeof(to),
		"Path too long: \"%s\"", to);
	pexit_if(mount("/proc", to, "bind", MS_BIND, NULL) == -1,
			"Mounting /proc on \"%s\" failed", to);
	return EXIT_SUCCESS;
}
