// Code generated by "stringer -linecomment -type SelectionKind /home/u/Desktop/go/src/github.com/llir/l/ir/enum"; DO NOT EDIT.

package enum

import "fmt"
import "github.com/llir/l/ir/enum"

const _SelectionKind_name = "anyexactmatchlargestnoduplicatessamesize"

var _SelectionKind_index = [...]uint8{0, 3, 13, 20, 32, 40}

func SelectionKindFromString(s string) enum.SelectionKind {
	if len(s) == 0 {
		return 0
	}
	for i := range _SelectionKind_index[:len(_SelectionKind_index)-1] {
		if s == _SelectionKind_name[_SelectionKind_index[i]:_SelectionKind_index[i+1]] {
			return enum.SelectionKind(i)
		}
	}
	panic(fmt.Errorf("unable to locate SelectionKind enum corresponding to %q", s))
}
