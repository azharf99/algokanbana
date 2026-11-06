package algokanbana

import (
	"embed"
	"io/fs"
)

//go:embed all:web/dist
var rawWebFS embed.FS

func WebDistFS() fs.FS {
	sub, err := fs.Sub(rawWebFS, "web/dist")
	if err != nil {
		return nil
	}
	return sub
}
