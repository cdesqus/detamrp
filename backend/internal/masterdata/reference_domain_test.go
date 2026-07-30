package masterdata

import "testing"

func TestCategoryInputNormalizesFields(t *testing.T) {
	input := CategoryInput{Code: " chemical ", Name: " Chemical ", Description: "  Controlled material "}
	if fields := input.NormalizeAndValidate(); len(fields) != 0 {
		t.Fatalf("unexpected validation fields: %#v", fields)
	}
	if input.Code != "CHEMICAL" || input.Name != "Chemical" || input.Description != "Controlled material" {
		t.Fatalf("category input was not normalized: %#v", input)
	}
}

func TestPackingInputRequiresCodeAndName(t *testing.T) {
	input := PackingInput{Description: " Carton "}
	fields := input.NormalizeAndValidate()
	if fields["code"] != "Code is required" || fields["name"] != "Name is required" {
		t.Fatalf("packing validation fields = %#v", fields)
	}
}

func TestPlantInputNormalizesAddressAndRequiresIdentity(t *testing.T) {
	input := PlantInput{Code: " plant-1 ", Name: " Main Plant ", Address: "  Jakarta "}
	if fields := input.NormalizeAndValidate(); len(fields) != 0 {
		t.Fatalf("unexpected validation fields: %#v", fields)
	}
	if input.Code != "PLANT-1" || input.Name != "Main Plant" || input.Address != "Jakarta" {
		t.Fatalf("plant input was not normalized: %#v", input)
	}
}
