package main

import (
	"reflect"
	"testing"
)

func TestFSM(t *testing.T) {
	testArg := []interface{}{
		"argument",
	}
	testCases := []struct {
		name                 string
		plan                 []string
		dirs                 []string
		testCallbacks        testCallback
		expectedBeforeEvents []Event
		expectedEnterEvents  []Event
	}{
		{
			name: "nominal",
			plan: []string{
				"#####",
				"#$ X#",
				"# @B#",
				"#####",
			},
			testCallbacks: newCallbackRecorder(),
			dirs: []string{
				EAST,
				NORTH,
				WEST,
				WEST,
			},
			expectedBeforeEvents: []Event{
				Event{Event: EAST, Dst: 'B', dstC: Pair{3, 2}, Args: testArg},
				Event{Event: NORTH, Dst: 'X', dstC: Pair{3, 1}, Args: testArg},
				Event{Event: WEST, Dst: ' ', dstC: Pair{2, 1}, Args: testArg},
				Event{Event: WEST, Dst: '$', dstC: Pair{1, 1}, Args: testArg},
			},
			expectedEnterEvents: []Event{
				Event{Event: EAST, Dst: 'B', dstC: Pair{3, 2}, Args: testArg},
				Event{Event: NORTH, Dst: 'X', dstC: Pair{3, 1}, Args: testArg},
				Event{Event: WEST, Dst: ' ', dstC: Pair{2, 1}, Args: testArg},
				Event{Event: WEST, Dst: '$', dstC: Pair{1, 1}, Args: testArg},
			},
		},
		{
			name: "cancelled",
			plan: []string{
				"#####",
				"#$  #",
				"#@ X#",
				"#####",
			},
			testCallbacks: newCallbackRecorderCancel(2),
			dirs: []string{
				EAST,
				EAST,
				NORTH,
				WEST,
			},
			expectedBeforeEvents: []Event{
				Event{Event: EAST, Dst: ' ', dstC: Pair{2, 2}, Args: testArg},
				Event{Event: EAST, Dst: 'X', dstC: Pair{3, 2}, Args: testArg},
				Event{Event: NORTH, Dst: ' ', dstC: Pair{2, 1}, Args: testArg},
				Event{Event: WEST, Dst: '$', dstC: Pair{1, 1}, Args: testArg},
			},
			expectedEnterEvents: []Event{
				Event{Event: EAST, Dst: ' ', dstC: Pair{2, 2}, Args: testArg},
				Event{Event: NORTH, Dst: ' ', dstC: Pair{2, 1}, Args: testArg},
				Event{Event: WEST, Dst: '$', dstC: Pair{1, 1}, Args: testArg},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fsm := NewFSM(tc.plan, tc.testCallbacks.before, tc.testCallbacks.enter)
			for _, d := range tc.dirs {
				fsm.Event(d, testArg...)
			}

			for i, act := range tc.testCallbacks.beforeStack() {
				exp := tc.expectedBeforeEvents[i]
				if !eventEqual(exp, act, fsm) {
					t.Errorf("Test case %q: event %v doesn't match expected %v", tc.name, act, exp)
				}
			}
			for i, act := range tc.testCallbacks.enterStack() {
				exp := tc.expectedEnterEvents[i]
				if !eventEqual(exp, act, fsm) {
					t.Errorf("Test case %q: event %v doesn't match expected %v", tc.name, act, exp)
				}
			}
		})
	}
}

func TestBenderSimulator(t *testing.T) {
	stateNum := 9
	bender := NewBenderSimulator(stateNum)

	// start from the first priority
	dir := bender.Direction()
	if dir != SOUTH {
		t.Fatalf("Wrong priority direction. Expected %s, got %s", SOUTH, dir)
	}
	// must continue the same direction if no path modifier or next direction
	dir = bender.Direction()
	if dir != SOUTH {
		t.Fatalf("Wrong continuation of priority direction. Expected %s, got %s", SOUTH, dir)
	}
	// must choose the next priority direction
	bender.NextDirection()
	dir = bender.Direction()
	if dir != EAST {
		t.Fatalf("Wrong next priority direction. Expected %s, got %s", EAST, dir)
	}
	// must get back to the first priority
	bender.NextDirection()
	bender.NextDirection()
	bender.NextDirection()
	dir = bender.Direction()
	if dir != SOUTH {
		t.Fatalf("No cycle in priority direction. Expected %s, got %s", SOUTH, dir)
	}
	// must stick with the path modifier
	bender.PathModifier(NORTH)
	dir = bender.Direction()
	if dir != NORTH {
		t.Fatalf("Wrong path modifier. Expected %s, got %s", NORTH, dir)
	}
	// obstacle case, must get back to the priorities
	bender.Boom()
	dir = bender.Direction()
	if dir != SOUTH {
		t.Fatalf("Failed to get back to priorities. Expected %s, got %s", SOUTH, dir)
	}
	// looking for a way out of the obstacles
	bender.NextDirection()
	if !bender.Hurts() {
		t.Fatalf("Obstacle is not recorded")
	}
	if bender.Hurts() {
		bender.BackOnTrack()
	}
	bender.NextDirection()
	dir = bender.Direction()
	if dir != SOUTH {
		t.Fatalf("Priorities not reset. Expected %s, got %s", SOUTH, dir)
	}
	// invert priorities
	bender.InvertPriorities()
	bender.Boom()
	bender.NextDirection()
	dir = bender.Direction()
	if dir != WEST {
		t.Fatalf("Failed to invert priorities. Expected %s, got %s", WEST, dir)
	}
	bender.NextDirection()
	bender.NextDirection()
	dir = bender.Direction()
	if dir != EAST {
		t.Fatalf("Failed to continue on inverted priorities. Expected %s, got %s", EAST, dir)
	}
	bender.InvertPriorities()
	bender.Boom()
	bender.NextDirection()
	dir = bender.Direction()
	if dir != SOUTH {
		t.Fatalf("Failed to invert back the priorities. Expected %s, got %s", SOUTH, dir)
	}
	// path
	dirs := []string{
		SOUTH,
		SOUTH,
		EAST,
		EAST,
	}
	bender.Remember(dirs[0], " 11")
	bender.Remember(dirs[1], " 12")
	bender.Remember(dirs[2], " 22")
	bender.Remember(dirs[3], "B32")
	for i, p := range bender.ShowPath() {
		if dirs[i] != p {
			t.Fatalf("Wrong path. Expected %s, got %s", dirs[i], p)
		}
	}
	bender.Remember(dirs[0], " 11")
	bender.Remember(dirs[1], " 12")
	bender.Remember(dirs[2], " 22")
	bender.Remember(dirs[3], "B32")
	bender.Remember(dirs[0], " 11")
	bender.Remember(dirs[1], " 12")
	bender.Remember(dirs[2], " 22")
	bender.Remember(dirs[3], "B32")
	if bender.Loop() {
		t.Fatalf("False positive loop detection")
	}
	bender.Remember(dirs[0], " 11")
	bender.Remember(dirs[1], " 12")
	bender.Remember(dirs[2], " 22")
	bender.Remember(dirs[3], "B32")
	if !bender.Loop() {
		t.Fatalf("Loop was not detected")
	}
	// breaker mode
	br := bender.Breaker()
	bender.InvertBreaker()
	ibr := bender.Breaker()
	if br != !ibr {
		t.Fatalf("Failed to invert the breaker mode #1")
	}
	bender.InvertBreaker()
	ibr = bender.Breaker()
	if br != ibr {
		t.Fatalf("Failed to invert the breaker mode #2")
	}
	// booth reached
	bender.Reached()
	if !bender.Done() {
		t.Fatalf("Failed to become done")
	}
}

func TestCalcNumStates(t *testing.T) {
	plan := []string{
		"#####",
		"#   #",
		"#   #",
		"#   #",
		"#####",
	}
	num := calcNumStates(plan)
	if num != 9 {
		t.Fatalf("Wrong number of valid states. Expected %d, got %d.", 9, num)
	}
	plan = []string{
		"#####",
		"#   #",
		"#   #",
		"#####",
	}
	num = calcNumStates(plan)
	if num != 6 {
		t.Fatalf("Wrong number of valid states. Expected %d, got %d.", 6, num)
	}
}

type testCallback interface {
	before(*Event)
	enter(*Event)
	beforeStack() []Event
	enterStack() []Event
}

type callbackRecorder struct {
	bStack []Event
	eStack []Event
}

func newCallbackRecorder() *callbackRecorder {
	return &callbackRecorder{
		bStack: []Event{},
		eStack: []Event{},
	}
}

func (c *callbackRecorder) before(e *Event) {
	c.bStack = append(c.bStack, *e)
}

func (c *callbackRecorder) enter(e *Event) {
	c.eStack = append(c.eStack, *e)
}

func (c *callbackRecorder) beforeStack() []Event {
	return c.bStack
}

func (c *callbackRecorder) enterStack() []Event {
	return c.eStack
}

type callbackRecorderCancel struct {
	bStack    []Event
	eStack    []Event
	cancelIdx int
	beforeCnt int
}

func newCallbackRecorderCancel(idx int) *callbackRecorderCancel {
	return &callbackRecorderCancel{
		bStack:    []Event{},
		eStack:    []Event{},
		cancelIdx: idx,
	}
}

func (c *callbackRecorderCancel) before(e *Event) {
	c.bStack = append(c.bStack, *e)
	c.beforeCnt++
	if c.cancelIdx == c.beforeCnt {
		e.Cancel()
	}
}

func (c *callbackRecorderCancel) enter(e *Event) {
	c.eStack = append(c.eStack, *e)
}

func (c *callbackRecorderCancel) beforeStack() []Event {
	return c.bStack
}

func (c *callbackRecorderCancel) enterStack() []Event {
	return c.eStack
}

func eventEqual(exp, act Event, fsm *FSM) bool {
	if act.FSM != fsm {
		return false
	}
	if exp.Event != act.Event {
		return false
	}
	if exp.Dst != act.Dst {
		return false
	}
	if exp.dstC != act.dstC {
		return false
	}
	if !reflect.DeepEqual(exp.Args, act.Args) {
		return false
	}
	return true
}
