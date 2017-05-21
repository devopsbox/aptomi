package slinga

import (
	"bytes"
	"fmt"
	"github.com/awalterschulze/gographviz"
	"github.com/golang/glog"
	"io/ioutil"
	"os/exec"
	"sort"
	"strconv"
	"strings"
)

// See http://www.graphviz.org/doc/info/colors.html
const colorScheme = "set19"
const colorCount = 9

// Visualization for the policy
type PolicyVisualization struct {
	diff *ServiceUsageStateDiff
}

// Policy visualization object
func NewPolicyVisualization(diff *ServiceUsageStateDiff) PolicyVisualization {
	return PolicyVisualization{diff: diff}
}

// Draws and stores both pictures
func (vis PolicyVisualization) DrawAndStore() {
	// Draw & save resulting state
	nextGraph := vis.diff.Next.DrawVisualAndStore("complete")
	vis.saveGraph("complete", nextGraph)

	// Draw previous state
	prevGraph := vis.diff.Prev.DrawVisualAndStore("prev")
	vis.saveGraph("prev", prevGraph)

	// Draw delta (i.e. difference between resulting state and previous state)
	deltaGraph := vis.Delta(prevGraph, nextGraph)
	vis.saveGraph("delta", deltaGraph)
}

// Opens a picture
func (vis PolicyVisualization) OpenInPreview() {
	vis.OpenInPreviewWithSuffix("delta")
}

// Opens a picture, given a suffix
func (vis PolicyVisualization) OpenInPreviewWithSuffix(suffix string) {
	fileName := vis.getVisualFileNamePNG(suffix)
	command := exec.Command("open", []string{fileName}...)
	if err := command.Run(); err != nil {
		fmt.Print("Allocations (PNG): " + fileName)
	}
}

// Calculates delta between two graphs (it actually modifies "next" graph)
// TODO: deal with escape bulshit
func (vis PolicyVisualization) Delta(prev *gographviz.Escape, next *gographviz.Escape) *gographviz.Escape {
	// New nodes, edges, subgraphs must be highlighted
	{
		for _, s := range next.SubGraphs.SubGraphs {
			if _, inPrev := prev.SubGraphs.SubGraphs[s.Name]; !inPrev {
				// New subgraph -> no special treatment needed
				// s.Attrs.Add("style", "filled")
			}
		}
		for _, n := range next.Nodes.Nodes {
			if _, inPrev := prev.Nodes.Lookup[n.Name]; !inPrev {
				// New node -> filled green
				n.Attrs.Add("style", "filled")
				n.Attrs.Add("color", "green2")
			}
		}
		for _, e := range next.Edges.Edges {
			if _, inPrev := prev.Edges.SrcToDsts[e.Src][e.Dst]; !inPrev {
				// New edge -> bold, same color
				e.Attrs.Add("penwidth", "4")
			}
		}
	}

	// Removed nodes, edges, subgraphs must be highlighted
	{
		for _, s := range prev.SubGraphs.SubGraphs {
			if _, inNext := next.SubGraphs.SubGraphs[s.Name]; !inNext {
				// Removed subgraph -> add a sugraph filled red
				next.AddSubGraph(next.Name, s.Name, map[string]string{"style": "filled", "fillcolor": "gray18", "fontcolor": "white", "label": s.Attrs["label"]})
			}
		}

		for _, n := range prev.Nodes.Nodes {
			if _, inNext := next.Nodes.Lookup[n.Name]; !inNext {
				// Removed node -> add a node filled red
				n.Attrs.Add("style", "filled")
				n.Attrs.Add("color", "red")

				// Find previous subgraph & put it into the same subgraph
				subgraphName := vis.findSubraphName(prev, n.Name)
				next.Relations.Add(subgraphName, n.Name)

				next.Nodes.Add(n)
			}
		}
		for _, e := range prev.Edges.Edges {
			if _, inNext := next.Edges.SrcToDsts[e.Src][e.Dst]; !inNext {
				// Removed edge -> add an edge, dashed
				e.Attrs.Add("style", "dashed")
				next.Edges.Add(e)
			}
		}
	}

	return next
}

// Finds subgraph name from relations
func (vis PolicyVisualization) findSubraphName(prev *gographviz.Escape, nName string) string {
	for gName := range prev.Relations.ParentToChildren {
		if prev.Relations.ParentToChildren[gName][nName] {
			return gName
		}
	}
	return prev.Name
}

// Returns name of the file where visual is stored
func (vis PolicyVisualization) getVisualFileNamePNG(suffix string) string {
	return GetAptomiDBDir() + "/" + "graph_" + suffix + ".png"
}

// Stores usage state visual into a file
func (usage ServiceUsageState) DrawVisualAndStore(suffix string) *gographviz.Escape {
	users := LoadUsersFromDir(GetAptomiPolicyDir())

	// Write graph into a file
	graph := gographviz.NewEscape()
	graph.SetName("Main")
	graph.AddAttr("Main", "compound", "true")
	graph.SetDir(true)

	was := make(map[string]bool)

	// Add box/subgraph for users
	addSubgraphOnce(graph, "Main", "cluster_Users", map[string]string{"label": "Users"}, was)

	// Add box/subgraph for services
	addSubgraphOnce(graph, "Main", "cluster_Services", map[string]string{"label": "Services"}, was)

	// How many colors have been used
	usedColors := 0
	colorForUser := make(map[string]int)

	// First of all, let's show all dependencies (who requested what)
	if usage.Dependencies != nil {
		for service, dependencies := range usage.Dependencies.Dependencies {
			// Add a node with service
			addNodeOnce(graph, "cluster_Services", service, nil, was)

			// For every user who has a dependency on this service
			for _, d := range dependencies {
				color := getUserColor(d.UserId, colorForUser, &usedColors)

				// Add a node with user
				user := users.Users[d.UserId]
				label := "Name: " + user.Name + " (" + user.Id + ")"
				var keys []string
				for k := range user.Labels {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				for _, k := range keys {
					label += "\n" + k + " = " + user.Labels[k]
				}
				addNodeOnce(graph, "cluster_Users", d.UserId, map[string]string{"label": label, "style": "filled", "fillcolor": "/" + colorScheme + "/" + strconv.Itoa(color)}, was)

				// Add an edge from user to a service
				addEdge(graph, d.UserId, service, map[string]string{"color": "/" + colorScheme + "/" + strconv.Itoa(color)})
			}
		}
	}

	// Second, visualize evaluated links
	for key, linkStruct := range usage.ResolvedLinks {
		keyArray := strings.Split(key, "#")
		service := keyArray[0]
		contextAndAllocation := keyArray[1] + "#" + keyArray[2]
		component := keyArray[3]

		// only add edges to "root" components (i.e. services)
		if component != componentRootName {
			continue
		}

		// Key for allocation
		serviceAllocationKey := service + "_" + contextAndAllocation

		// Add a node with service
		addNodeOnce(graph, "cluster_Services", service, nil, was)

		// Add box/subgraph for a given service, containing all its allocations
		addSubgraphOnce(graph, "Main", "cluster_Service_Allocations_"+service, map[string]string{"label": "Allocations for service: " + service}, was)

		// Add a node with allocation
		addNodeOnce(graph, "cluster_Service_Allocations_"+service, serviceAllocationKey, map[string]string{"label": "Context: " + keyArray[1] + "\n" + "Allocation: " + keyArray[2]}, was)

		// Add an edge from service to allocation box
		for _, userId := range linkStruct.UserIds {
			color := getUserColor(userId, colorForUser, &usedColors)
			addEdge(graph, service, serviceAllocationKey, map[string]string{"color": "/" + colorScheme + "/" + strconv.Itoa(color)})
		}
	}

	// Third, show cross-service dependencies
	if usage.Policy != nil {
		for serviceName1, service1 := range usage.Policy.Services {
			// Resolve every component
			for _, component := range service1.Components {
				serviceName2 := component.Service
				if serviceName2 != "" {
					// Add a node with service1
					addNodeOnce(graph, "cluster_Services", serviceName1, nil, was)

					// Add a node with service2
					addNodeOnce(graph, "cluster_Services", serviceName2, nil, was)

					// Show dependency
					addEdge(graph, serviceName1, serviceName2, map[string]string{"color": "gray60"})
				}
			}
		}
	} else {
		addNodeOnce(graph, "", "No entries", nil, was)
	}

	return graph
}

// Saves graph into a file
func (vis PolicyVisualization) saveGraph(suffix string, graph *gographviz.Escape) {
	fileNameDot := GetAptomiDBDir() + "/" + "graph_" + suffix + "_full.dot"
	fileNameDotFlat := GetAptomiDBDir() + "/" + "graph_" + suffix + "_flat.dot"
	err := ioutil.WriteFile(fileNameDot, []byte(graph.String()), 0644)
	if err != nil {
		glog.Fatalf("Unable to write to a file: %s", fileNameDot)
	}
	// Call graphviz to flatten an image
	{
		cmd := "unflatten"
		args := []string{"-f", "-l", "4", "-o" + fileNameDotFlat, fileNameDot}
		command := exec.Command(cmd, args...)
		var outb, errb bytes.Buffer
		command.Stdout = &outb
		command.Stderr = &errb
		if err := command.Run(); err != nil {
			glog.Fatalf("Unable to execute graphviz (%s): %s %s %v", cmd, outb.String(), errb.String(), err)
		}
	}
	// Call graphviz to generate an image
	{
		cmd := "dot"
		args := []string{"-Tpng", "-o" + vis.getVisualFileNamePNG(suffix), fileNameDotFlat}
		// args := []string{"-Tpng", "-Kfdp", "-o" + vis.getVisualFileNamePNG(suffix), fileNameDotFlat}
		command := exec.Command(cmd, args...)
		var outb, errb bytes.Buffer
		command.Stdout = &outb
		command.Stderr = &errb
		if err := command.Run(); err != nil {
			glog.Fatalf("Unable to execute graphviz (%s): %s %s %v", cmd, outb.String(), errb.String(), err)
		}
	}
}

// Returns a color for the given user
func getUserColor(userId string, colorForUser map[string]int, usedColors *int) int {
	color, ok := colorForUser[userId]
	if !ok {
		*usedColors++
		if *usedColors > colorCount {
			*usedColors = 1
		}
		color = *usedColors
		colorForUser[userId] = color
	}
	return color
}

// Adds an edge if it doesn't exist already
func addEdge(g *gographviz.Escape, src string, dst string, attrs map[string]string) {
	g.AddEdge(src, dst, true, attrs)
}

// Adds a subgraph if it doesn't exist already
func addSubgraphOnce(g *gographviz.Escape, parentGraph string, name string, attrs map[string]string, was map[string]bool) {
	wasKey := "SUBGRAPH" + "#" + parentGraph + "#" + name
	if !was[wasKey] {
		g.AddSubGraph(parentGraph, name, attrs)
		was[wasKey] = true
	}
}

// Adds a node if it doesn't exist already
func addNodeOnce(g *gographviz.Escape, parentGraph string, name string, attrs map[string]string, was map[string]bool) {
	wasKey := "NODE" + "#" + parentGraph + "#" + name
	if !was[wasKey] {
		g.AddNode(parentGraph, name, attrs)
		was[wasKey] = true
	}
}
