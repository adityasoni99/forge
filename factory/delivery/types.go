package delivery

type DeliveryConfig struct {
	Remote     string
	BaseBranch string
	PRTitle    string
	PRBody     string
}

type DeliveryResult struct {
	Branch    string
	PRURL     string
	Pushed    bool
	PRCreated bool
}
