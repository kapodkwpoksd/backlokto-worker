package targets

import (
	"path"
	"strconv"
	"strings"
	"time"
)

func createFilename(oldName string) string {
	filenameSlice := strings.Split(path.Base(oldName), ".")
	return filenameSlice[0] + "_" + strconv.FormatInt(time.Now().Unix(), 10) + ".sql"
}
