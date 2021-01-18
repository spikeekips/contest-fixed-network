package host

import (
	"context"
	"fmt"

	"github.com/spikeekips/contest/config"
)

type Sequence struct {
	condition *Condition
	action    Action
	register  config.DesignRegister
}

func NewSequence(condition *Condition, action Action, register config.DesignRegister) (*Sequence, error) {
	sq := &Sequence{
		condition: condition,
		action:    action,
		register:  register,
	}

	return sq, nil
}

func (sq *Sequence) Action() Action {
	return sq.action
}

func (sq *Sequence) Register() config.DesignRegister {
	return sq.register
}

func (sq *Sequence) SetRegister(vars *config.Vars, record interface{}) {
	if sq.Register().IsEmpty() {
		return
	}

	vars.Set(fmt.Sprintf("Register.%s", sq.Register().To), record)
}

func (sq *Sequence) Condition() *Condition {
	return sq.condition
}

type Action interface {
	Name() string
	Run(context.Context) error
}

type NullAction struct{}

func (ac NullAction) Name() string {
	return "null-action"
}

func (ac NullAction) Run(context.Context) error {
	return nil
}
