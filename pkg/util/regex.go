package util

import "regexp"

type Group struct {
	Value string
	Start int
	End   int
}

func ReplaceAllStringSubmatchFunc(re *regexp.Regexp, str string, repl func([]Group) []string) string {
	result := ""
	lastIndex := 0

	for _, v := range re.FindAllSubmatchIndex([]byte(str), -1) {
		groups := []Group{}
		for i := 0; i < len(v); i += 2 {
			if i == 0 {
				// Skip the match of the entire string
				continue
			}
			groups = append(groups, Group{
				Value: str[v[i]:v[i+1]],
				Start: v[i],
				End:   v[i+1],
			})
		}
		for i, subst := range repl(groups) {
			group := groups[i]
			result += str[lastIndex:group.Start] + subst
			lastIndex = group.End
		}
	}
	return result + str[lastIndex:]
}
