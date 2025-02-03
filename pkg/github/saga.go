package github

import "go.temporal.io/sdk/workflow"

type Saga struct {
	toUnwind  []any
	arguments [][]any
}

func (s *Saga) Add(activity any, parameters ...any) {
	s.toUnwind = append(s.toUnwind, activity)
	s.arguments = append(s.arguments, parameters)
}

func (s Saga) Unwind(ctx workflow.Context) {
	for i := len(s.toUnwind) - 1; i >= 0; i-- {
		errCompensation := workflow.ExecuteActivity(ctx, s.toUnwind[i], s.arguments[i]...).Get(ctx, nil)
		if errCompensation != nil {
			workflow.GetLogger(ctx).Error("Executing compensation failed", "Error", errCompensation)
		}
	}
}

func NewSaga() *Saga {
	return &Saga{}
}
