package configure_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/SafetyCulture/mitti-exporter/cmd/mitti-exporter/cmd"
)

func TestCommandConfigure_should_not_throw_error(t *testing.T) {
	b := bytes.NewBufferString("")
	cmd.RootCmd.SetOut(b)
	cmd.RootCmd.SetArgs([]string{"configure"})
	cmd.Execute()
	_, err := io.ReadAll(b)
	if err != nil {
		t.Fatal(err)
	}
}
