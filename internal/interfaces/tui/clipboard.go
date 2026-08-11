package tui

import (
	"encoding/base64"

	"golang.design/x/clipboard"
)

func init() {
	clipboard.Init()
}

func getImageFromClipboard() (string, bool) {
	data := clipboard.Read(clipboard.FmtImage)
	if len(data) == 0 {
		return "", false
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(data), true
}
