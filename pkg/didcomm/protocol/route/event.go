package route

type forwardEvent struct {
	theirDID string
}

func (r *forwardEvent) TheirDID() string {
	return r.theirDID
}
