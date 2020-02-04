package main

import (
	"fmt"
)

const (
	// SOUTH direction
	SOUTH = "SOUTH"
	// NORTH direction
	NORTH = "NORTH"
	// EAST direction
	EAST = "EAST"
	// WEST direction
	WEST = "WEST"
	// LOOP indicator
	LOOP = "LOOP"
)

// BenderSimulator simulates more rudimentary Bender
type BenderSimulator struct {
	done         bool
	breaker      bool
	boom         bool
	resetDir     bool
	invertPrio   bool
	currDir      int
	priorities   []string
	pathModifier string
	path         []string
	cache        map[string]bool
	loopCnt      int
	maxNumStates int
}

// NewBenderSimulator returns an instance of a bender simulator
// the number of valid (without the frame) states is expected as parameter
func NewBenderSimulator(stateNum int) *BenderSimulator {
	return &BenderSimulator{
		priorities: []string{
			SOUTH,
			EAST,
			NORTH,
			WEST,
		},
		path:         []string{},
		cache:        map[string]bool{},
		maxNumStates: stateNum,
	}
}

// Done returns true if the suicide booth is reached
func (b *BenderSimulator) Done() bool {
	return b.done
}

// Loop returns true if an endless cycle is found
func (b *BenderSimulator) Loop() bool {
	if b.loopCnt > b.maxNumStates {
		return true
	}
	return false
}

// Direction gives the direction to be followed
func (b *BenderSimulator) Direction() string {
	if b.pathModifier != "" {
		return b.pathModifier
	}
	return b.priorities[b.currDir]
}

// ShowPath returns the recorded path
func (b *BenderSimulator) ShowPath() []string {
	if b.Loop() {
		return []string{LOOP}
	}
	return b.path
}

// Breaker returns true if the simulator went to the breaker mode
func (b *BenderSimulator) Breaker() bool {
	return b.breaker
}

// InvertBreaker inverts the breaker mode
func (b *BenderSimulator) InvertBreaker() {
	if b.breaker {
		b.breaker = false
		return
	}
	b.breaker = true
}

// Reached signals that the suicide booth is reached
func (b *BenderSimulator) Reached() {
	b.done = true
}

// InvertPriorities signals that the priorities needs to be inverted
// when next obstacle is reached
func (b *BenderSimulator) InvertPriorities() {
	if b.invertPrio {
		b.invertPrio = false
		return
	}
	b.invertPrio = true
}

// turnoverPriorities turn the list of priorities up side down
func (b *BenderSimulator) turnoverPriorities() {
	for i, j := 0, len(b.priorities)-1; i < len(b.priorities)/2; i, j = i+1, j-1 {
		b.priorities[i], b.priorities[j] = b.priorities[j], b.priorities[i]
	}
	b.invertPrio = false
}

// Remember records the given direction and the state
// of course, they are supposed to be passed and visited
func (b *BenderSimulator) Remember(dir, state string) {
	b.path = append(b.path, dir)
	if _, exist := b.cache[state]; exist {
		// already visited this state: increment the loop counter
		b.loopCnt++
	} else {
		// unknown state: reset the loop counter
		b.cache[state] = true
		b.loopCnt = 0
	}
}

// PathModifier unsets the priority directions with the given one
func (b *BenderSimulator) PathModifier(dir string) {
	b.pathModifier = dir
}

// NextDirection calculates the next direction to be given after an obstacle is hit
func (b *BenderSimulator) NextDirection() {
	if b.resetDir {
		b.currDir = 0
		b.resetDir = false
	} else {
		if b.currDir+1 >= len(b.priorities) {
			b.currDir = 0
		} else {
			b.currDir++
		}
	}
}

// Boom signals a hit against an obstacle
func (b *BenderSimulator) Boom() {
	b.boom = true
	// back to priorities
	b.pathModifier = ""
	// turnover the priorities if passed by an inverted before
	if b.invertPrio {
		b.turnoverPriorities()
		// we need to start from the top
		b.resetDir = true
	}
}

// Hurts returns true if the simulator just hit the obstacle
func (b *BenderSimulator) Hurts() bool {
	return b.boom
}

// BackOnTrack signals that the way out of the obstacles is found
func (b *BenderSimulator) BackOnTrack() {
	b.boom = false
	b.resetDir = true
}

// Pair is a pair of coordinates
type Pair struct {
	x, y int
}

// FSM is a 2D array Finite State Machine.
// Each item in the array is a state.
// Transitions between the states are the cardinal directions.
// Example:
// [1,1] SOUTH [1,2]
// [1,1] NORTH [1,0]
// [1,1] EAST  [2,1]
// [1,1] WEST  [0,1]
type FSM struct {
	states         [][]byte
	curr           Pair
	teleports      []Pair
	beforeCallback Callback
	enterCallback  Callback
}

// NewFSM returns an instance of FSM from given map
// before callback is called when the state is not yet entered
// enter callback is called when the state is already entered
func NewFSM(plan []string, beforeCB, enterCB Callback) *FSM {
	states := make([][]byte, 0, len(plan))
	start := Pair{}
	tp := []Pair{}

	for i, s := range plan {
		states = append(states, []byte(s))
		for j, c := range s {
			if len(tp) == 2 && (start != Pair{}) {
				break
			}
			switch c {
			case '@':
				start = Pair{j, i}
			case 'T':
				tp = append(tp, Pair{j, i})
			}
		}
	}

	return &FSM{
		states:         states,
		curr:           start,
		teleports:      tp,
		beforeCallback: beforeCB,
		enterCallback:  enterCB,
	}
}

// Event changes the state according to the direction given
// runs the before and enter callbacks passing the given arguments to them
func (f *FSM) Event(evt string, args ...interface{}) error {
	var dst Pair
	switch evt {
	case SOUTH:
		dst = Pair{f.curr.x, f.curr.y + 1}
	case NORTH:
		dst = Pair{f.curr.x, f.curr.y - 1}
	case EAST:
		dst = Pair{f.curr.x + 1, f.curr.y}
	case WEST:
		dst = Pair{f.curr.x - 1, f.curr.y}
	}

	if dst.x < 0 || dst.x >= len(f.states[0]) || dst.y < 0 || dst.y >= len(f.states) {
		return fmt.Errorf("unknown state %v", dst)
	}

	e := &Event{
		FSM:   f,
		Event: evt,
		Dst:   f.states[dst.y][dst.x],
		dstC:  dst,
		Args:  args,
	}

	f.beforeCallback(e)
	if e.Cancelled {
		// don't enter the state
		return nil
	}
	f.curr = dst
	f.enterCallback(e)
	return nil
}

// SetState sets the current state of the machine
func (f *FSM) SetState(p Pair) {
	f.curr = p
}

// TeleportDst gives the destination coordinates of the given teleport
func (f *FSM) TeleportDst(ps Pair) Pair {
	if len(f.teleports) != 2 {
		panic("teleports badly setup")
	}

	if f.teleports[0].x == ps.x && f.teleports[0].y == ps.y {
		return f.teleports[1]
	}
	return f.teleports[0]
}

// Callback type to handle state actions
type Callback func(e *Event)

// Event represents the transition event
type Event struct {
	// pointer back to the finite state machine
	FSM *FSM
	// name of the event (direction)
	Event string
	// destination state
	Dst byte
	// destination state's coordinates
	dstC Pair
	// true if event was cancelled
	Cancelled bool
	// arguments for the callbacks
	Args []interface{}
}

// Cancel cancels the event.
// Events cancelled before entering the state will not be entered.
func (e *Event) Cancel() {
	e.Cancelled = true
}

// ChangeDst sets the destination state with the given value
func (e *Event) ChangeDst(dst byte) {
	e.FSM.states[e.dstC.y][e.dstC.x] = dst
}

// UniqueDst generates the unique destination id (value+coordinates)
func (e *Event) UniqueDst() string {
	return fmt.Sprintf("%c%d%d", e.Dst, e.dstC.x, e.dstC.y)
}

// before handles only obstacles
// we cancel the event before entering it
func beforeCallback(e *Event) {
	bender := e.Args[0].(*BenderSimulator)

	switch e.Dst {
	case '#':
		bender.Boom()
		bender.NextDirection()
		e.Cancel()
	case 'X':
		if bender.Breaker() {
			// destroy the obstacle
			e.ChangeDst(' ')
		} else {
			bender.Boom()
			bender.NextDirection()
			e.Cancel()
		}
	}
}

// enter handles all non obstacle states
func enterCallback(e *Event) {
	bender := e.Args[0].(*BenderSimulator)

	if bender.Hurts() {
		// managed to enter the state: obstacle is behind
		bender.BackOnTrack()
	}

	switch e.Dst {
	case 'B':
		bender.InvertBreaker()
	case 'S':
		bender.PathModifier(SOUTH)
	case 'N':
		bender.PathModifier(NORTH)
	case 'E':
		bender.PathModifier(EAST)
	case 'W':
		bender.PathModifier(WEST)
	case 'I':
		bender.InvertPriorities()
	case 'T':
		e.FSM.SetState(e.FSM.TeleportDst(e.dstC))
	case '$':
		bender.Reached()
	}
	bender.Remember(e.Event, e.UniqueDst())
}

// returns the number of valid (frame excluded) states of a map
func calcNumStates(plan []string) int {
	l := len(plan[0])
	w := len(plan)
	return (w - 2) * (l - 2)
}

func main() {
	plan := []string{
		"########",
		"#     $#",
		"#      #",
		"#      #",
		"#  @   #",
		"#      #",
		"#      #",
		"########",
	}
	fmt.Println("Plan:")
	for _, s := range plan {
		fmt.Println(s)
	}

	m := NewFSM(plan, beforeCallback, enterCallback)
	bender := NewBenderSimulator(calcNumStates(plan))

	for !bender.Done() && !bender.Loop() {
		err := m.Event(bender.Direction(), bender)
		if err != nil {
			fmt.Println("Failed with error: ", err)
			return
		}
	}
	fmt.Println(bender.ShowPath())
}
