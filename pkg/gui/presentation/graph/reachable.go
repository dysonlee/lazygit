package graph

import (
	"github.com/jesseduffield/generics/set"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
)

// ReachableFrom returns the hashes of the commits in the list that can be
// reached from the commit at index start by following parents, including
// that commit itself. Parents may appear anywhere in the list, so this works
// for any log order.
func ReachableFrom(commits []*models.Commit, start int) *set.Set[*string] {
	commitsByHash := make(map[*string]*models.Commit, len(commits))
	for _, commit := range commits {
		commitsByHash[commit.HashPtr()] = commit
	}

	reachable := set.NewFromSlice([]*string{commits[start].HashPtr()})
	queue := []*models.Commit{commits[start]}
	for len(queue) > 0 {
		commit := queue[0]
		queue = queue[1:]
		for _, parentHash := range commit.ParentPtrs() {
			parent, ok := commitsByHash[parentHash]
			if ok && !reachable.Includes(parentHash) {
				reachable.Add(parentHash)
				queue = append(queue, parent)
			}
		}
	}

	return reachable
}
