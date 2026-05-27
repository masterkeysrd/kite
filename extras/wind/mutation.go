package wind

import (
	"context"

	"github.com/masterkeysrd/kite/extras/kitex"
)

type MutationContext struct {
	Client *Client
}

type MutationOptions[V any, R any] struct {
	OnSuccess func(R, V, MutationContext)
	OnError   func(error, V, MutationContext)
}

type MutationResult[V any] struct {
	IsPending bool
	IsError   bool
	Error     error
	Mutate    func(V)
}

type mutationState struct {
	isPending bool
	isError   bool
	err       error
}

func UseMutation[V any, R any](
	mutationFn func(context.Context, V) (R, error),
	opts ...MutationOptions[V, R],
) MutationResult[V] {
	client := UseClient()

	getMutState, setMutState := kitex.UseState(mutationState{
		isPending: false,
		isError:   false,
		err:       nil,
	})

	mutate := func(variables V) {
		setMutState(mutationState{
			isPending: true,
			isError:   false,
			err:       nil,
		})

		go func() {
			res, err := mutationFn(context.Background(), variables)

			if err != nil {
				setMutState(mutationState{
					isPending: false,
					isError:   true,
					err:       err,
				})
				mCtx := MutationContext{Client: client}
				for _, opt := range opts {
					if opt.OnError != nil {
						opt.OnError(err, variables, mCtx)
					}
				}
			} else {
				setMutState(mutationState{
					isPending: false,
					isError:   false,
					err:       nil,
				})
				mCtx := MutationContext{Client: client}
				for _, opt := range opts {
					if opt.OnSuccess != nil {
						opt.OnSuccess(res, variables, mCtx)
					}
				}
			}
		}()
	}

	s := getMutState()
	return MutationResult[V]{
		IsPending: s.isPending,
		IsError:   s.isError,
		Error:     s.err,
		Mutate:    mutate,
	}
}
