package util

import "os"

// GetAndCleanEnviron cleans the provided env variables and returns their current value
func GetAndCleanEnviron(keys []string) (map[string]string, error) {
	environ := map[string]string{}
	for _, key := range keys {
		value, set := os.LookupEnv(key)
		if set {
			environ[key] = value
			err := os.Unsetenv(key)
			if err != nil {
				return environ, err
			}
		}
	}
	return environ, nil
}

// RestoreEnviron sets the in the environment the environment variables provided as input
func RestoreEnviron(environ map[string]string) error {
	for key, value := range environ {
		err := os.Setenv(key, value)
		if err != nil {
			return err
		}
	}
	return nil
}
