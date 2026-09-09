package salesorder

import (
	"encoding/json"
	"fmt"

	"order-stock/backend/internal/bom"
)

// RequirementLine is calculated solely from the snapshot captured at submission.
type RequirementLine struct {
	LineID string                `json:"lineId"`
	Result bom.RequirementResult `json:"result"`
}

func CalculateRequirements(lines []Line) ([]RequirementLine, error) {
	results := make([]RequirementLine, 0, len(lines))
	for _, line := range lines {
		if len(line.Calculation) == 0 || string(line.Calculation) == "null" {
			return nil, fmt.Errorf("sales order line %s has no submitted calculation snapshot", line.ID)
		}
		var snapshot bom.Snapshot
		if err := json.Unmarshal(line.Calculation, &snapshot); err != nil {
			return nil, fmt.Errorf("invalid calculation snapshot for line %s: %w", line.ID, err)
		}
		result, err := bom.Calculate(snapshot, line.Quantity)
		if err != nil {
			return nil, fmt.Errorf("calculate line %s: %w", line.ID, err)
		}
		results = append(results, RequirementLine{LineID: line.ID.String(), Result: result})
	}
	return results, nil
}
