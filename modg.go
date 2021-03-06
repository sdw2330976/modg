package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"html/template"
	"io"
	"os"
	"os/exec"
	"strings"
)

type ModGraph struct {
	r         io.Reader
	Mod       map[string]*GraphNode
	DepMod    map[int][]int
	Keyword   string
	FillColor string
}

type GraphNode struct {
	NodeId    int
	FillColor string
}

func NewGraphNode(id int) *GraphNode {

	return &GraphNode{
		NodeId: id,
	}
}
func NewModGraph(r io.Reader) *ModGraph {
	return &ModGraph{
		r:      r,
		Mod:    make(map[string]*GraphNode),
		DepMod: make(map[int][]int),
	}
}

func (m *ModGraph) isTargetNode(lib string) bool {
	if m.Keyword != "" && strings.Contains(lib, m.Keyword) {
		return true
	}
	return false
}
func (m *ModGraph) Parse() {
	scanner := bufio.NewScanner(m.r)
	var num int
	for scanner.Scan() {
		line := scanner.Text()
		relation := strings.Split(line, " ")
		lib, depLib := strings.TrimSpace(relation[0]), strings.TrimSpace(relation[1])
		if !m.isTargetNode(lib) && !m.isTargetNode(depLib) {
			continue
		}
		mod, ok := m.Mod[lib]
		if !ok {
			mod = NewGraphNode(num)
			if m.isTargetNode(lib) {
				mod.FillColor = m.FillColor
			}
			m.Mod[lib] = mod
			num += 1
		}
		depMod, ok := m.Mod[depLib]
		if !ok {
			depMod = NewGraphNode(num)
			if m.isTargetNode(depLib) {
				depMod.FillColor = m.FillColor
			}
			m.Mod[depLib] = depMod
			num += 1
		}
		if arr, ok := m.DepMod[mod.NodeId]; !ok {
			m.DepMod[mod.NodeId] = []int{depMod.NodeId}
		} else {
			m.DepMod[mod.NodeId] = append(arr, depMod.NodeId)
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "os stdin format not valid,", err)
	}
}
func (m *ModGraph) Render(w io.Writer) error {
	graphTemplate := `
		digraph{
			{{- if eq .direction "horizontal" -}}
			rankdir=LR;
			{{ end -}}
		  node [shape=box];
			{{ range $mod,$modNode := .mods -}}
				{{ $modNode.NodeId }} [ {{if $modNode.FillColor }}style=filled, fillcolor = {{$modNode.FillColor}} ,{{ end }} label = "{{ $mod }}"];
			{{ end -}}
	
			{{ range $modId,$depModIds := .depMods -}}
				{{ range $_, $depMod := $depModIds -}}
					{{ $modId }} -> {{ $depMod }};
				{{ end -}}
			{{ end -}}
		}`
	tmplate, err := template.New("modGraph").Parse(graphTemplate)
	if err != nil {
		return fmt.Errorf("parse template error:%v", err)
	}
	var direction string
	if len(m.DepMod) >= 1 {
		direction = "horizontal"
	}
	if err := tmplate.Execute(w, map[string]interface{}{
		"mods":      m.Mod,
		"depMods":   m.DepMod,
		"direction": direction,
	}); err != nil {
		return fmt.Errorf("execute template error:%v", err)
	}
	return nil
}
func main() {
	/*file, err := os.Stdin.Stat()
	if err != nil {
		fmt.Println("os stdin stat error:", err)
		os.Exit(1)
	}
	if file.Mode()&os.ModeNamedPipe == 0 {
		fmt.Println("this command should use with pipes")
		os.Exit(1)
	}*/
	keyword := flag.String("k", "", "specific keyword to filter lib")
	fillColor := flag.String("c", "yellow", "specific mod node fill color")
	flag.Parse()
	cmd := exec.Command("go", "mod", "graph")
	buffer := &bytes.Buffer{}
	cmd.Stdout = buffer
	err := cmd.Start()
	if err != nil {
		panic(err)
	}
	if err = cmd.Wait(); err != nil {
		panic(err)
	} else {
		graph := NewModGraph(buffer)
		graph.Keyword = *keyword
		graph.FillColor = *fillColor
		graph.Parse()
		graph.Render(os.Stdout)
	}
}
