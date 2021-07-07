package launcher

import (
	"fmt"
	"time"

	"github.com/briandowns/spinner"
)

// MaybeSpinner is a spinner that can also just log text
type MaybeSpinner struct {
	Spin    bool
	Spinner *spinner.Spinner
	Msg     string
}

// Start might start the spinner
func (m *MaybeSpinner) Start() {
	if m.Spin {
		m.Spinner.Start()
	} else if m.Msg != "nil" {
		fmt.Println(m.Msg)
	}
}

// Stop will stop the spinner
func (m *MaybeSpinner) Stop() {
	m.Spinner.Stop()
}

// Update will update the spinner text
func (m *MaybeSpinner) Update(t string) {
	m.Spinner.Suffix = " " + t

	if !m.Spin {
		fmt.Println(t)
	}
}

// NewMaybeSpinner will return a new MaybeSpinner
func NewMaybeSpinner(spin bool) *MaybeSpinner {
	s := &MaybeSpinner{
		Spin:    spin,
		Spinner: spinner.New(spinner.CharSets[9], 300*time.Millisecond),
		Msg:     "",
	}
	s.Spinner.Prefix = " "
	return s
}
