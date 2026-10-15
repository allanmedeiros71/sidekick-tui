package inspect

import (
	"encoding/base64"
	"os"

	"github.com/atotto/clipboard"
)

// copyToClipboard puts s on the system clipboard (pbcopy, wl-copy, xclip or
// xsel). Where none is available, as over SSH, it falls back to an OSC 52
// escape sequence, which most terminals and tmux (with set-clipboard on)
// turn into a clipboard write.
func copyToClipboard(s string) error {
	if err := clipboard.WriteAll(s); err == nil {
		return nil
	}
	_, err := os.Stdout.WriteString("\x1b]52;c;" + base64.StdEncoding.EncodeToString([]byte(s)) + "\a")
	return err
}
