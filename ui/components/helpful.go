package components

import "github.com/charmbracelet/bubbles/key"

type Helpful interface {
	Help() []key.Binding
}
