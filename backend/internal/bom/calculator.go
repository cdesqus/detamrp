package bom

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"

	"github.com/shopspring/decimal"
)

// Node describes usage per one base unit of its parent, never per Kanban.
type Node struct {
	ItemID       string          `json:"itemId"`
	Kind         string          `json:"kind"`
	ItemCode     string          `json:"itemCode"`
	Name         string          `json:"name"`
	Unit         string          `json:"unit"`
	Discrete     bool            `json:"discrete"`
	Usage        decimal.Decimal `json:"usage"`
	BOMID        string          `json:"bomId,omitempty"`
	Revision     int             `json:"revision,omitempty"`
	UnitPrice    decimal.Decimal `json:"unitPrice"`
	Currency     string          `json:"currency,omitempty"`
	QtyPerKanban decimal.Decimal `json:"qtyPerKanban"`
	Children     []Node          `json:"children,omitempty"`
}
type Snapshot struct {
	Version int  `json:"version"`
	Root    Node `json:"root"`
}
type RequirementNode struct {
	Path       string          `json:"path"`
	ParentPath string          `json:"parentPath"`
	ItemID     string          `json:"itemId"`
	Kind       string          `json:"kind"`
	ItemCode   string          `json:"itemCode"`
	Name       string          `json:"name"`
	Unit       string          `json:"unit"`
	Usage      decimal.Decimal `json:"usage"`
	Quantity   decimal.Decimal `json:"quantity"`
	Terminal   bool            `json:"terminal"`
	Depth      int             `json:"depth"`
}
type MaterialRequirement struct {
	ItemID           string          `json:"itemId"`
	ItemCode         string          `json:"itemCode"`
	Name             string          `json:"name"`
	Unit             string          `json:"unit"`
	Quantity         decimal.Decimal `json:"quantity"`
	UnitPrice        decimal.Decimal `json:"unitPrice"`
	Currency         string          `json:"currency"`
	Value            decimal.Decimal `json:"value"`
	QtyPerKanban     decimal.Decimal `json:"qtyPerKanban"`
	KanbanEquivalent decimal.Decimal `json:"kanbanEquivalent"`
	PurchaseKanban   decimal.Decimal `json:"purchaseKanban"`
	Paths            []string        `json:"paths"`
}
type RequirementResult struct {
	Nodes        []RequirementNode          `json:"nodes"`
	Materials    []MaterialRequirement      `json:"materials"`
	Totals       map[string]decimal.Decimal `json:"totals"`
	CostComplete bool                       `json:"costComplete"`
}

var maxStoredDecimal = decimal.RequireFromString("99999999999999.999999")

func validQuantity(q decimal.Decimal) bool {
	return !q.IsNegative() && q.LessThanOrEqual(maxStoredDecimal) && q.Equal(q.Round(6))
}

func roundedStored(q decimal.Decimal) (decimal.Decimal, bool) {
	rounded := q.Round(6)
	return rounded, validQuantity(rounded)
}

// BuildSnapshot validates and deep-copies the graph so later master edits cannot change it.
func BuildSnapshot(root Node) (Snapshot, error) {
	if root.Kind != "FG" && root.Kind != "RAW_MATERIAL" {
		return Snapshot{}, fmt.Errorf("invalid output kind")
	}
	if err := validateNode(root, map[string]bool{}, map[string]Node{}, true, 0); err != nil {
		return Snapshot{}, err
	}
	data, err := json.Marshal(root)
	if err != nil {
		return Snapshot{}, err
	}
	var saved Node
	if err = json.Unmarshal(data, &saved); err != nil {
		return Snapshot{}, err
	}
	return Snapshot{Version: 1, Root: saved}, nil
}
func validateNode(n Node, stack map[string]bool, seen map[string]Node, isRoot bool, depth int) error {
	if depth > 64 {
		return fmt.Errorf("BOM exceeds 64 levels")
	}
	if n.ItemID == "" || n.ItemCode == "" || n.Unit == "" {
		return fmt.Errorf("item identity, code and unit are required")
	}
	if n.Kind != "FG" && n.Kind != "RAW_MATERIAL" {
		return fmt.Errorf("invalid item kind for %s", n.ItemCode)
	}
	if stack[n.ItemID] {
		return fmt.Errorf("circular BOM at %s", n.ItemCode)
	}
	if !n.Usage.IsPositive() || !validQuantity(n.Usage) {
		return fmt.Errorf("invalid usage for %s", n.ItemCode)
	}
	if n.Discrete && !n.Usage.Equal(n.Usage.Truncate(0)) {
		return fmt.Errorf("whole units required for %s", n.ItemCode)
	}
	if !validQuantity(n.UnitPrice) {
		return fmt.Errorf("invalid price for %s", n.ItemCode)
	}
	if !validQuantity(n.QtyPerKanban) {
		return fmt.Errorf("invalid Kanban factor for %s", n.ItemCode)
	}
	if !isRoot && n.Kind != "RAW_MATERIAL" {
		return fmt.Errorf("BOM components must be raw materials: %s", n.ItemCode)
	}
	if previous, exists := seen[n.ItemID]; exists {
		if previous.Kind != n.Kind || previous.ItemCode != n.ItemCode || previous.Name != n.Name || previous.Unit != n.Unit ||
			previous.Discrete != n.Discrete || !previous.UnitPrice.Equal(n.UnitPrice) || previous.Currency != n.Currency ||
			!previous.QtyPerKanban.Equal(n.QtyPerKanban) {
			return fmt.Errorf("inconsistent metadata for %s", n.ItemCode)
		}
	} else {
		seen[n.ItemID] = n
	}
	if len(n.Children) == 0 && n.Currency == "" {
		return fmt.Errorf("material currency missing for %s", n.ItemCode)
	}
	stack[n.ItemID] = true
	defer delete(stack, n.ItemID)
	for _, child := range n.Children {
		if err := validateNode(child, stack, seen, false, depth+1); err != nil {
			return err
		}
	}
	return nil
}

// Calculate uses only captured data. Zero is useful for fully delivered SO lines.
func Calculate(snapshot Snapshot, quantity decimal.Decimal) (RequirementResult, error) {
	result := RequirementResult{Nodes: []RequirementNode{}, Materials: []MaterialRequirement{}, Totals: map[string]decimal.Decimal{}, CostComplete: true}
	if snapshot.Version != 1 {
		return result, fmt.Errorf("unsupported calculation snapshot version")
	}
	if !validQuantity(quantity) {
		return result, fmt.Errorf("invalid requested quantity")
	}
	if err := validateNode(snapshot.Root, map[string]bool{}, map[string]Node{}, true, 0); err != nil {
		return result, err
	}
	if quantity.IsZero() {
		return result, nil
	}
	grouped := map[string]*MaterialRequirement{}
	var walk func(Node, decimal.Decimal, string, string, int) error
	walk = func(n Node, parent decimal.Decimal, path, parentPath string, depth int) error {
		exactQuantity := parent.Mul(n.Usage)
		q, stored := roundedStored(exactQuantity)
		if !stored {
			return fmt.Errorf("quantity exceeds range for %s", n.ItemCode)
		}
		if n.Discrete && !exactQuantity.Equal(exactQuantity.Truncate(0)) {
			return fmt.Errorf("whole units required for %s", n.ItemCode)
		}
		terminal := len(n.Children) == 0
		result.Nodes = append(result.Nodes, RequirementNode{Path: path, ParentPath: parentPath, ItemID: n.ItemID, Kind: n.Kind, ItemCode: n.ItemCode, Name: n.Name, Unit: n.Unit, Usage: n.Usage, Quantity: q, Terminal: terminal, Depth: depth})
		if terminal {
			key := n.ItemID + "|" + n.Unit + "|" + n.Currency + "|" + n.UnitPrice.String() + "|" + n.QtyPerKanban.String()
			row := grouped[key]
			if row == nil {
				row = &MaterialRequirement{ItemID: n.ItemID, ItemCode: n.ItemCode, Name: n.Name, Unit: n.Unit, UnitPrice: n.UnitPrice, Currency: n.Currency, QtyPerKanban: n.QtyPerKanban}
				grouped[key] = row
			}
			row.Quantity = row.Quantity.Add(exactQuantity)
			row.Paths = append(row.Paths, path)
			if n.UnitPrice.IsZero() {
				result.CostComplete = false
			}
		}
		for i, child := range n.Children {
			if err := walk(child, exactQuantity, path+"."+strconv.Itoa(i), path, depth+1); err != nil {
				return err
			}
		}
		return nil
	}
	if err := walk(snapshot.Root, quantity, "0", "", 0); err != nil {
		return result, err
	}
	keys := make([]string, 0, len(grouped))
	for key := range grouped {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		row := grouped[key]
		var stored bool
		row.Quantity, stored = roundedStored(row.Quantity)
		if !stored {
			return result, fmt.Errorf("aggregate quantity overflow for %s", row.ItemCode)
		}
		row.Value, stored = roundedStored(row.Quantity.Mul(row.UnitPrice))
		if !validQuantity(row.Value) {
			return result, fmt.Errorf("material value overflow for %s", row.ItemCode)
		}
		if row.QtyPerKanban.IsPositive() {
			row.KanbanEquivalent = row.Quantity.DivRound(row.QtyPerKanban, 6)
			if _, ok := roundedStored(row.KanbanEquivalent); !ok {
				return result, fmt.Errorf("Kanban equivalent overflow for %s", row.ItemCode)
			}
			whole, remainder := row.Quantity.QuoRem(row.QtyPerKanban, 0)
			row.PurchaseKanban = whole
			if !remainder.IsZero() {
				row.PurchaseKanban = row.PurchaseKanban.Add(decimal.NewFromInt(1))
			}
			if _, ok := roundedStored(row.PurchaseKanban); !ok {
				return result, fmt.Errorf("purchase Kanban overflow for %s", row.ItemCode)
			}
		}
		result.Materials = append(result.Materials, *row)
		result.Totals[row.Currency] = result.Totals[row.Currency].Add(row.Value)
		if !validQuantity(result.Totals[row.Currency]) {
			return result, fmt.Errorf("currency total overflow")
		}
	}
	return result, nil
}
