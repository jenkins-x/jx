package git

import (
	"fmt"
	"strings"

	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

// node is a tree node in commit tree
type node struct {
	value  *object.Commit
	parent *node
}

// errNoDiffBetweenReferences is triggered when we can't
// produce any diff between 2 references
type errNoDiffBetweenReferences struct {
	from string
	to   string
}

func (e errNoDiffBetweenReferences) Error() string {
	return fmt.Sprintf(`can't produce a diff between %s and %s, check your range is correct by running "git log %[1]s..%[2]s" command`, e.from, e.to)
}

// errRepositoryPath is triggered when repository path can't be opened
type errRepositoryPath struct {
	path string
}

func (e errRepositoryPath) Error() string {
	return fmt.Sprintf(`check "%s" is an existing git repository path`, e.path)
}

// errReferenceNotFound is triggered when reference can't be
// found in git repository
type errReferenceNotFound struct {
	ref string
}

func (e errReferenceNotFound) Error() string {
	return fmt.Sprintf(`reference "%s" can't be found in git repository`, e.ref)
}

// errBrowsingTree is triggered when something wrong occurred during commit analysis process
var errBrowsingTree = fmt.Errorf("an issue occurred during tree analysis")

// resolveRef gives hash commit for a given string reference
func resolveRef(refCommit string, repository *git.Repository) (*object.Commit, error) {
	hash := plumbing.Hash{}

	if strings.ToLower(refCommit) == "head" {
		head, err := repository.Head()

		if err == nil {
			return repository.CommitObject(head.Hash())
		}
	}

	iter, err := repository.References()

	if err != nil {
		return &object.Commit{}, errReferenceNotFound{refCommit}
	}

	err = iter.ForEach(func(ref *plumbing.Reference) error {
		if ref.Name().Short() == refCommit {
			hash = ref.Hash()
		}

		return nil
	})

	if err == nil && !hash.IsZero() {
		return repository.CommitObject(hash)
	}

	hash = plumbing.NewHash(refCommit)

	if !hash.IsZero() {
		return repository.CommitObject(hash)
	}

	return &object.Commit{}, errReferenceNotFound{refCommit}
}

// FetchCommits retrieves commits in a reference range
func FetchCommits(repoPath string, fromRef string, toRef string) (*[]object.Commit, error) {
	rep, err := git.PlainOpen(repoPath)

	if err != nil {
		return nil, errRepositoryPath{repoPath}
	}

	fromCommit, err := resolveRef(fromRef, rep)

	if err != nil {
		return &[]object.Commit{}, err
	}

	toCommit, err := resolveRef(toRef, rep)

	if err != nil {
		return &[]object.Commit{}, err
	}

	var ok bool
	var commits *[]object.Commit

	exclusionList, err := buildOriginCommitList(fromCommit)

	if err != nil {
		return nil, err
	}

	if _, ok = exclusionList[toCommit.ID().String()]; ok {
		return nil, errNoDiffBetweenReferences{fromRef, toRef}
	}

	commits, err = findDiffCommits(toCommit, &exclusionList)

	if err != nil {
		return nil, err
	}

	if len(*commits) == 0 {
		return nil, errNoDiffBetweenReferences{fromRef, toRef}
	}

	return commits, nil
}

// buildOriginCommitList browses git tree from a given commit
// till root commit using kind of breadth first search algorithm
// and grab commit ID to a map with ID as key
func buildOriginCommitList(commit *object.Commit) (map[string]bool, error) {
	queue := append([]*object.Commit{}, commit)
	seen := map[string]bool{commit.ID().String(): true}

	for len(queue) > 0 {
		current := queue[0]
		queue = append([]*object.Commit{}, queue[1:]...)

		err := current.Parents().ForEach(
			func(c *object.Commit) error {
				if _, ok := seen[c.ID().String()]; !ok {
					seen[c.ID().String()] = true
					queue = append(queue, c)
				}

				return nil
			})

		if err != nil && err.Error() != plumbing.ErrObjectNotFound.Error() {
			return seen, errBrowsingTree
		}
	}

	return seen, nil
}

// findDiffCommits extracts commits that are no part of a given commit list
// using kind of depth first search algorithm to keep commits ordered
func findDiffCommits(commit *object.Commit, exclusionList *map[string]bool) (*[]object.Commit, error) {
	commits := []object.Commit{}
	queue := append([]*node{}, &node{value: commit})
	seen := map[string]bool{commit.ID().String(): true}
	var current *node

	for len(queue) > 0 {
		current = queue[0]
		queue = append([]*node{}, queue[1:]...)

		if _, ok := (*exclusionList)[current.value.ID().String()]; !ok {
			commits = append(commits, *(current.value))
		}

		err := current.value.Parents().ForEach(
			func(c *object.Commit) error {
				if _, ok := seen[c.ID().String()]; !ok {
					seen[c.ID().String()] = true
					n := &node{value: c, parent: current}
					queue = append([]*node{n}, queue...)
				}

				return nil
			})

		if err != nil && err.Error() != plumbing.ErrObjectNotFound.Error() {
			return &commits, errBrowsingTree
		}
	}

	return &commits, nil
}
