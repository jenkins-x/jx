package util_test

import (
	"fmt"
	"regexp"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jenkins-x/jx/pkg/util"
)

func TestReplaceAllStringSubmatchFunc(t *testing.T) {
	incrementVersionRepl := func(groups []util.Group) []string {
		answer := make([]string, 0)
		for _, group := range groups {
			i, err := strconv.Atoi(group.Value)
			assert.NoError(t, err)
			answer = append(answer, fmt.Sprintf("%d", i+1))
		}
		return answer
	}
	type args struct {
		re   *regexp.Regexp
		str  string
		repl func([]util.Group) []string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "JX_VERSION",
			args: args{
				re:   regexp.MustCompile(`JX_VERSION=(.*)`),
				str:  "JX_VERSION=10",
				repl: incrementVersionRepl,
			},
			want: "JX_VERSION=11",
		},
		{
			name: "release",
			args: args{
				re:   regexp.MustCompile(`\s*release = \"(.*)\"`),
				str:  "    release = \"10\"",
				repl: incrementVersionRepl,
			},
			want: "    release = \"11\"",
		},
		{
			name: "twice",
			args: args{
				re:   regexp.MustCompile(`\s*release = \"(.*)\"`),
				str:  "    release = \"10\"\n    release = \"8\"",
				repl: incrementVersionRepl,
			},
			want: "    release = \"11\"\n    release = \"9\"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := util.ReplaceAllStringSubmatchFunc(tt.args.re, tt.args.str, tt.args.repl); got != tt.want {
				t.Errorf("ReplaceAllStringSubmatchFunc() = %v, want %v", got, tt.want)
			}
		})
	}
}
