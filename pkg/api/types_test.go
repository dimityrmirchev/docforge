// Copyright (c) 2018 SAP SE or an SAP affiliate company. All rights reserved.
// This file is licensed under the Apache Software License, v.2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package api

import (
	"fmt"
	"reflect"
	"testing"
)

func TestParents(t *testing.T) {
	n11 := &Node{
		Source: []string{"https://github.com/gardener/gardener/blob/master/docs/README.md"},
	}
	n31 := &Node{
		Source: []string{"https://github.com/gardener/gardener/blob/master/docs/concepts/apiserver.md"},
	}
	n12 := &Node{
		Source: []string{"https://github.com/gardener/gardener/tree/master/docs/concepts"},
		Nodes:  []*Node{n31},
	}
	n0 := &Node{
		Nodes: []*Node{n11, n12},
	}
	n11.parent = n0
	n12.parent = n0
	n31.parent = n12
	cases := []struct {
		description string
		inNode      *Node
		want        []*Node
	}{
		{
			"get parents of node",
			n31,
			[]*Node{n0, n12},
		},
	}
	for _, c := range cases {
		fmt.Println(c.description)
		got := n31.Parents()
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("parents(%v) == %v, want %v", n31, got, c.want)
		}
	}
}
