package mouse

import (
	"fmt"
)

type ExperimentalButtons struct {
	Mask uint8

	Left        bool
	Right       bool
	Middle      bool
	SideBack    bool
	SideForward bool

	StateA     uint8
	StateB     uint8
	EventState uint8
}

type ExperimentalMotion struct {
	MoveX int8
	MoveY int8

	EventLatch  uint8
	EventGroup  uint8
	EventStateA uint8
	EventStateB uint8
	EventStateC uint8
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
	stateA, err := readReg(RegExpB1ButtonsStateA)
	if err != nil {
		return ExperimentalStatus{}, err
	}
	stateB, err := readReg(RegExpB1ButtonsStateB)
	if err != nil {
		return ExperimentalStatus{}, err
	}
	eventState, err := readReg(RegExpB1EventState)
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
	eventLatch, err := readReg(RegExpB0EventLatch)
	if err != nil {
		return ExperimentalStatus{}, err
	}
	eventGroup, err := readReg(RegExpB0EventGroup)
	if err != nil {
		return ExperimentalStatus{}, err
	}
	eventStateA0, err := readReg(RegExpB0EventStateA)
	if err != nil {
		return ExperimentalStatus{}, err
	}
	eventStateB0, err := readReg(RegExpB0EventStateB)
	if err != nil {
		return ExperimentalStatus{}, err
	}
	eventStateC0, err := readReg(RegExpB0EventStateC)
	if err != nil {
		return ExperimentalStatus{}, err
	}

	left, right, middle, sideBack, sideForward := decodeButtonMask(mask)

	return ExperimentalStatus{
		Buttons: ExperimentalButtons{
			Mask:        mask,
			Left:        left,
			Right:       right,
			Middle:      middle,
			SideBack:    sideBack,
			SideForward: sideForward,
			StateA:      stateA,
			StateB:      stateB,
			EventState:  eventState,
		},
		Motion: ExperimentalMotion{
			MoveX:       int8(moveX),
			MoveY:       int8(moveY),
			EventLatch:  eventLatch,
			EventGroup:  eventGroup,
			EventStateA: eventStateA0,
			EventStateB: eventStateB0,
			EventStateC: eventStateC0,
		},
	}, nil
}
