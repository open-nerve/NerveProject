package domain

// StateGroup is a state's group, a value of states."group" (M3 design 4.9,
// Plane's StateGroup, db/models/state.py:14-20).
type StateGroup string

// The groups. The triage group holds the intake's state only (M3 design
// 3.17).
const (
	GroupBacklog   StateGroup = "backlog"
	GroupUnstarted StateGroup = "unstarted"
	GroupStarted   StateGroup = "started"
	GroupCompleted StateGroup = "completed"
	GroupCancelled StateGroup = "cancelled"
	GroupTriage    StateGroup = "triage"
)

// NewState is a state to create: its values (M3 design 3.17).
type NewState struct {
	Name     string
	Color    string
	Sequence float64
	Group    StateGroup
	Default  bool
}

// DefaultStates are the six states a new project has, in their order:
// Plane's DEFAULT_STATES (db/models/state.py:24-62), Backlog the default,
// Triage the triage state (M3 design 3.17).
func DefaultStates() []NewState {
	return []NewState{
		{Name: "Backlog", Color: "#60646C", Sequence: 15000, Group: GroupBacklog, Default: true},
		{Name: "Todo", Color: "#60646C", Sequence: 25000, Group: GroupUnstarted},
		{Name: "In Progress", Color: "#F59E0B", Sequence: 35000, Group: GroupStarted},
		{Name: "Done", Color: "#46A758", Sequence: 45000, Group: GroupCompleted},
		{Name: "Cancelled", Color: "#9AA4BC", Sequence: 55000, Group: GroupCancelled},
		{Name: "Triage", Color: "#4E5355", Sequence: 65000, Group: GroupTriage},
	}
}
