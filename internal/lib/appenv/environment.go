package appenv

import "fmt"

type Environment string

const (
	EnvironmentDevelopment Environment = "development"
	EnvironmentTest        Environment = "test"
	EnvironmentStaging     Environment = "staging"
	EnvironmentProduction  Environment = "production"
)

func (e Environment) IsValid() bool {
	switch e {
	case EnvironmentDevelopment, EnvironmentTest, EnvironmentStaging, EnvironmentProduction:
		return true
	default:
		return false
	}
}

func (e Environment) IsDevelopment() bool {
	return e == EnvironmentDevelopment
}

func (e Environment) IsTest() bool {
	return e == EnvironmentTest
}

func (e Environment) IsDevOrTest() bool {
	return e.IsDevelopment() || e.IsTest()
}

func (e Environment) IsProduction() bool {
	return e == EnvironmentProduction
}

func (e *Environment) UnmarshalText(text []byte) error {
	value := Environment(string(text))
	if !value.IsValid() {
		return fmt.Errorf("invalid environment %q", value)
	}

	*e = value
	return nil
}
