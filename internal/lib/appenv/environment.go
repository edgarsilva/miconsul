package appenv

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

func IsDevelopment(environment Environment) bool {
	return environment == EnvironmentDevelopment
}

func IsTest(environment Environment) bool {
	return environment == EnvironmentTest
}

func IsDevOrTest(environment Environment) bool {
	return IsDevelopment(environment) || IsTest(environment)
}

func IsProduction(environment Environment) bool {
	return environment == EnvironmentProduction
}
