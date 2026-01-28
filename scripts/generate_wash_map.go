package main  // 修改这里，从 package scripts 改为 package main

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// AddressNode 表示地址节点
type AddressNode struct {
    ID       string        `json:"id"`
    Value    string        `json:"value"`
    Next     []*AddressNode `json:"next,omitempty"`
    Type     string        `json:"type"`
    CountID  int           `json:"count_id"`    // 新增：表示在当前 count 中的序号
    DepthID  int           `json:"depth_id"`    // 新增：表示在当前深度中的序号
}

// Edge 表示节点之间的边关系
type Edge struct {
	From      string `json:"from"`
	To        string `json:"to"`
	FromValue string `json:"from_value"`
	ToValue   string `json:"to_value"`
}

// 生成随机值
func getRandomValue() string {
	return fmt.Sprintf("#%d", rand.Intn(1000))
}

// BuildLinearChains 构建线性链
func BuildLinearChains(rootLabel string, count int, depth int) []*AddressNode {
    // 创建共享的根节点
    root := &AddressNode{
        ID:      rootLabel,
        Value:   getRandomValue(),
        Type:    "root",
        Next:    make([]*AddressNode, 0),
        CountID: 0,        // 根节点的 CountID 设为 0
        DepthID: 0,        // 根节点的 DepthID 设为 0
    }
    
    // 为每个 count 创建一条链
    for i := 1; i <= count; i++ {
        lastNode := root
        // 根据 depth 创建指定深度的链
        for d := 1; d <= depth; d++ {
            nodeType := "leaf"
            if d < depth {
                nodeType = "intermediate"
            }
            
            current := &AddressNode{
                ID:      fmt.Sprintf("%c%d", 'A'+d-1, i),
                Value:   getRandomValue(),
                Type:    nodeType,
                Next:    make([]*AddressNode, 0),
                CountID: i,          // 设置当前的 count 序号
                DepthID: d,      // 设置当前的深度序号
            }
            
            // 将当前节点添加到上一个节点的子节点列表中
            lastNode.Next = append(lastNode.Next, current)
            lastNode = current
        }
    }
    
    return []*AddressNode{root}
}

// PrintChains 打印链结构
func PrintChains(chains []*AddressNode) {
    for _, chain := range chains {
        visited := make(map[string]bool)
        printNode(chain, visited, "")
    }
}

func printNode(node *AddressNode, visited map[string]bool, prefix string) {
    if node == nil || visited[node.ID] {
        return
    }
    
    fmt.Printf("%s%s (Value: %s, Type: %s, CountID: %d, DepthID: %d)\n", 
        prefix, node.ID, node.Value, node.Type, node.CountID, node.DepthID)
    visited[node.ID] = true
    
    for _, next := range node.Next {
        printNode(next, visited, prefix+"  ")
    }
}

// GenerateMermaidDiagram 生成 Mermaid 图表
func GenerateMermaidDiagram(chains []*AddressNode) string {
    lines := []string{"graph TD"}
    for _, chain := range chains {
        visited := make(map[string]bool)
        generateMermaidLines(chain, visited, &lines)
    }
    mermaidContent := strings.Join(lines, "\n")

    // 创建输出目录
    outputDir := filepath.Join("output", "mermaid-diagram")
    os.MkdirAll(outputDir, 0755)

    // 生成文件名
    timestamp := time.Now().Format("2006-01-02T15-04-05")
    filename := filepath.Join(outputDir, timestamp+".md")

    // 写入文件
    if err := os.WriteFile(filename, []byte(mermaidContent), 0644); err != nil {
        fmt.Printf("保存 Mermaid 图表时出错: %v\n", err)
    } else {
        fmt.Printf("已将 Mermaid 图表保存到: %s\n", filename)
    }

    return mermaidContent
}

func generateMermaidLines(node *AddressNode, visited map[string]bool, lines *[]string) {
    if node == nil || visited[node.ID] {
        return
    }
    
    visited[node.ID] = true
    for _, next := range node.Next {
        *lines = append(*lines, fmt.Sprintf("  %s --> %s", node.ID, next.ID))
        generateMermaidLines(next, visited, lines)
    }
}

// GenerateEdges 生成边关系数组
func GenerateEdges(chains []*AddressNode) []Edge {
    edges := make([]Edge, 0)
    for _, chain := range chains {
        visited := make(map[string]bool)
        generateEdgesForNode(chain, visited, &edges)
    }
    return edges
}

func generateEdgesForNode(node *AddressNode, visited map[string]bool, edges *[]Edge) {
    if node == nil || visited[node.ID] {
        return
    }
    
    visited[node.ID] = true
    for _, next := range node.Next {
        *edges = append(*edges, Edge{
            From:      node.ID,
            To:        next.ID,
            FromValue: node.Value,
            ToValue:  next.Value,
        })
        generateEdgesForNode(next, visited, edges)
    }
}

func ExportAllNodes(chains []*AddressNode) []*AddressNode {
    nodes := make([]*AddressNode, 0)
    visited := make(map[string]*AddressNode)
    
    for _, chain := range chains {
        exportNodesForNode(chain, visited, &nodes)
    }
    return nodes
}

func exportNodesForNode(node *AddressNode, visited map[string]*AddressNode, nodes *[]*AddressNode) {
    if node == nil || visited[node.ID] != nil {
        return
    }
    
    newNode := &AddressNode{
        ID:    node.ID,
        Value: node.Value,
        Type:  node.Type,
        CountID: node.CountID,
        DepthID: node.DepthID,
        Next:  make([]*AddressNode, 0),
    }
    *nodes = append(*nodes, newNode)
    visited[node.ID] = newNode
    
    for _, next := range node.Next {
        exportNodesForNode(next, visited, nodes)
    }
}

// 示例使用
func main() {
	// 初始化随机数生成器
	rand.Seed(time.Now().UnixNano())

	// 生成线性链
	linear := BuildLinearChains("账户", 6, 4)
    
    // 打印所有节点信息
    fmt.Println("All Nodes:")
    for _, node := range ExportAllNodes(linear) {
        fmt.Printf("ID: %s, Value: %s, CountID: %d, DepthID: %d\n", node.ID, node.Value, node.CountID, node.DepthID)
    }
	
	// print chains
	PrintChains(linear)
	
	// 生成并保存图表
	// GenerateMermaidDiagram(linear)
	
	// 打印边关系
	fmt.Println("\nlinear edges:")
	edges := GenerateEdges(linear)
	fmt.Printf("%+v\n", edges)
}