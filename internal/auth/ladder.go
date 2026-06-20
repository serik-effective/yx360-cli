package auth

import (
	"context"
	"errors"
)

type Ladder struct {
	rungs []Provider
}

func NewLadder(loopback, device Provider) *Ladder {
	return &Ladder{rungs: []Provider{loopback, device}}
}

func (l *Ladder) Authenticate(ctx context.Context, opts AuthOptions) (*Credential, error) {
	rungs := l.rungs
	if opts.PreferDevice && len(rungs) == 2 {
		rungs = []Provider{rungs[1]}
	}

	var lastErr error
	for _, rung := range rungs {
		cred, err := rung.Authenticate(ctx, opts)
		if err == nil {
			return cred, nil
		}
		if errors.Is(err, errRungUnavailable) {
			lastErr = err
			continue
		}
		return nil, err
	}
	if lastErr == nil {
		lastErr = errRungUnavailable
	}
	return nil, lastErr
}
