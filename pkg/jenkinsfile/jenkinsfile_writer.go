package jenkinsfile

import (
	"bytes"
	"github.com/jenkins-x/jx/pkg/util"
	"strings"
)

var (
	contextFunctions = map[string]bool{
		"container": true,
		"dir":       true,
	}
)

// JenkinsfileStatement represents a statement
type JenkinsfileStatement struct {
	Function  string
	Arguments []string
	Statement string
	Children  []*JenkinsfileStatement
}

// Text returns the text line of the current function or statement
func (s *JenkinsfileStatement) Text() string {
	if s.Function != "" {
		text := s.Function
		expressions := []string{}
		for _, arg := range s.Arguments {
			expressions = append(expressions, asArgumentExpression(arg))
		}
		return text + "(" + strings.Join(expressions, ", ") + ")"
	}
	return s.Statement
}

// ContextEquals returns true if this statement is a context statement and it equals
// the same context as that statement
func (s *JenkinsfileStatement) ContextEquals(that *JenkinsfileStatement) bool {
	if s.Function == that.Function && contextFunctions[s.Function] {
		return util.StringArraysEqual(s.Arguments, that.Arguments)
	}
	return false
}

func asArgumentExpression(arg string) string {
	return "'" + arg + "'"
}

// JenkinsfileWriter implements the struct for Jenkinsfilewriter
type JenkinsfileWriter struct {
	InitialIndent string
	IndentText    string
	Buffer        bytes.Buffer
	IndentCount   int
}

// WriteJenkinsfileStatements writes the given Jenkinsfile statements as a string
func WriteJenkinsfileStatements(indentCount int, statements []*JenkinsfileStatement) string {
	writer := NewJenkinsfileWriter(indentCount)
	writer.Write(statements)
	return writer.String()
}

// NewJenkinsfileWriter creates a Jenkinsfile writer
func NewJenkinsfileWriter(indentCount int) *JenkinsfileWriter {
	return &JenkinsfileWriter{
		IndentText:  "  ",
		IndentCount: indentCount,
	}
}

func (w *JenkinsfileWriter) Write(inputStatements []*JenkinsfileStatement) {
	statements := w.combineSimilarContexts(inputStatements)
	w.writeStatement(nil, statements)
}

func (w *JenkinsfileWriter) writeStatement(parent *JenkinsfileStatement, statements []*JenkinsfileStatement) {
	for _, s := range statements {
		text := s.Text()
		hasChildren := len(s.Children) > 0
		if hasChildren {
			text = text + " {"
		}
		w.println(text)
		if hasChildren {
			w.IndentCount++
			w.writeStatement(s, s.Children)
			w.IndentCount--
		}
		if hasChildren {
			w.println("}")
		}
	}
}

func (w *JenkinsfileWriter) println(text string) {
	if text != "" {
		for i := 0; i < w.IndentCount; i++ {
			w.Buffer.WriteString(w.IndentText)
		}
		w.Buffer.WriteString(text)
	}
	w.Buffer.WriteString("\n")
}

// String returns the string value of this writer
func (w *JenkinsfileWriter) String() string {
	return w.Buffer.String()
}

func (w *JenkinsfileWriter) combineSimilarContexts(statements []*JenkinsfileStatement) []*JenkinsfileStatement {
	answer := append([]*JenkinsfileStatement{}, statements...)
	for i := 0; i < len(answer)-1; {
		s1 := answer[i]
		s2 := answer[i+1]
		// lets combine the children to the first node if the contexts are equal
		if s1.ContextEquals(s2) {
			s1.Children = append(s1.Children, s2.Children...)
			answer = append(answer[0:i+1], answer[i+2:]...)
		} else {
			i++
		}
	}
	for _, s := range answer {
		s.Children = w.combineSimilarContexts(s.Children)
	}
	return answer
}
