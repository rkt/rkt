package tarheader

import (
	"archive/tar"
	"os"
)

var populateHeaderStat []func(h *tar.Header, fi os.FileInfo)

func Populate(h *tar.Header, fi os.FileInfo) {
	for _, pop := range populateHeaderStat {
		pop(h, fi)
	}
}
