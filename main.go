package main

import (
	"fmt"
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v2"
)

// Helper function to convert YAML to a map[string]interface{}
func toStringMapInterface(data interface{}) interface{} {
	switch v := data.(type) {
	case map[interface{}]interface{}:
		m := make(map[string]interface{})
		for key, value := range v {
			strKey := fmt.Sprintf("%v", key)
			m[strKey] = toStringMapInterface(value)
		}
		return m
	case []interface{}:
		for i, value := range v {
			v[i] = toStringMapInterface(value)
		}
		return v
	default:
		return data
	}
}

func readYAML(filePath string) (map[string]interface{}, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var result interface{}
	err = yaml.Unmarshal(data, &result)
	if err != nil {
		return nil, err
	}

	return toStringMapInterface(result).(map[string]interface{}), nil
}

// Function to write a map to a YAML file
func writeYAML(filePath string, data map[string]interface{}) error {
	out, err := yaml.Marshal(data)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filePath, out, 0644)
	if err != nil {
		return err
	}

	return nil
}

func extractExamples(data map[string]interface{}) map[string]interface{} {
	examples := make(map[string]interface{})
	for key, value := range data {
		switch typedValue := value.(type) {
		case map[string]interface{}:
			if key == "example" || key == "examples" || key == "x-example" {
				examples[key] = typedValue
				fmt.Printf("Found example: %s\n", key)
			} else {
				nestedExamples := extractExamples(typedValue)
				if len(nestedExamples) > 0 {
					examples[key] = nestedExamples
				}
			}
		case []interface{}:
			for i, item := range typedValue {
				if nestedMap, ok := item.(map[string]interface{}); ok {
					nestedExamples := extractExamples(nestedMap)
					if len(nestedExamples) > 0 {
						examples[fmt.Sprintf("%s[%d]", key, i)] = nestedExamples
					}
				}
			}
		case int, string, bool:
			if key == "example" || key == "examples" || key == "x-example" {
				examples[key] = typedValue
				fmt.Printf("Found example: %s\n", key)
			}
		default:
			fmt.Printf("Unexpected type for key %s: %T\n", key, typedValue)
		}
	}
	return examples
}

// Function to find matching paths in the original YAML
func findMatchingPaths(originalPaths, examplePaths map[string]interface{}) map[string]interface{} {
	matchingPaths := make(map[string]interface{})
	for path, examplePathItem := range examplePaths {
		if originalPathItem, exists := originalPaths[path]; exists {
			matchingPaths[path] = map[string]interface{}{
				"original": originalPathItem,
				"example":  examplePathItem,
			}
		}
	}
	return matchingPaths
}

// Insert examples specifically into paths and responses
func insertExamplesIntoPaths(original, examples map[string]interface{}, addMissing bool) {
	if originalPaths, ok := original["paths"].(map[string]interface{}); ok {
		if examplePaths, ok := examples["paths"].(map[string]interface{}); ok {
			matchingPaths := findMatchingPaths(originalPaths, examplePaths)
			for _, items := range matchingPaths {
				originalPathItem := items.(map[string]interface{})["original"].(map[string]interface{})
				examplePathItem := items.(map[string]interface{})["example"].(map[string]interface{})
				insertExamples(originalPathItem, examplePathItem, addMissing)
			}
		}
	}
}

// Insert examples into the original YAML structure
func insertExamples(original map[string]interface{}, examples map[string]interface{}, addMissing bool) {
	for key, value := range examples {
		if originalValue, exists := original[key]; exists {
			switch originalTyped := originalValue.(type) {
			case map[string]interface{}:
				if examplesTyped, ok := value.(map[string]interface{}); ok {
					insertExamples(originalTyped, examplesTyped, addMissing)
				}
			case []interface{}:
				for _, item := range originalTyped {
					if nestedMap, ok := item.(map[string]interface{}); ok {
						if nestedExamples, ok := value.(map[string]interface{}); ok {
							insertExamples(nestedMap, nestedExamples, addMissing)
						}
					}
				}
			default:
				original[key] = value
			}
		} else if addMissing {
			original[key] = value
		}
	}
}

func main() {
	fmt.Println("comparing start !")

	originalYAML, err := readYAML("example_original_openapi.yaml")
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	convertedYAML, err := readYAML("example_convert_openapi_json.yaml")
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	examples := extractExamples(convertedYAML)

	insertExamplesIntoPaths(originalYAML, examples, true)

	// Save the modified original YAML
	err = writeYAML("modified_original_openapi.yaml", originalYAML)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	fmt.Println("Examples extracted and inserted successfully.")
}
