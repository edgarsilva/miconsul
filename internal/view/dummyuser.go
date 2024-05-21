package view

type DummyUser struct{}

func (du DummyUser) IsLoggedIn() bool { return false }
func (du DummyUser) ID() uint         { return 0 }
func (du DummyUser) Email() string    { return "" }
