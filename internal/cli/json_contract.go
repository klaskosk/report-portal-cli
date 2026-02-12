package cli

import (
	"encoding/json"
	"fmt"
)

// ensureJSONObjectKeys converts value to a JSON object map and ensures required keys exist. Missing keys are added with
// null values to stabilize machine-readable JSON contracts. Values are preserved as raw JSON bytes to avoid numeric
// precision loss from float64 round-tripping.
func ensureJSONObjectKeys(value any, requiredKeys ...string) (map[string]json.RawMessage, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshaling json object: %w", err)
	}

	var object map[string]json.RawMessage

	err = json.Unmarshal(payload, &object)
	if err != nil {
		return nil, fmt.Errorf("unmarshaling json object: %w", err)
	}

	for _, key := range requiredKeys {
		if _, exists := object[key]; !exists {
			object[key] = json.RawMessage("null")
		}
	}

	return object, nil
}

// ensureJSONArrayObjectKeys converts values to a JSON array of object maps and ensures required keys exist in each
// object. Missing keys are added with null values. Values are preserved as raw JSON bytes to avoid numeric precision
// loss from float64 round-tripping.
func ensureJSONArrayObjectKeys(
	values any,
	requiredKeys ...string,
) ([]map[string]json.RawMessage, error) {
	payload, err := json.Marshal(values)
	if err != nil {
		return nil, fmt.Errorf("marshaling json array: %w", err)
	}

	var objects []map[string]json.RawMessage

	err = json.Unmarshal(payload, &objects)
	if err != nil {
		return nil, fmt.Errorf("unmarshaling json array: %w", err)
	}

	for index := range objects {
		for _, key := range requiredKeys {
			if _, exists := objects[index][key]; !exists {
				objects[index][key] = json.RawMessage("null")
			}
		}
	}

	return objects, nil
}

func printSingleResource(
	printer *Printer,
	value any,
	headers []string,
	row []string,
	requiredKeys []string,
	resourceLabel string,
) error {
	outputValue := value
	if printer.isJSON() {
		normalized, err := ensureJSONObjectKeys(value, requiredKeys...)
		if err != nil {
			printer.PrintError(err)

			return fmt.Errorf("normalizing %s json output: %w", resourceLabel, err)
		}

		outputValue = normalized
	}

	return printer.Print(outputValue, headers, [][]string{row})
}

func printResourceList(
	printer *Printer,
	values any,
	headers []string,
	rows [][]string,
	requiredKeys []string,
	resourceLabel string,
) error {
	outputValue := values
	if printer.isJSON() {
		normalized, err := ensureJSONArrayObjectKeys(values, requiredKeys...)
		if err != nil {
			printer.PrintError(err)

			return fmt.Errorf("normalizing %s list json output: %w", resourceLabel, err)
		}

		outputValue = normalized
	}

	return printer.Print(outputValue, headers, rows)
}
