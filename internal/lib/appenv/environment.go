package appenv

import "fmt"

type Environment string

const (
	EnvironmentDevelopment Environment = "development"
	EnvironmentTest        Environment = "test"
	EnvironmentStaging     Environment = "staging"
	EnvironmentProduction  Environment = "production"
)

func IsValidEnvironment(environment Environment) bool {
	switch environment {
	case EnvironmentDevelopment, EnvironmentTest, EnvironmentStaging, EnvironmentProduction:
		return true
	default:
		return false
	}
}

func (e Environment) IsValid() bool {
	return IsValidEnvironment(e)
}

func IsDevelopment(environment Environment) bool {
	return environment == EnvironmentDevelopment
}

func (e Environment) IsDevelopment() bool {
	return IsDevelopment(e)
}

func IsTest(environment Environment) bool {
	return environment == EnvironmentTest
}

func (e Environment) IsTest() bool {
	return IsTest(e)
}

func IsDevOrTest(environment Environment) bool {
	return IsDevelopment(environment) || IsTest(environment)
}

func (e Environment) IsDevOrTest() bool {
	return IsDevOrTest(e)
}

func IsProduction(environment Environment) bool {
	return environment == EnvironmentProduction
}

func (e Environment) IsProduction() bool {
	return IsProduction(e)
}

func (e *Environment) UnmarshalText(text []byte) error {
	value := Environment(string(text))
	if !IsValidEnvironment(value) {
		return fmt.Errorf("invalid environment %q", value)
	}

	*e = value
	return nil
}
