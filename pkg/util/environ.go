package util

import "os"

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

func RestoreEnviron(environ map[string]string) error {
	for key, value := range environ {
		err := os.Setenv(key, value)
		if err != nil {
			return err
		}
	}
	return nil
}
