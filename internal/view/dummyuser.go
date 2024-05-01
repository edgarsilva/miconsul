package view

type DummyUser struct{}

func (du DummyUser) IsLoggedIn() bool { return true }
func (du DummyUser) ID() uint         { return 0 }
func (du DummyUser) UID() string      { return "" }
func (du DummyUser) Email() string    { return "" }
func (du DummyUser) JWT() string      { return "" }
