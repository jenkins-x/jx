package main

import (
	"fmt"
	"os"
	"testing"
)

func TestDelete(t *testing.T) {

	testCases := []struct {
		src     string
		wantErr bool
	}{
		{"testdata/delete/src/simple-go-error", true},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("delete %s", tc.src), func(t *testing.T) {
			delete := &deleteCmd{
				appName: "",
				out:     os.Stdout,
			}
			err := delete.run(defaultDraftEnvironment())

			// Error checking
			if err != nil != tc.wantErr {
				t.Errorf("draft delete error = %v, wantErr %v", err, tc.wantErr)
				return
			}

		})
	}
}
