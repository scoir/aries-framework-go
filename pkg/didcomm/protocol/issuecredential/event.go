package issuecredential

type credentialEvent struct {
	theirDID string
}

func (r *credentialEvent) TheirDID() string {
	return r.theirDID
}
