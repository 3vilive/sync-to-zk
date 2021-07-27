package main

import (
	"path"
	"strings"
)

func GetParentPath(p string) string {
	parentPath, _ := path.Split(p)
	if parentPath == "/" {
		return parentPath
	}

	return strings.TrimRight(parentPath, "/")
}
