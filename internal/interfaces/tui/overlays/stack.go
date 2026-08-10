package overlays

type OverlayType int

const (
	OverlayNone OverlayType = iota
	OverlayApproval
	OverlayHelp
	OverlayCommand
	OverlaySession
	OverlayToast
	OverlaySkills
	OverlayMCP
	OverlayMemory
	OverlayWorkspace
	OverlayNewSession
	OverlayRename
	OverlayRollback
	OverlayStatus
	OverlayVersion
	OverlayBackground
	OverlayDashboard
	OverlaySettings
	OverlayReasoning
)

type OverlayStack struct {
	stack   []OverlayType
	Current OverlayType
}

func (os *OverlayStack) Open(t OverlayType) {
	os.stack = append(os.stack, t)
	os.Current = t
}

func (os *OverlayStack) Close() bool {
	if len(os.stack) == 0 {
		os.Current = OverlayNone
		return false
	}
	os.stack = os.stack[:len(os.stack)-1]
	if len(os.stack) == 0 {
		os.Current = OverlayNone
	} else {
		os.Current = os.stack[len(os.stack)-1]
	}
	return true
}

func (os *OverlayStack) IsOpen() bool {
	return os.Current != OverlayNone
}
