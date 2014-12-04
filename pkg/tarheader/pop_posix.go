package tarheader

import (
	"archive/tar"
	"os"
	"syscall"
)

func init() {
	populateHeaderStat = append(populateHeaderStat, populateHeaderUnix)
}

func populateHeaderUnix(h *tar.Header, fi os.FileInfo) {
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return
	}
	h.Uid = int(st.Uid)
	h.Gid = int(st.Gid)
}
