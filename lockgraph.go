package deadlock

import (
	"crypto/sha1"
	"fmt"
	"strings"
	"sync"
	"unsafe"
)

var (
	globalGraph = globalLockGraph{
		mergedGraph: newLockGraph(),
	}
)

type lockGraph struct {
	startVertices map[unsafe.Pointer]struct{}
	edges         map[unsafe.Pointer][]lockEdge
	sha1          [20]byte
}

func (lg *lockGraph) updateSha1() {
	edges := []string{}
	lg.dfsByEdge(nil, func(edge *lockEdge) {
		s := fmt.Sprintf("%s@%v=>%s@%v", edge.start.line, edge.start.lockInst, edge.end.line, edge.end.lockInst)
		edges = append(edges, s)
	}, nil)

	lg.sha1 = sha1.Sum([]byte(strings.Join(edges, "\n")))
}

func newLockGraph() *lockGraph {
	return &lockGraph{
		startVertices: map[unsafe.Pointer]struct{}{},
		edges:         map[unsafe.Pointer][]lockEdge{},
	}
}

type globalLockGraph struct {
	lock             sync.Mutex
	no               int
	mergedGraph      *lockGraph
	historySubGraphs []*lockGraph
}

type lockEdge struct {
	start lockOp
	end   lockOp
	no    int
}

func (lg *lockGraph) addStartVertices(vertex unsafe.Pointer) {
	lg.startVertices[vertex] = struct{}{}
}

func (lg *lockGraph) addEdge(lockOpA, lockOpB lockOp) {
	lg.edges[lockOpA.lockInst] = append(lg.edges[lockOpA.lockInst], lockEdge{
		start: lockOpA,
		end:   lockOpB,
	})
}

func (lg *lockGraph) checkCyclicExists() {
	var (
		iterVisited   = map[*lockEdge]struct{}{}
		lockEdges     []*lockEdge
		deadlockFound bool
	)
	lg.dfsByEdge(func(visitedEdge *lockEdge) bool {
		if _, ok := iterVisited[visitedEdge]; ok {
			distinctLockedEdges := map[int]*lockEdge{}
			for _, edge := range lockEdges {
				distinctLockedEdges[edge.no] = edge
			}

			deadlockFound = true
			infos := make([]DeadlockInfo, 0, len(distinctLockedEdges))
			for _, edge := range distinctLockedEdges {
				infos = append(infos, DeadlockInfo{
					stack:    edge.end.stack,
					lockPath: genLockPath(lg.findLockPath(edge)),
				})
			}
			opts.OnDeadlock(TypeCyclic, infos)
		}
		iterVisited[visitedEdge] = struct{}{}
		return deadlockFound
	}, func(visitedEdge *lockEdge) {
		for len(lockEdges) > 0 {
			if visitedEdge.start.Equal(lockEdges[len(lockEdges)-1].end) {
				break
			}

			lockEdges = lockEdges[:len(lockEdges)-1]
		}

		lockEdges = append(lockEdges, visitedEdge)
	}, func() {
		iterVisited = map[*lockEdge]struct{}{}
	})
}

func (lg *lockGraph) dfsByEdge(filter func(edge *lockEdge) bool, visit func(edge *lockEdge), newIter func()) {
	visited := map[*lockEdge]struct{}{}
	filterWrap := func(edge *lockEdge) bool {
		if filter != nil && filter(edge) {
			return true
		}
		if _, ok := visited[edge]; ok {
			return true
		}
		visited[edge] = struct{}{}
		return false
	}

	for v := range lg.startVertices {
		if newIter != nil {
			newIter()
		}
		lg.doDfsByEdge(v, filterWrap, visit)
	}
}

func (lg *lockGraph) doDfsByEdge(v unsafe.Pointer, filter func(edge *lockEdge) bool, visit func(edge *lockEdge)) {
	for i := range lg.edges[v] {
		edge := &lg.edges[v][i]

		if filter != nil && filter(edge) {
			continue
		}

		visit(edge)
		lg.doDfsByEdge(edge.end.lockInst, filter, visit)
	}
}

func (lg *lockGraph) findLockPath(edge *lockEdge) []lockOp {
	lockPaths := make([]lockOp, 0)

	found := false
	lg.dfsByEdge(func(visitedEdge *lockEdge) bool {
		return edge.no != visitedEdge.no || found
	}, func(visitedEdge *lockEdge) {
		for len(lockPaths) > 0 {
			if visitedEdge.start.Equal(lockPaths[len(lockPaths)-1]) {
				break
			}

			lockPaths = lockPaths[:len(lockPaths)-1]
		}

		if len(lockPaths) == 0 {
			lockPaths = append(lockPaths, visitedEdge.start, visitedEdge.end)
		} else {
			lockPaths = append(lockPaths, visitedEdge.end)
		}

		if visitedEdge.start.Equal(edge.start) && visitedEdge.end.Equal(edge.end) {
			found = true
		}
	}, nil)

	return lockPaths
}

func (glg *globalLockGraph) merge(graph *lockGraph) {
	if len(graph.edges) == 0 {
		return
	}

	graph.updateSha1()

	glg.lock.Lock()
	defer glg.lock.Unlock()

	if glg.lockGraphExists(graph) {
		return
	}

	glg.historySubGraphs = append(glg.historySubGraphs, graph)

	//graph.dfsByEdge(nil, func(edge *lockEdge) {
	//	fmt.Println("mergeGraph start:", edge.start.line, edge.start.lockInst, " end:", edge.end.line, edge.end.lockInst)
	//}, nil)

	for v, _ := range graph.startVertices {
		glg.mergedGraph.startVertices[v] = struct{}{}
	}

	glg.no += 1
	for v, edges := range graph.edges {
		for _, e := range edges {
			e.no = glg.no
			glg.mergedGraph.edges[v] = append(glg.mergedGraph.edges[v], e)
		}
	}

	glg.mergedGraph.checkCyclicExists()
}

func (glg *globalLockGraph) lockGraphExists(graph *lockGraph) bool {
	for _, subGraph := range glg.historySubGraphs {
		if subGraph.sha1 == graph.sha1 {
			return true
		}
	}

	return false
}
