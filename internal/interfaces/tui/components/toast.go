package components

type Toast struct {
	Message string
	IsError bool
	Visible bool
}

func NewToast() Toast {
	return Toast{}
}

func (t *Toast) Show(message string, isError bool) {
	t.Message = message
	t.IsError = isError
	t.Visible = true
}

func (t *Toast) Hide() {
	t.Visible = false
}
