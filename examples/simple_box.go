// Example: Simple box design using Lignin graph architecture.
package main

import (
	"fmt"
	"log"

	"github.com/chazu/lignin/pkg/graph"
)

func main() {
	// Create a new design builder
	db := graph.NewDesignBuilder()

	// Define dimensions (in mm)
	legSize := graph.Vector3{X: 50, Y: 50, Z: 750}
	apronSize := graph.Vector3{X: 100, Y: 50, Z: 600}
	topSize := graph.Vector3{X: 600, Y: 600, Z: 25}

	fmt.Println("Building simple box design...")

	// Create primitive shapes
	legPrimitive := db.AddPrimitive("leg", "cuboid", legSize)
	apronPrimitive := db.AddPrimitive("apron", "cuboid", apronSize)
	topPrimitive := db.AddPrimitive("top", "cuboid", topSize)

	// Create leg parts (4 legs)
	leg1Node, leg1Part, err := db.AddPart("leg-front-left", []graph.NodeID{legPrimitive}, graph.GrainZ, "oak")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Created part: %s (node: %s)\n", leg1Part, leg1Node)

	leg2Node, leg2Part, err := db.AddPart("leg-front-right", []graph.NodeID{legPrimitive}, graph.GrainZ, "oak")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Created part: %s (node: %s)\n", leg2Part, leg2Node)

	leg3Node, leg3Part, err := db.AddPart("leg-back-left", []graph.NodeID{legPrimitive}, graph.GrainZ, "oak")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Created part: %s (node: %s)\n", leg3Part, leg3Node)

	leg4Node, leg4Part, err := db.AddPart("leg-back-right", []graph.NodeID{legPrimitive}, graph.GrainZ, "oak")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Created part: %s (node: %s)\n", leg4Part, leg4Node)

	// Create apron parts (4 aprons)
	apron1Node, apron1Part, err := db.AddPart("apron-front", []graph.NodeID{apronPrimitive}, graph.GrainX, "oak")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Created part: %s (node: %s)\n", apron1Part, apron1Node)

	apron2Node, apron2Part, err := db.AddPart("apron-back", []graph.NodeID{apronPrimitive}, graph.GrainX, "oak")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Created part: %s (node: %s)\n", apron2Part, apron2Node)

	apron3Node, apron3Part, err := db.AddPart("apron-left", []graph.NodeID{apronPrimitive}, graph.GrainX, "oak")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Created part: %s (node: %s)\n", apron3Part, apron3Node)

	apron4Node, apron4Part, err := db.AddPart("apron-right", []graph.NodeID{apronPrimitive}, graph.GrainX, "oak")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Created part: %s (node: %s)\n", apron4Part, apron4Node)

	// Create top part
	topNode, topPart, err := db.AddPart("top", []graph.NodeID{topPrimitive}, graph.GrainX, "oak")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Created part: %s (node: %s)\n", topPart, topNode)

	// Create butt joints between legs and aprons
	fmt.Println("\nCreating butt joints...")

	// Front left joint (leg1 to apron1)
	joint1, err := db.AddJoin(graph.JoinTypeButt, leg1Part, apron1Part, 0, 2, 0.2)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Created joint: %s\n", joint1)

	// Front right joint (leg2 to apron1)
	joint2, err := db.AddJoin(graph.JoinTypeButt, leg2Part, apron1Part, 1, 3, 0.2)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Created joint: %s\n", joint2)

	// Back left joint (leg3 to apron2)
	joint3, err := db.AddJoin(graph.JoinTypeButt, leg3Part, apron2Part, 2, 0, 0.2)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Created joint: %s\n", joint3)

	// Back right joint (leg4 to apron2)
	joint4, err := db.AddJoin(graph.JoinTypeButt, leg4Part, apron2Part, 3, 1, 0.2)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Created joint: %s\n", joint4)

	// Left side joints (apron3 to legs)
	joint5, err := db.AddJoin(graph.JoinTypeButt, leg1Part, apron3Part, 4, 0, 0.2)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Created joint: %s\n", joint5)

	joint6, err := db.AddJoin(graph.JoinTypeButt, leg3Part, apron3Part, 5, 1, 0.2)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Created joint: %s\n", joint6)

	// Right side joints (apron4 to legs)
	joint7, err := db.AddJoin(graph.JoinTypeButt, leg2Part, apron4Part, 6, 2, 0.2)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Created joint: %s\n", joint7)

	joint8, err := db.AddJoin(graph.JoinTypeButt, leg4Part, apron4Part, 7, 3, 0.2)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Created joint: %s\n", joint8)

	// Build the complete design
	design := db.BuildDesign("1.0")

	// Print design statistics
	fmt.Printf("\nDesign Statistics:\n")
	fmt.Printf("  Version: %s\n", design.Version)
	fmt.Printf("  Total nodes: %d\n", len(design.Graph.Nodes))
	fmt.Printf("  Total parts: %d\n", len(design.Parts))

	// Count node types
	var primitives, transforms, parts, joins int
	for _, node := range design.Graph.Nodes {
		switch node.Type {
		case graph.NodeTypePrimitive:
			primitives++
		case graph.NodeTypeTransform:
			transforms++
		case graph.NodeTypePart:
			parts++
		case graph.NodeTypeJoin:
			joins++
		}
	}

	fmt.Printf("  Primitives: %d\n", primitives)
	fmt.Printf("  Transforms: %d\n", transforms)
	fmt.Printf("  Parts: %d\n", parts)
	fmt.Printf("  Joins: %d\n", joins)

	// List all parts
	fmt.Println("\nParts in design:")
	for _, part := range design.Parts {
		fmt.Printf("  - %s (grain: %v, material: %s)\n",
			part.Name, part.Metadata.GrainAxis, part.Metadata.Material.Type)
	}

	// Show graph structure
	fmt.Println("\nGraph structure (simplified):")
	fmt.Println("  Primitives → Parts → Joins")
	fmt.Println("  (leg, apron, top primitives)")
	fmt.Println("  ↓")
	fmt.Println("  (4 leg parts, 4 apron parts, 1 top part)")
	fmt.Println("  ↓")
	fmt.Println("  (8 butt joints connecting legs to aprons)")

	// Export to Lisp representation
	fmt.Println("\nLisp representation (simplified):")
	fmt.Println("```lisp")
	fmt.Println("(defprimitive leg :cuboid [50 50 750])")
	fmt.Println("(defprimitive apron :cuboid [100 50 600])")
	fmt.Println("(defprimitive top :cuboid [600 600 25])")
	fmt.Println()
	fmt.Println("(define-part \"leg-front-left\"")
	fmt.Println("  :solids [leg]")
	fmt.Println("  :grain :z")
	fmt.Println("  :material \"oak\")")
	fmt.Println()
	fmt.Println("(butt-join")
	fmt.Println("  :part-a \"leg-front-left\"")
	fmt.Println("  :face-a 0")
	fmt.Println("  :part-b \"apron-front\"")
	fmt.Println("  :face-b 2")
	fmt.Println("  :clearance 0.2)")
	fmt.Println("```")
}