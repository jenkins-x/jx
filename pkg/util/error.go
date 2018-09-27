package util

import "k8s.io/apimachinery/pkg/util/errors"

// Combine combines the non null errors into a single error or returns null
func CombineErrors(errs ...error) error {
	answer := []error{}
	for _, err := range errs {
		if err != nil {
			answer = append(answer, err)
		}
	}
	if len(answer) == 0 {
		return nil
	} else if len(answer) == 1 {
		return answer[0]
	}
	return errors.NewAggregate(answer)
}
