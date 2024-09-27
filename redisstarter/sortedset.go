package redisstarter

type cmdSortedSet struct {
}

var sortedSetCmd = new(cmdSortedSet)

func SortedSetCmd() *cmdSortedSet {
	return sortedSetCmd
}
