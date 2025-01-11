package config

import (
	"log"
	"os"
	"server/util"
)

var requiredEnvs = [...]string{"JURY_ADMIN_PASSWORD", "EMAIL_FROM"}
var smtpEnvs = []string{"EMAIL_HOST", "EMAIL_USERNAME", "EMAIL_PASSWORD"}
var sendgridEnvs = []string{"SENDGRID_API_KEY", "EMAIL_FROM_NAME"}
var mailgunEnvs = []string{"MAILGUN_API_KEY", "MAILGUN_DOMAIN", "EMAIL_FROM_NAME"}

// Checks to see if all required environmental variables are defined
func CheckEnv() {
	for _, v := range requiredEnvs {
		if !hasEnv(v) {
			log.Fatalf("ERROR: %s environmental variable not defined\n", v)
		}
	}

	// Check to see if either all smtp envs, all sendgrid envs, or all mailgun envs are defined
	if !util.All(util.Map(smtpEnvs, hasEnv)) && !util.All(util.Map(sendgridEnvs, hasEnv)) && !util.All(util.Map(mailgunEnvs, hasEnv)) {
		log.Fatalf("ERROR: either all envs for smtp, sendgrid, or mailgun must be defined (one of these sets): %v OR %v OR %v\n", smtpEnvs, sendgridEnvs, mailgunEnvs)
	}
}

// hasEnv returns true if the environmental variable is defined and not empty
func hasEnv(key string) bool {
	val, ok := os.LookupEnv(key)
	if !ok {
		return false
	}
	return val != ""
}

// GetEnv returns the value of the environmental variable or panics if it does not exist
func GetEnv(key string) string {
	val, ok := os.LookupEnv(key)
	if !ok {
		log.Fatalf("ERROR: %s environmental variable not defined\n", key)
		return ""
	}
	return val
}

// GetOptEnv returns the value of the environmental variable or the default value if it does not exist
func GetOptEnv(key string, defaultVal string) string {
	val, ok := os.LookupEnv(key)
	if !ok {
		return defaultVal
	}
	return val
}
