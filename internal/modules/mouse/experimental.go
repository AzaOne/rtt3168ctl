package mouse

import (
	"fmt"
)

type ExperimentalButtons struct {
	Mask          uint8
	MaskMirror    uint8
	EffectiveMask uint8

	Left        bool
	Right       bool
	Middle      bool
	SideBack    bool
	SideForward bool

	StateA       uint8
	StateAMirror uint8
	StateB       uint8
	StateBMirror uint8
	EventState   uint8
	EventStateM  uint8
}

type ExperimentalMotion struct {
	MoveX       int8
	MoveY       int8
	MoveXMirror int8
	MoveYMirror int8

	EventLatch       uint8
	EventLatchMirror uint8
	EventGroup       uint8
	EventGroupMirror uint8
	EventStateA      uint8
	EventStateAM     uint8
	EventStateB      uint8
	EventStateBM     uint8
	EventStateC      uint8
	EventStateCM     uint8
}

type ExperimentalStatus struct {
	Buttons ExperimentalButtons
	Motion  ExperimentalMotion
}

func (s *Service) ReadExperimentalStatus() (ExperimentalStatus, error) {
	return s.readExperimentalStatus()
}

func (s *Service) readExperimentalStatus() (ExperimentalStatus, error) {
	readReg := func(reg uint16) (uint8, error) {
		val, err := s.repo.ReadRegister(reg)
		if err != nil {
			return 0, fmt.Errorf("reg 0x%02X: %w", reg, err)
		}
		return val, nil
	}

	if err := s.enterBank1(); err != nil {
		return ExperimentalStatus{}, fmt.Errorf("enter bank1: %w", err)
	}

	mask, err := readReg(RegExpB1ButtonsMask)
	if err != nil {
		return ExperimentalStatus{}, err
	}
	maskMirror, err := readReg(RegExpB1ButtonsMaskMirror)
	if err != nil {
		return ExperimentalStatus{}, err
	}
	stateA, err := readReg(RegExpB1ButtonsStateA)
	if err != nil {
		return ExperimentalStatus{}, err
	}
	stateAMirror, err := readReg(RegExpB1ButtonsStateAMirr)
	if err != nil {
		return ExperimentalStatus{}, err
	}
	stateB, err := readReg(RegExpB1ButtonsStateB)
	if err != nil {
		return ExperimentalStatus{}, err
	}
	stateBMirror, err := readReg(RegExpB1ButtonsStateBMirr)
	if err != nil {
		return ExperimentalStatus{}, err
	}
	eventState, err := readReg(RegExpB1EventState)
	if err != nil {
		return ExperimentalStatus{}, err
	}
	eventStateMirror, err := readReg(RegExpB1EventStateMirror)
	if err != nil {
		return ExperimentalStatus{}, err
	}

	if err := s.enterBank0(); err != nil {
		return ExperimentalStatus{}, fmt.Errorf("enter bank0: %w", err)
	}

	moveX, err := readReg(RegExpB0MoveX)
	if err != nil {
		return ExperimentalStatus{}, err
	}
	moveY, err := readReg(RegExpB0MoveY)
	if err != nil {
		return ExperimentalStatus{}, err
	}
	moveXMirror, err := readReg(RegExpB0MoveXMirror)
	if err != nil {
		return ExperimentalStatus{}, err
	}
	moveYMirror, err := readReg(RegExpB0MoveYMirror)
	if err != nil {
		return ExperimentalStatus{}, err
	}
	eventLatch, err := readReg(RegExpB0EventLatch)
	if err != nil {
		return ExperimentalStatus{}, err
	}
	eventLatchMirror, err := readReg(RegExpB0EventLatchMirror)
	if err != nil {
		return ExperimentalStatus{}, err
	}
	eventGroup, err := readReg(RegExpB0EventGroup)
	if err != nil {
		return ExperimentalStatus{}, err
	}
	eventGroupMirror, err := readReg(RegExpB0EventGroupMirror)
	if err != nil {
		return ExperimentalStatus{}, err
	}
	eventStateA0, err := readReg(RegExpB0EventStateA)
	if err != nil {
		return ExperimentalStatus{}, err
	}
	eventStateAM0, err := readReg(RegExpB0EventStateAMirr)
	if err != nil {
		return ExperimentalStatus{}, err
	}
	eventStateB0, err := readReg(RegExpB0EventStateB)
	if err != nil {
		return ExperimentalStatus{}, err
	}
	eventStateBM0, err := readReg(RegExpB0EventStateBMirr)
	if err != nil {
		return ExperimentalStatus{}, err
	}
	eventStateC0, err := readReg(RegExpB0EventStateC)
	if err != nil {
		return ExperimentalStatus{}, err
	}
	eventStateCM0, err := readReg(RegExpB0EventStateCMirr)
	if err != nil {
		return ExperimentalStatus{}, err
	}

	effectiveMask := mask | maskMirror
	left, right, middle, sideBack, sideForward := decodeButtonMask(effectiveMask)

	return ExperimentalStatus{
		Buttons: ExperimentalButtons{
			Mask:          mask,
			MaskMirror:    maskMirror,
			EffectiveMask: effectiveMask,
			Left:          left,
			Right:         right,
			Middle:        middle,
			SideBack:      sideBack,
			SideForward:   sideForward,
			StateA:        stateA,
			StateAMirror:  stateAMirror,
			StateB:        stateB,
			StateBMirror:  stateBMirror,
			EventState:    eventState,
			EventStateM:   eventStateMirror,
		},
		Motion: ExperimentalMotion{
			MoveX:            int8(moveX),
			MoveY:            int8(moveY),
			MoveXMirror:      int8(moveXMirror),
			MoveYMirror:      int8(moveYMirror),
			EventLatch:       eventLatch,
			EventLatchMirror: eventLatchMirror,
			EventGroup:       eventGroup,
			EventGroupMirror: eventGroupMirror,
			EventStateA:      eventStateA0,
			EventStateAM:     eventStateAM0,
			EventStateB:      eventStateB0,
			EventStateBM:     eventStateBM0,
			EventStateC:      eventStateC0,
			EventStateCM:     eventStateCM0,
		},
	}, nil
}
